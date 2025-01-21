package xtremeconsole

import (
	command2 "github.com/globalxtreme/go-core/v2/console/command"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
	"time"
)

type BaseCommand interface {
	Command(cmd *cobra.Command)
	Handle()
}

func Commands(cobraCmd *cobra.Command, newCommands []BaseCommand) {
	addCommand(cobraCmd, &command2.DeleteLogFileCommand{})

	for _, newCommand := range newCommands {
		addCommand(cobraCmd, newCommand)
	}
}

func addCommand(cmd *cobra.Command, newCmd BaseCommand) {
	newCmd.Command(cmd)
}

func Schedules(callback func(*gocron.Scheduler)) {
	sch := gocron.NewScheduler(time.UTC)

	// Schedules
	addSchedule(sch.Every(1).Day().At("00:01"), &command2.DeleteLogFileCommand{})
	callback(sch)

	sch.StartBlocking()
}

func addSchedule(schedule *gocron.Scheduler, command BaseCommand) {
	schedule.Do(command.Handle)
}
