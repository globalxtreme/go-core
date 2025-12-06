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

type BaseCommandSchedule interface {
	Command(cmd *cobra.Command)
	Prepare() (cancel func())
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
	sch := gocron.NewScheduler(time.Local)

	// Schedules
	addSchedule(sch.Every(1).Day().At("00:01"), &command2.DeleteLogFileCommand{FromSchedule: true})
	callback(sch)

	sch.StartBlocking()
}

func addSchedule(schedule *gocron.Scheduler, command BaseCommandSchedule) {
	schedule.Do(func() {
		cancel := command.Prepare()
		defer cancel()

		command.Handle()
	})
}
