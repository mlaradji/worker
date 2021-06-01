package worker

import (
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
	Key        JobKey
	Command    string
	Args       []string
	JobStatus  pb.JobStatus
	ExitCode   int32
	CreatedAt  time.Time
	FinishedAt time.Time

	WaitGroup   *sync.WaitGroup // WaitGroup will block if and only if the job is currently running.
	StopChannel chan struct{}   // StopChannel can receive an empty struct, which would cause the job to stop if it's running.
}

// LogFilepath returns the path to the job's log file.
func (job *Job) LogFilepath() string {
	return filepath.Join(job.LogDirectory(), "output.log")
}

// LogDirectory returns the path to the directory containing the job's log file.
func (job *Job) LogDirectory() string {
	return filepath.Join("tmp", "jobs", job.Key.UserId, job.Key.JobId)
}

// NotRunning returns a channel that receives a value if and only if the job is not running.
func (job *Job) NotRunning() <-chan struct{} {
	notRunning := make(chan struct{})

	go func() {
		job.WaitGroup.Wait()
		close(notRunning)
	}()

	return notRunning
}

// Stop sends a signal to the StopChannel, which should trigger the job to stop. No error is returned if the job is not running.
func (job *Job) Stop() {
	logger := log.WithFields(log.Fields{"func": "Job.Stop", "jobKey": job.Key})

	select {
	case job.StopChannel <- struct{}{}:
		logger.Debug("stopped job")
	case <-job.NotRunning():
		logger.Debug("job was not stopped as it is not running")
	}
}

// Log follows content of job's log file and sends to the returned channel. The returned channel is only closed after the log file is completely read and the job is not running.
func (job *Job) Log() (<-chan []byte, error) {
	logger := log.WithFields(log.Fields{"func": "Job.Log", "jobKey": job.Key, "LogFilepath": job.LogFilepath()})

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
			case <-job.NotRunning():
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
func NewJob(userId string, command string, args []string) Job {
	jobId := uuid.New().String()
	return Job{
		Key:         JobKey{UserId: userId, JobId: jobId},
		Command:     command,
		Args:        args,
		JobStatus:   pb.JobStatus_CREATED,
		ExitCode:    -1,
		CreatedAt:   time.Now(),
		StopChannel: make(chan struct{}),
		WaitGroup:   &sync.WaitGroup{},
	}
}
