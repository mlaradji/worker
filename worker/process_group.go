package worker

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// ProcessGroupCommand groups the main process and descendants of an exec.Cmd run.
type ProcessGroupCommand struct {
	Cmd  *exec.Cmd
	Done chan struct{} // Done is a channel that is closed if and only if the process finished running or was stopped.

	mu          *sync.RWMutex   // mu controls access to `stopped`.
	stop        chan struct{}   // stop is a channel that can receive stop requests. It is initialized at process definition, and is closed after the process ends.
	stopSenders *sync.WaitGroup // stopSenders controls access to the stop channel. It is initialized at process definition, and the stop channel is not closed until after this wait group is done.
	stopped     bool            // stopped is true if and only if the process was killed because of a stop request.
}

// GetExitCode returns the exit code of the process. It is equal to -1 if the job is still running or was stopped.
func (group *ProcessGroupCommand) GetExitCode() int {
	return group.Cmd.ProcessState.ExitCode()
}

// GetStopped returns the value of `stopped` in a thread-safe way.
func (group *ProcessGroupCommand) GetStopped() bool {
	group.mu.RLock()
	defer group.mu.RUnlock()
	return group.stopped
}

// Start starts the command.
func (group *ProcessGroupCommand) Start() error {
	logger := log.WithFields(log.Fields{"func": "ProcessGroupCommand.Start"})

	err := group.Cmd.Start()
	if err != nil {
		logger.WithError(err).Error("unable to start command")
		return err
	}

	// wait for the process to end, and then close the done channel
	go func() {
		defer close(group.Done)

		err := group.Cmd.Wait()
		if err != nil {
			logger.WithError(err).Debug("the process has failed")
		}
	}()

	// close the stop channel after the job is done
	go func() {
		defer close(group.stop)

		// wait for the job to finish
		<-group.Done
		// wait for any sending goroutines to finish before closing the channel
		group.stopSenders.Wait()
	}()

	// monitor the stop channel for stop requests
	go func() {
		select {
		case <-group.Done: // the process ended; stop this goroutine
			return
		case <-group.stop:
			err := syscall.Kill(-group.Cmd.Process.Pid, syscall.SIGKILL)
			if err == nil {
				group.mu.Lock()
				defer group.mu.Unlock()
				group.stopped = true
				return
			}

			logger.WithError(err).Error("unable to kill process group")
		}
	}()

	return nil
}

// Stop stops the command if it is running.
func (group *ProcessGroupCommand) Stop() {
	// add this function to the wait group of stop senders
	// This accomplishes two things:
	//		1. If the process is currently running, but finishes before this Stop function finishes, the stop channel will only close after this function finishes.
	//		2. If the process just stopped, and it is waiting for other instances of this Stop function to finish, then WaitGroup.Add will block until the process status was updated. This means that the Done channel will not block and so we avoid sending data to a closed channel.
	group.stopSenders.Add(1)
	defer group.stopSenders.Done()

	select {
	case <-group.Done:
		log.WithField("func", "ProcessGroupCommand.Stop").Debug("job is not running")
		return
	case group.stop <- struct{}{}:
		log.WithField("func", "ProcessGroupCommand.Stop").Debug("sent a stop request")
		return
	}
}

// NewProcessGroupCommand returns a new ProcessGroupCommand that can execute `name` with `args`. The STDOUT and STDERR output of the process will be written to `stdoutLogWriter` and `stderrLogWriter`, respectively.
func NewProcessGroupCommand(name string, args []string, stdoutLogWriter io.Writer, stderrLogWriter io.Writer) *ProcessGroupCommand {
	// ?: Command might buffer output, which means the client would receive log data in large chunks. Is this ideal?

	cmd := exec.CommandContext(
		context.Background(),
		name,
		args...,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // make sure descendants are put in the same process group
	cmd.Stdout = stdoutLogWriter
	cmd.Stderr = stderrLogWriter

	return &ProcessGroupCommand{
		Cmd:         cmd,
		Done:        make(chan struct{}),
		mu:          &sync.RWMutex{},
		stop:        make(chan struct{}),
		stopSenders: &sync.WaitGroup{},
		stopped:     false,
	}
}
