package worker

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mlaradji/int-backend-mohamed/pb"
	log "github.com/sirupsen/logrus"
)

// JobKey is used as the key in the Job Store map.
type JobKey struct {
	JobId  string
	UserId string
}

// Job represents a single job with all of its related objects.
type Job struct {
	Key       JobKey
	Command   string
	Args      []string
	CreatedAt time.Time
	Done      chan struct{} // Done is a channel that's closed after the job process is done and the job is updated with the status.

	// these fields can be changed, and should only be accessed through the Get methods
	jobStatus  pb.JobStatus
	exitCode   int32
	finishedAt time.Time

	mu    *sync.RWMutex        // mu is a read-write mutex to synchronize job updates.
	group *ProcessGroupCommand // group is the process group command providing access to the executing command.
}

// GetJobStatus locks the job mutex for reading and returns the job's status.
func (job *Job) GetJobStatus() pb.JobStatus {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.jobStatus
}

// GetExitCode locks the job mutex for reading and returns the exit code.
func (job *Job) GetExitCode() int32 {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.exitCode
}

// GetFinishedAt locks the job mutex for reading and returns the time the job finished.
func (job *Job) GetFinishedAt() time.Time {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.finishedAt
}

// LogFilepath returns the path to the job's log file.
func (job *Job) LogFilepath() string {
	return filepath.Join(job.LogDirectory(), "output.log")
}

// LogDirectory returns the path to the directory containing the job's log file.
func (job *Job) LogDirectory() string {
	return filepath.Join("tmp", "jobs", job.Key.UserId, job.Key.JobId)
}

/* Start runs the job without blocking.*/
func (job *Job) Start() error {
	logger := log.WithFields(log.Fields{"func": "Job.Start", "jobKey": job.Key})

	// open the logFile for writing, and pass it to the process group command
	logFile, err := os.OpenFile(job.LogFilepath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger.WithError(err).Error("unable to open file for writing")
		return err
	}

	// start the process
	if err := job.group.Start(logFile, logFile); err != nil {
		logger.WithError(err).Error("unable to start process")
		return err
	}

	// update the job status to RUNNING
	job.mu.Lock()
	job.jobStatus = pb.JobStatus_RUNNING
	job.mu.Unlock()

	go func() {
		// close the logFile and the Done channel after the process is done
		defer close(job.Done)
		defer logFile.Close()

		// wait for the command to finish
		<-job.group.Done

		// update job status and exit code
		job.mu.Lock()
		defer job.mu.Unlock()

		job.finishedAt = job.group.GetDoneAt()
		if job.group.GetStopped() {
			job.jobStatus = pb.JobStatus_STOPPED
		} else if job.group.GetExitCode() != 0 {
			logger.WithError(err).Debug("process has failed")
			job.jobStatus = pb.JobStatus_FAILED
		} else {
			job.jobStatus = pb.JobStatus_SUCCEEDED
		}
		job.exitCode = int32(job.group.GetExitCode())
	}()

	return nil
}

// Stop sends a signal to the process group to trigger the job to stop. This method does not block.
func (job *Job) Stop() {
	go job.group.Stop()
}

// Log follows content of job's log file and sends to the returned channel. The returned channel is only closed after the log file is completely read and the job is not running.
func (job *Job) Log() (<-chan []byte, error) {
	logger := log.WithFields(log.Fields{"func": "Job.Log", "jobKey": job.Key, "logFilepath": job.LogFilepath()})

	followDone := make(chan struct{}) // this is closed when the log file has been completely read and the job is not running

	logChannel, err := TailFollowFile(followDone, job.LogFilepath())
	if err != nil {
		logger.WithError(err).Error("unable to tail logfile")
		return nil, err
	}

	// initialize output channel
	outputChan := make(chan []byte)

	// send lines to channel
	go func() {
		defer close(outputChan)

	ForLoop:
		for {
			select {
			case chunk, ok := <-logChannel:
				outputChan <- chunk
				if !ok {
					return
				}
			case <-job.group.Done:
				close(followDone)
				break ForLoop
			}
		}

		// send remaining contents
		for chunk := range logChannel {
			outputChan <- chunk
		}
	}()

	return outputChan, nil
}

// NewJob generates a new Job object with status CREATED and exit code -1.
func NewJob(userId string, command string, args []string) *Job {
	jobId := uuid.New().String()
	return &Job{
		Key:       JobKey{UserId: userId, JobId: jobId},
		Command:   command,
		Args:      args,
		CreatedAt: time.Now(),
		jobStatus: pb.JobStatus_CREATED,
		exitCode:  -1,
		mu:        &sync.RWMutex{},
		group:     NewProcessGroupCommand(command, args),
		Done:      make(chan struct{}),
	}
}
