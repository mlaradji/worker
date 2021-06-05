package worker

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/singleflight"

	log "github.com/sirupsen/logrus"
)

// ProcessGroupCommand groups the main process and descendants of an exec.Cmd run.
type ProcessGroupCommand struct {
	Cmd  *exec.Cmd
	Done chan struct{} // Done is a channel that is closed if and only if the process finished running or was stopped.

	isDone      bool          // isDone is true if and only if the job has finished. It is also true if and only if the stop channel is closed.
	mu          *sync.RWMutex // mu controls access to `stopped` and `doneAt`.
	stop        chan struct{} // stop is a channel that can receive stop requests. It is initialized at process definition, and is closed after the process ends.
	stopSenders *sync.WaitGroup
	stopMutex   *sync.Mutex         // stopMutex controls access to the stop channel and to `isDone`. It is initialized at process definition, and the stop channel is not closed until after locking this.
	group       *singleflight.Group // group ensures that only one stop command is running at a time.
	stopped     bool                // Stopped is true if and only if the process was killed because of a stop request.
	doneAt      time.Time           // doneAt is the time that the process stopped executing.
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

// GetDoneAt returns the value of `doneAt` in a thread-safe way.
func (group *ProcessGroupCommand) GetDoneAt() time.Time {
	group.mu.RLock()
	defer group.mu.RUnlock()
	return group.doneAt
}

// Start starts the command and logs its output to the attached files.
func (group *ProcessGroupCommand) Start(stdoutLogWriter io.Writer, stderrLogWriter io.Writer) error {
	logger := log.WithFields(log.Fields{"func": "ProcessGroupCommand.Start"})

	// attach logs
	group.Cmd.Stdout = stdoutLogWriter
	group.Cmd.Stderr = stderrLogWriter

	err := group.Cmd.Start()
	if err != nil {
		logger.WithError(err).Error("unable to start command")
		return err
	}

	// wait for the process to end, and then close the done channel
	go func() {
		// update doneAt and close done channel at end
		defer func() {
			close(group.Done)
			doneAt := time.Now()
			group.mu.Lock()
			group.doneAt = doneAt
			group.mu.Unlock()
		}()

		err := group.Cmd.Wait()
		if err != nil {
			logger.WithError(err).Debug("the process has failed")
		}
	}()

	// close the stop channel after the job is done
	go func() {
		// wait for the job to finish
		<-group.Done
		// wait for any sending goroutines to finish before closing the channel
		group.stopMutex.Lock()
		close(group.stop)
		group.isDone = true
		group.stopMutex.Unlock()
	}()

	// monitor the stop channel for stop requests
	go func() {
		select {
		case <-group.Done: // the process ended
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

// Stop stops the command if it is running. This method blocks until the signal is sent or the job is not running.
func (group *ProcessGroupCommand) Stop() bool {
	logger := log.WithField("func", "ProcessGroupCommand.Stop")

	_, _, shared := group.group.Do("stop", func() (interface{}, error) {
		// add this function to the wait group of stop senders and check if the job is done
		// This accomplishes two things:
		//		1. If the process is currently running, but finishes before this Stop function finishes, the stop channel will only close after this function finishes.
		//		2. If the process just stopped, and it is waiting for other instances of this Stop function to finish, then WaitGroup.Add will block until the process status was updated. This means that isDone will be true and so we avoid sending data to a closed channel.
		group.stopMutex.Lock()
		defer group.stopMutex.Unlock()

		if group.isDone {
			logger.Debug("job is has already finished")
			return struct{}{}, nil
		}

		select {
		case <-group.Done:
			break
		case group.stop <- struct{}{}:
			logger.Debug("sent a stop request")
			break
		}

		return struct{}{}, nil
	})

	if shared {
		logger.Debug("did not send a stop request as another request was already in progress")
	}

	return shared
}

// NewProcessGroupCommand returns a new ProcessGroupCommand that can execute `name` with `args`. The STDOUT and STDERR output of the process will be written to `stdoutLogWriter` and `stderrLogWriter`, respectively.
func NewProcessGroupCommand(name string, args []string) *ProcessGroupCommand {
	// ?: Command might buffer output, which means the client would receive log data in large chunks. Is this ideal?
	// ?: Loggers should only be attached at job start?

	cmd := exec.CommandContext(
		context.Background(),
		name,
		args...,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // make sure descendants are put in the same process group

	return &ProcessGroupCommand{
		Cmd:         cmd,
		Done:        make(chan struct{}),
		mu:          &sync.RWMutex{},
		stop:        make(chan struct{}),
		stopSenders: &sync.WaitGroup{},
		stopMutex:   &sync.Mutex{},
		stopped:     false,
		group:       &singleflight.Group{},
	}
}
