package main

import (
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	store := worker.NewJobStore()
	// // job, _ := store.AddJob("me", "sleep", []string{"5s"})
	// job, _ := store.AddJob("me", "python", []string{"script.py"})
	// store.StartJob(job)
	// log.Debug("IM HERE")
	// // exitChannel, _ := store.MakeExitChannel(job)
	// // <-exitChannel
	// time.Sleep(5 * time.Second)

	// // add a long running process that spawns multiple children
	// job, _ := store.AddJob("me", "watch", []string{"date", "&"})

	// // create an exit channel that will be signalled at process end
	// exitChannel, _ := store.MakeExitChannel(job)

	// store.StartJob(job)

	// // stop the job and wait for the exit code to be published
	// log.Info("HERE")
	// job.Stop()
	// log.Error("sent signal")
	// <-exitChannel

	// // load the job from store and check that the status changed to stop
	// job, _ = store.LoadJob(job.Key)

	// log.Info(job)

	// add a long running process that spawns multiple children
	job, _ := store.AddJob("me", "cat", []string{"testing"})

	// // create an exit channel that will be signalled at process end
	// exitChannel, _ := store.MakeExitChannel(job)

	store.StartJob(job)

	// // stop the job and wait for the exit code to be published
	// log.Info("HERE")
	// job.Stop()
	// log.Error("sent signal")
	// <-exitChannel

	// load the job from store and check that the status changed to stop
	job, _ = store.LoadJob(job.Key)

	log.Info(job)
}
