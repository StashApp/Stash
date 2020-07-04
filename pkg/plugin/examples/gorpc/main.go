// +build plugin_example

package main

import (
	"os"
	"time"

	exampleCommon "github.com/stashapp/stash/pkg/plugin/examples/common"

	"github.com/stashapp/stash/pkg/plugin/common"
	"github.com/stashapp/stash/pkg/plugin/common/log"
	"github.com/stashapp/stash/pkg/plugin/util"
)

type api struct {
	stopping bool
}

func (a *api) Stop(input struct{}, output *bool) error {
	log.Info("Stopping...")
	a.stopping = true
	*output = true
	return nil
}

func (a *api) Run(input common.PluginInput, output *common.PluginOutput) error {
	client := util.NewClient(input)

	modeArg := common.GetValue(input.Args, "mode")

	var err error
	if modeArg == nil || modeArg.String() == "add" {
		err = exampleCommon.AddTag(client)
	} else if modeArg.String() == "remove" {
		err = exampleCommon.RemoveTag(client)
	} else if modeArg.String() == "long" {
		err = a.doLongTask()
	} else if modeArg.String() == "indef" {
		err = a.doIndefiniteTask()
	}

	if err != nil {
		errStr := err.Error()
		*output = common.PluginOutput{
			Error: &errStr,
		}
		return nil
	}

	outputStr := "ok"
	*output = common.PluginOutput{
		Output: &outputStr,
	}

	return nil
}

func (a *api) doLongTask() error {
	const total = 100
	upTo := 0

	log.Info("Doing long task")
	for upTo < total {
		time.Sleep(time.Second)
		if a.stopping {
			return nil
		}

		log.Progress(float64(upTo) / float64(total))
		upTo++
	}

	return nil
}

func (a *api) doIndefiniteTask() error {
	log.Warn("Sleeping indefinitely")
	for {
		time.Sleep(time.Second)
		if a.stopping {
			return nil
		}
	}

	return nil
}

func main() {
	debug := false

	if len(os.Args) >= 2 && os.Args[1] == "debug" {
		debug = true
	}

	if debug {
		api := api{}
		output := common.PluginOutput{}
		err := api.Run(common.PluginInput{
			ServerConnection: common.StashServerConnection{
				Scheme: "http",
				Port:   9999,
			},
		}, &output)

		if err != nil {
			panic(err)
		}

		if output.Error != nil {
			panic(*output.Error)
		}

		return
	}

	err := common.ServePlugin(&api{})
	if err != nil {
		panic(err)
	}
}