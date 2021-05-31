package worker_test

import (
	"testing"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/worker"
	"github.com/stretchr/testify/require"
)

// TestJobStopped executes a long running process and stops it.
func TestJobStopped(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// add a long running process that spawns multiple children
	job, err := store.AddJob(userId, "watch", []string{"date", "&"})
	require.Nil(t, err)

	err = store.StartJob(job)
	require.Nil(t, err)

	// stop the job and wait for the job to end
	job.Stop()
	job.WaitGroup.Wait()

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.Nil(t, err)
	require.Equal(t, pb.JobStatus_STOPPED, job.JobStatus)
	require.NotEqual(t, 0, job.ExitCode)
}

// TestJobSucceeded executes a quick process that should be successful. It also stops the process after it ends.
func TestJobSucceeded(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	job, err := store.AddJob(userId, "echo", []string{"testing"})
	require.Nil(t, err)

	// start the job and wait for it to finish
	err = store.StartJob(job)
	require.Nil(t, err)
	job.WaitGroup.Wait()

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.Nil(t, err)
	require.Equal(t, pb.JobStatus_SUCCEEDED, job.JobStatus)
	require.Equal(t, int32(0), job.ExitCode)
}

// TestJobFailed executes a quick process that should fail.
func TestJobFailed(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// run a process that exits with code 12
	job, err := store.AddJob(userId, "sh", []string{"-c", "exit 12"})
	require.Nil(t, err)

	// start the job and wait for it to finish
	err = store.StartJob(job)
	require.Nil(t, err)
	job.WaitGroup.Wait()

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.Nil(t, err)
	require.Equal(t, pb.JobStatus_FAILED, job.JobStatus)
	require.Equal(t, int32(12), job.ExitCode)
}

// TestJobMultiStop executes a quick process that will be stopped after it ends, and a slow process that will be stopped twice quickly.
func TestJobMultiStop(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// quick process

	fastJob, err := store.AddJob(userId, "echo", []string{"testing"})
	require.Nil(t, err)

	// start the job and wait for it to finish
	err = store.StartJob(fastJob)
	require.Nil(t, err)
	fastJob.WaitGroup.Wait()

	// stop the job after the process ended
	// this should not block
	fastJob.Stop()

	// slow process

	// add a long running process that spawns multiple children
	slowJob, err := store.AddJob(userId, "watch", []string{"date", "&"})
	require.Nil(t, err)

	err = store.StartJob(slowJob)
	require.Nil(t, err)

	// stop the job multiple times and wait for the job to end
	slowJob.Stop()
	slowJob.Stop()
	slowJob.Stop()
	slowJob.WaitGroup.Wait()
}