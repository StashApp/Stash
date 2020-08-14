package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"sync"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/plugin/common"
)

type rawTaskBuilder struct{}

func (*rawTaskBuilder) build(task pluginTask) Task {
	return &rawPluginTask{
		pluginTask: task,
	}
}

type rawPluginTask struct {
	pluginTask

	started   bool
	waitGroup sync.WaitGroup
	cmd       *exec.Cmd
	done      chan bool
}

func (t *rawPluginTask) Start() error {
	if t.started {
		return errors.New("task already started")
	}

	command := t.plugin.getExecCommand(t.operation)
	if len(command) == 0 {
		return fmt.Errorf("empty exec value in operation %s", t.operation.Name)
	}

	cmd := exec.Command(command[0], command[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error getting plugin process stdin: %s", err.Error())
	}

	go func() {
		defer stdin.Close()

		input := t.buildPluginInput()
		inBytes, _ := json.Marshal(input)
		io.WriteString(stdin, string(inBytes))
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("Plugin stderr not available: " + err.Error())
	}

	stdout, err := cmd.StdoutPipe()
	if nil != err {
		logger.Error("Plugin stdout not available: " + err.Error())
	}

	t.waitGroup.Add(1)
	t.done = make(chan bool, 1)
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("Error running plugin: %s", err.Error())
	}

	go t.handlePluginStderr(stderr)
	t.cmd = cmd

	// send the stdout to the plugin output
	go func() {
		defer t.waitGroup.Done()
		defer close(t.done)
		stdoutData, _ := ioutil.ReadAll(stdout)
		stdoutString := string(stdoutData)

		output := t.getOutput(stdoutString)

		err := cmd.Wait()
		if err != nil && output.Error == nil {
			errStr := err.Error()
			output.Error = &errStr
		}

		t.result = &output
	}()

	t.started = true
	return nil
}

func (t *rawPluginTask) getOutput(output string) common.PluginOutput {
	// try to parse the output as a PluginOutput json. If it fails just
	// get the raw output
	ret := common.PluginOutput{}
	decodeErr := json.Unmarshal([]byte(output), &ret)

	if decodeErr != nil {
		ret.Output = &output
	}

	return ret
}

func (t *rawPluginTask) Wait() {
	t.waitGroup.Wait()
}

func (t *rawPluginTask) Stop() error {
	if t.cmd == nil {
		return nil
	}

	return t.cmd.Process.Kill()
}
