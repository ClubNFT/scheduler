package main

import (
	"io"
	"log"
	"time"

	"github.com/ClubNFT/scheduler"
	"github.com/ClubNFT/scheduler/storage"
)

func TaskWithoutArgs() {
	log.Println("TaskWithoutArgs is executed")
}

func TaskWithArgs(message string) {
	log.Println("TaskWithArgs is executed. message:", message)
}

func main() {
	storage, err := storage.NewPostgresStorage(
		storage.PostgresDBConfig{
			DbURL: "postgresql://<user>:<password>@localhost:5432/<db>?sslmode=disable",
		},
	)
	if err != nil {
		log.Fatalf("Couldn't create scheduler storage : %v", err)
	}

	stubStorage := map[string]interface{}{
		"main.TaskWithArgs":    TaskWithArgs,
		"main.TaskWithoutArgs": TaskWithoutArgs,
	}

	s := scheduler.New(storage, stubStorage)

	s.Start()

	go func(s scheduler.Scheduler, store io.Closer) {
		time.Sleep(time.Minute * 5)
		// store.Close()
		s.Stop()
	}(s, storage)

	// Start a task without arguments and execute only once
	if _, err := s.RunAfter(60*time.Second, TaskWithoutArgs); err != nil {
		log.Fatal(err)
	}

	// Start a task with arguments and execute only once
	if _, err := s.RunAfter(30*time.Second, TaskWithArgs, "reportId"); err != nil {
		log.Fatal(err)
	}

	// Start a task with arguments periodically
	if _, err := s.RunEvery(5*time.Second, TaskWithArgs, "Hello from recurring task 1"); err != nil {
		log.Fatal(err)
	}

	// Start the same task as above with a different argument
	if _, err := s.RunEvery(10*time.Second, TaskWithArgs, "Hello from recurring task 2"); err != nil {
		log.Fatal(err)
	}
	s.Wait()
}
