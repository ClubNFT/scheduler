// Package scheduler is a small library that you can use within your application that enables you to execute callbacks (goroutines) after a pre-defined amount of time.
// GTS also provides task storage which is used to invoke callbacks for tasks which couldn’t be executed during down-time as well
// as maintaining a history of the callbacks that got executed.
package scheduler

import (
	"fmt"
	"github.com/ClubNFT/scheduler/config"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClubNFT/scheduler/storage"
	"github.com/ClubNFT/scheduler/task"
)

// Scheduler is used to schedule tasks. It holds information about those tasks
// including metadata such as argument types and schedule times
type Scheduler struct {
	stopChan    chan bool
	tasks       map[task.ID]*task.Task
	taskStore   storeBridge
	funcManager config.FunctionManager
}

// New will return a new instance of the Scheduler struct.
func New(store storage.TaskStore, stubStorage config.StubMapping) Scheduler {
	funcManager := *config.NewFunctionManager(stubStorage)
	return Scheduler{
		stopChan: make(chan bool),
		tasks:    make(map[task.ID]*task.Task),
		taskStore: storeBridge{
			store:       store,
			funcManager: funcManager,
		},
		funcManager: funcManager,
	}
}

// RunAt will schedule function to be executed once at the given time.
func (scheduler *Scheduler) RunAt(time time.Time, function task.Function, params ...string) (task.ID, error) {
	meta, err := task.Translate(function)
	if err != nil {
		return "", err
	}
	task := task.New(meta, params, scheduler.funcManager)

	task.NextRun = time

	scheduler.registerTask(task)
	err = scheduler.Refresh()
	if err != nil {
		return "", err
	}
	return task.Hash(), nil
}

// RunAfter executes function once after a specific duration has elapsed.
func (scheduler *Scheduler) RunAfter(duration time.Duration, function task.Function, params ...string) (task.ID, error) {
	return scheduler.RunAt(time.Now().Add(duration), function, params...)
}

// RunEvery will schedule function to be executed every time the duration has elapsed.
func (scheduler *Scheduler) RunEvery(duration time.Duration, function task.Function, params ...string) (task.ID, error) {
	meta, err := task.Translate(function)
	if err != nil {
		return "", err
	}
	task := task.New(meta, params, scheduler.funcManager)

	task.IsRecurring = true
	task.Duration = duration
	task.NextRun = time.Now().Add(duration)

	scheduler.registerTask(task)
	err = scheduler.Refresh()
	if err != nil {
		return "", err
	}

	return task.Hash(), nil
}

// Start will run the scheduler's timer and will trigger the execution
// of tasks depending on their schedule.
func (scheduler *Scheduler) Start() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Populate tasks from storage
	err := scheduler.Refresh()
	if err != nil {
		return err
	}

	scheduler.runPending()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				scheduler.runPending()
			case <-sigChan:
				scheduler.stopChan <- true
			case <-scheduler.stopChan:
				close(scheduler.stopChan)
			}
		}
	}()

	return nil
}

func (scheduler *Scheduler) Refresh() error {
	// Populate tasks from storage
	if err := scheduler.populateTasks(); err != nil {
		return err
	}
	if err := scheduler.persistRegisteredTasks(); err != nil {
		return err
	}
	return nil
}

// Stop will put the scheduler to halt
func (scheduler *Scheduler) Stop() {
	scheduler.taskStore.store.Close()
	scheduler.stopChan <- true
}

// Wait is a convenience function for blocking until the scheduler is stopped.
func (scheduler *Scheduler) Wait() {
	<-scheduler.stopChan
}

// Cancel is used to cancel the planned execution of a specific task using it's ID.
// The ID is returned when the task was scheduled using RunAt, RunAfter or RunEvery
func (scheduler *Scheduler) Cancel(taskID task.ID) error {
	task, found := scheduler.tasks[taskID]
	if !found {
		return fmt.Errorf("Task not found")
	}

	_ = scheduler.taskStore.Remove(task)
	delete(scheduler.tasks, taskID)
	return nil
}

// Clear will cancel the execution and clear all registered tasks.
func (scheduler *Scheduler) Clear() {
	for taskID, currentTask := range scheduler.tasks {
		_ = scheduler.taskStore.Remove(currentTask)
		delete(scheduler.tasks, taskID)
	}
}

func (scheduler *Scheduler) populateTasks() error {
	tasks, err := scheduler.taskStore.Fetch()
	if err != nil {
		return err
	}

	for _, dbTask := range tasks {
		//// If we can't find the function, it's been changed/removed by user
		//exists := scheduler.funcRegistry.Exists(dbTask.Func.Name)
		//if !exists {
		//	log.Printf("%s was not found, it will be removed\n", dbTask.Func.Name)
		//	_ = scheduler.taskStore.Remove(dbTask)
		//	continue
		//}

		// If the task instance is still registered with the same computed hash then move on.
		// Otherwise, one of the attributes changed and therefore, the task instance should
		// be added to the list of tasks to be executed with the stored params
		registeredTask, ok := scheduler.tasks[dbTask.Hash()]
		if !ok {
			log.Printf("Detected a change in attributes of one of the instances of task %s, \n",
				dbTask.Func.Name)
			//dbTask.Func, _ = scheduler.funcRegistry.Get(dbTask.Func.Name)
			registeredTask = dbTask
			scheduler.tasks[dbTask.Hash()] = registeredTask
		}

		// Duration may have changed for recurring tasks
		if dbTask.IsRecurring && registeredTask.Duration != dbTask.Duration {
			// Reschedule NextRun based on dbTask.LastRun + registeredTask.Duration
			registeredTask.NextRun = dbTask.LastRun.Add(registeredTask.Duration)
		}
	}
	return nil
}

func (scheduler *Scheduler) persistRegisteredTasks() error {
	for _, task := range scheduler.tasks {
		err := scheduler.taskStore.Add(task)
		if err != nil {
			return err
		}
	}
	return nil
}

func (scheduler *Scheduler) runPending() {
	for _, task := range scheduler.tasks {
		if task.IsDue() {

			// Reschedule task first to prevent running the task
			// again in case the execution time takes more than the
			// task's duration value.
			task.ScheduleNextRun()

			go task.Run()

			if !task.IsRecurring {
				_ = scheduler.taskStore.Remove(task)
				delete(scheduler.tasks, task.Hash())
			} else {
				_ = scheduler.taskStore.Update(task)
			}
		}
	}
}

func (scheduler *Scheduler) registerTask(task *task.Task) {
	scheduler.tasks[task.Hash()] = task
}
