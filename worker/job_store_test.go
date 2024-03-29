package worker_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/worker"
	"github.com/stretchr/testify/require"
)

const echoLoop = `#!/bin/sh

for i in {1..4}
do
  echo "Command no. $i"
  sleep 0.2
done

>&2 echo "Error 1"

for i in {5..7}
do
  echo "Command no. $i"
  sleep 0.2
done

>&2 echo "Error 2"

for i in {8..10}
do
  echo "Command no. $i"
  sleep 0.2
done`

// TestJobStopped executes a long running process and stops it.
func TestJobStopped(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// add a long running process that spawns multiple children
	job, err := store.AddJob(userId, "watch", []string{"date", "&"})
	require.NoError(t, err)

	err = job.Start()
	require.NoError(t, err)

	// stop the job and wait for the job to end
	job.Stop()
	<-job.Done

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_STOPPED, job.GetJobStatus())
	require.NotEqual(t, 0, job.GetExitCode())
}

// TestJobSucceeded executes a quick process that should be successful. It also stops the process after it ends.
func TestJobSucceeded(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	job, err := store.AddJob(userId, "echo", []string{"testing"})
	require.NoError(t, err)

	// start the job and wait for it to finish
	err = job.Start()
	require.NoError(t, err)
	<-job.Done

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_SUCCEEDED, job.GetJobStatus())
	require.Equal(t, int32(0), job.GetExitCode())
}

// TestJobFailed executes a quick process that should fail.
func TestJobFailed(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// run a process that exits with code 12
	job, err := store.AddJob(userId, "sh", []string{"-c", "exit 12"})
	require.NoError(t, err)

	// start the job and wait for it to finish
	err = job.Start()
	require.NoError(t, err)
	<-job.Done

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_FAILED, job.GetJobStatus())
	require.Equal(t, int32(12), job.GetExitCode())
}

// TestJobStartAfterLoad adds a new job to the store, loads it and then runs it.
func TestJobStartAfterLoad(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	job, err := store.AddJob(userId, "echo", []string{"testing"})
	require.NoError(t, err)

	// load and start
	loadedJob1, err := store.LoadJob(job.Key)
	require.NoError(t, err)

	loadedJob1.Start()
	<-loadedJob1.Done

	// load and check contents
	loadedJob2, err := store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_SUCCEEDED, loadedJob2.GetJobStatus())
	require.Equal(t, int32(0), loadedJob2.GetExitCode())
}

// TestJobStopAfterDone executes a quick process that will be stopped after it ends.
func TestJobStopAfterDone(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// quick process
	job, err := store.AddJob(userId, "echo", []string{"testing"})
	require.NoError(t, err)

	// start the job and wait for it to finish
	err = job.Start()
	require.NoError(t, err)
	<-job.Done

	// stop the job after the process ended
	// this should not block
	job.Stop()

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_SUCCEEDED, job.GetJobStatus())
	require.Equal(t, int32(0), job.GetExitCode())
}

// TestJobMultiStop executes a slow process that will be stopped multiple times quickly.
func TestJobMultiStop(t *testing.T) {
	t.Parallel()

	userId := "me"
	store := worker.NewJobStore()

	// add a long running process that spawns multiple children
	job, err := store.AddJob(userId, "watch", []string{"date", "&"})
	require.NoError(t, err)

	err = job.Start()
	require.NoError(t, err)

	// stop the job multiple times and wait for the job to end
	job.Stop()
	job.Stop()
	job.Stop()
	<-job.Done

	// load the job from store and check that the job information is correct
	job, err = store.LoadJob(job.Key)
	require.NoError(t, err)
	require.Equal(t, pb.JobStatus_STOPPED, job.GetJobStatus())
	require.NotEqual(t, 0, job.GetExitCode())
}

// TestJobFollowLogShort executes a quick process that will be stopped after it ends.
func TestJobFollowLogShort(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	userId := "me"
	store := worker.NewJobStore()

	echoBytes := []byte("this is a multiline test\nwe should get this too\n")
	expectedOutput := append(echoBytes, []byte("\n")...) // echo will emit an extra newline char

	job, err := store.AddJob(userId, "echo", []string{string(echoBytes)})
	require.NoError(t, err)

	// start the job and wait for it to finish
	err = job.Start()
	require.NoError(t, err)

	// get log channel
	outputChan, err := job.Log(ctx)
	require.NoError(t, err)

	actualOutput := []byte{}

	for line := range outputChan {
		actualOutput = append(actualOutput, line...)
	}
	require.Equal(t, expectedOutput, actualOutput, "expectedOutput", string(expectedOutput), "actualOutput", string(actualOutput))
}

// TestJobFollowLogLong executes a long process and checks that the log output is as expected.
func TestJobFollowLogLong(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	userId := "me"
	store := worker.NewJobStore()

	expectedOutput := []byte{}
	for i := 1; i < 5; i++ {
		expectedOutput = append(expectedOutput, []byte(fmt.Sprintf("Command no. %d\n", i))...) // echo will emit an extra newline char
	}
	expectedOutput = append(expectedOutput, []byte("Error 1\n")...)
	for i := 5; i < 8; i++ {
		expectedOutput = append(expectedOutput, []byte(fmt.Sprintf("Command no. %d\n", i))...) // echo will emit an extra newline char
	}
	expectedOutput = append(expectedOutput, []byte("Error 2\n")...)
	for i := 8; i < 11; i++ {
		expectedOutput = append(expectedOutput, []byte(fmt.Sprintf("Command no. %d\n", i))...) // echo will emit an extra newline char
	}

	job, err := store.AddJob(userId, "sh", []string{"-c", echoLoop})
	require.NoError(t, err)

	// start the job and wait for it to finish
	err = job.Start()
	require.NoError(t, err)

	// get log channel
	outputChan, err := job.Log(ctx)
	require.NoError(t, err)

	actualOutput := []byte{}

	for line := range outputChan {
		actualOutput = append(actualOutput, line...)
	}
	require.Equal(t, expectedOutput, actualOutput, "expectedOutput", string(expectedOutput), "actualOutput", string(actualOutput))
}

// TODO: Test a process that simultaneously outputs to both stdout and stderr. If the process attempts to write to both at the same time, they may be combined in a non-meaningful way (e.g. "HeErrorllo" instead of "Hello\nError\n").
