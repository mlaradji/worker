package worker

import (
	"errors"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	ErrJobDoesNotExist = errors.New("the job id and user id combination does not exist")
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
func (store *JobStore) AddJob(userId string, command string, args []string) (*Job, error) {
	job := NewJob(userId, command, args)
	logger := log.WithFields(log.Fields{"func": "JobStore.AddJob", "jobKey": job.Key})

	// create the log's directory if it doesn't already exist
	err := os.MkdirAll(job.LogDirectory(), os.ModePerm)
	if err != nil {
		logger.WithError(err).Error("unable to create log file directory")
		return nil, err
	}

	// add the job to the store
	_, loaded := store.Job.LoadOrStore(job.Key, job)
	if loaded {
		err := errors.New("the job id and user id combination already exists")
		logger.WithError(err).Error("unable to add job")
		return nil, err
	}

	return job, nil
}

// LoadJob loads a job from the store, and returns an error if the job does not exist or is invalid.
func (store *JobStore) LoadJob(jobKey JobKey) (*Job, error) {
	jobInterface, ok := store.Job.Load(jobKey)
	if !ok {
		return nil, ErrJobDoesNotExist
	}

	job, valid := jobInterface.(*Job)
	if !valid {
		return nil, errors.New("job was found but it is invalid")
	}

	return job, nil
}
