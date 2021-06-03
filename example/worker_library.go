package main

import (
	"os"
	"time"

	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
)

func main() {
	userId := "me"
	store := worker.NewJobStore()

	// run `watch -n 1 date`
	job, err := store.AddJob(userId, "watch", []string{"-n", "1", "date"})
	if err != nil {
		log.Fatal("unable to add job")
	}

	// start the job and wait for it to finish
	err = job.Start()
	if err != nil {
		log.Fatal("unable to start job")
	}

	// get log channel
	outputChan, err := job.Log()
	if err != nil {
		log.Fatal("unable to follow job logs")
	}

	wait := make(chan struct{})
	go func() {
		for chunk := range outputChan {
			os.Stdout.Write(chunk)
		}
		close(wait)
	}()

	// stop the job after 3 seconds
	time.Sleep(3 * time.Second)
	job.Stop() // stop the job. non-blocking

	<-job.Done // wait until the job is done
	<-wait

	log.Info("job is no longer running")
}
