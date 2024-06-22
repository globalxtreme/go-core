package command

import (
	xtremecore "github.com/globalxtreme/go-core"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"time"
)

type DeleteLogFileCommand struct{}

func (command *DeleteLogFileCommand) Command(cmd *cobra.Command) {
	cmd.AddCommand(&cobra.Command{
		Use:  "delete-log-file",
		Long: "Delete log file command",
		Run: func(cmd *cobra.Command, args []string) {
			xtremecore.InitDevMode()

			command.Handle()
		},
	})
}

func (command *DeleteLogFileCommand) Handle() {
	storageDir := os.Getenv("STORAGE_DIR") + "/logs/"

	logDays := 14
	logDaysEnv := os.Getenv("LOG_DAYS")
	if len(logDaysEnv) > 0 {
		logDays, _ = strconv.Atoi(logDaysEnv)
	}

	filename := time.Now().AddDate(0, 0, -logDays).Format("2006-01-02") + ".log"
	fullPath := storageDir + filename
	xtremecore.Debug(fullPath)

	_, err := os.Stat(fullPath)
	if err == nil {
		err := os.Remove(fullPath)
		if err != nil {
			xtremecore.Error(err)
		}
	}
}
