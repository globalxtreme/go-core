package xtremequeue

import (
	"fmt"
	"github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gocraft/work"
	"gorm.io/gorm/utils"
	"log"
	"os"
	"os/signal"
	"strings"
)

type JobConf struct {
	Context     interface{}
	JobFunc     interface{}
	QueueName   string
	JobName     string
	Priority    uint
	Concurrency uint
}

type Queue struct {
	Names string
}

func (q Queue) Work(workers []JobConf) {
	var names []string

	if len(q.Names) > 0 {
		q.Names = strings.ReplaceAll(q.Names, " ", "")
		names = strings.Split(q.Names, ",")
	}

	xtremepkg.InitRedisPool()

	var pools []*work.WorkerPool

	for _, worker := range workers {
		if len(names) > 0 {
			if !utils.Contains(names, worker.QueueName) {
				continue
			}
		}

		pool := work.NewWorkerPool(worker.Context, worker.Concurrency, worker.QueueName, xtremepkg.RedisPool)
		pool.JobWithOptions(worker.JobName, work.JobOptions{
			Priority: worker.Priority,
		}, worker.JobFunc)

		pool.Start()

		pools = append(pools, pool)

		log.Println(fmt.Sprintf("%s:%s", worker.QueueName, worker.JobName))
	}

	// Wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan

	for _, pool := range pools {
		pool.Stop()
	}

	fmt.Println("All worker is done!")
}
