package worker

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// ProcessGroupCommand groups the main process and descendants of an exec.Cmd run.
type ProcessGroupCommand struct {
	Cmd  *exec.Cmd
	Stop <-chan struct{} // Stop is a channel that receives stop requests.
}

// NewProcessGroupCommand returns a new ProcessGroupCommand that executes `name` with `args`. The channel `stop` receives stop requests. The STDOUT and STDERR logs are written to `logFile`.
func NewProcessGroupCommand(stop <-chan struct{}, logFile *os.File, name string, args []string) *ProcessGroupCommand {
	// ?: Command might buffer output, which means the client would receive log data in large chunks. Is this ideal?

	cmd := exec.CommandContext(
		context.Background(),
		name,
		args...,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // make sure descendants are put in the same process group
	cmd.Stdin = logFile
	cmd.Stdout = logFile

	return &ProcessGroupCommand{
		Cmd:  cmd,
		Stop: stop,
	}
}

// Start starts the command with PGID set.
func (group *ProcessGroupCommand) Start() error {
	err := group.Cmd.Start()
	return err
}

// Wait waits for the command to end while also listening for stop requests, returning `true` if the process stopped because of a stop request. Error is not `nil` if and only if the process was stopped or it finished with a non-zero exit code.
func (group *ProcessGroupCommand) Wait() (bool, error) {
	logger := log.WithFields(log.Fields{"func": "ProcessGroupCommand.Wait", "pid": group.Cmd.Process.Pid})

	stopped := make(chan bool) // if true, the process exited because of a stop request
	defer close(stopped)

	end := make(chan struct{}) // this will be closed when process exits normally

	go func() {
		select {
		case <-group.Stop:
			err := syscall.Kill(-group.Cmd.Process.Pid, syscall.SIGKILL)
			if err != nil {
				logger.WithError(err).Error("unable to kill process group")
			}
			stopped <- true
			return
		case <-end: // the process ended; stop this goroutine
			stopped <- false
			return
		}
	}()

	err := group.Cmd.Wait()
	close(end)
	return <-stopped, err
}
