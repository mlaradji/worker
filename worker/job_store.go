package worker

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/mlaradji/int-backend-mohamed/pb"
	log "github.com/sirupsen/logrus"
)

var (
	ErrJobDoesNotExist = errors.New("the job id and user id combination does not exist")
	NEWLINE            = []byte("\n")
)

// JobStore stores Job objects, keyed by JobKey (jobId+userId).
type JobStore struct {
	Job *sync.Map // Job is a thread-safe `map[JobKey]Job`.
}

// NewStore initializes a new job store.
func NewJobStore() *JobStore {
	return &JobStore{Job: &sync.Map{}}
}

// AddJob initializes a new job, creates log directories for it and adds it to the store.
func (store *JobStore) AddJob(userId string, command string, args []string) (Job, error) {
	job := NewJob(userId, command, args)
	logger := log.WithFields(log.Fields{"func": "JobStore.AddJob", "jobKey": job.Key})

	// create the log's directory if it doesn't already exist
	err := os.MkdirAll(job.LogDirectory(), os.ModePerm)
	if err != nil {
		logger.WithError(err).Error("unable to create log file directory")
		return Job{}, err
	}

	// add the job to the store
	_, loaded := store.Job.LoadOrStore(job.Key, job)
	if loaded {
		err := errors.New("the job id and user id combination already exists")
		logger.WithError(err).Error("unable to add job")
		return Job{}, err
	}

	return job, nil
}

// LoadJob loads a job from the store, and returns an error if the job does not exist or is invalid.
func (store *JobStore) LoadJob(jobKey JobKey) (Job, error) {
	jobInterface, ok := store.Job.Load(jobKey)
	if !ok {
		return Job{}, ErrJobDoesNotExist
	}

	job, valid := jobInterface.(Job)
	if !valid {
		return Job{}, errors.New("job was found but it is invalid")
	}

	return job, nil
}

/* ExecuteJob starts an already added job. */
func (store *JobStore) StartJob(job Job) error {
	logger := log.WithFields(log.Fields{"func": "JobStore.StartJob", "jobKey": job.Key})

	// open the logFile for writing, and pass it to the process group command
	logFile, err := os.OpenFile(job.LogFilepath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger.WithError(err).Error("unable to open file for writing")
		return err
	}
	group := NewProcessGroupCommand(job.StopChannel, logFile, job.Command, job.Args)

	job.WaitGroup.Add(1) // block goroutines waiting for the job to finish

	// start the process
	if err := group.Start(); err != nil {
		logger.WithError(err).Error("unable to start process")
		return err
	}

	// update the job status to RUNNING
	job.JobStatus = pb.JobStatus_RUNNING
	store.Job.Store(job.Key, job)

	go func() {
		// close the logFile after the process is done
		defer logFile.Close()

		// Wait for the command to finish
		stopped, err := group.Wait()
		job.FinishedAt = time.Now()

		// update job status and exit code
		if stopped {
			job.JobStatus = pb.JobStatus_STOPPED
		} else if err != nil {
			logger.WithError(err).Error("process has failed")
			job.JobStatus = pb.JobStatus_FAILED
		} else {
			job.JobStatus = pb.JobStatus_SUCCEEDED
		}
		exitCode := int32(group.Cmd.ProcessState.ExitCode())
		job.ExitCode = exitCode

		// save job
		store.Job.Store(job.Key, job)

		job.WaitGroup.Done() // release goroutines waiting for the job to finish
	}()

	return nil
}

// JobFollowLog follows content of job's log file and sends to the returned channel. The returned channel receives data or blocks until the log file is completely read and the job is not running.
func (store *JobStore) JobFollowLog(job Job) (<-chan []byte, error) {
	logger := log.WithFields(log.Fields{"func": "JobStore.JobFollowLog", "jobKey": job.Key, "LogFilepath": job.LogFilepath()})

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
