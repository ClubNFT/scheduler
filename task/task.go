package task

import (
	"crypto/sha1"
	"fmt"
	"github.com/ClubNFT/scheduler/config"
	"io"
	"log"
	"time"
)

// ID is returned upon scheduling a task to be executed
type ID string

// Schedule holds information about the execution times of a specific task
type Schedule struct {
	IsRecurring bool
	LastRun     time.Time
	NextRun     time.Time
	Duration    time.Duration
}

// Task holds information about task
type Task struct {
	Schedule
	Func        FunctionMeta
	Params      []string
	FuncManager config.FunctionManager
}

// New returns an instance of task
func New(function FunctionMeta, params []string, funcManager config.FunctionManager) *Task {
	return &Task{
		Func:        function,
		Params:      params,
		FuncManager: funcManager,
	}
}

// NewWithSchedule creates an instance of task with the provided schedule information
func NewWithSchedule(function FunctionMeta, params []string, schedule Schedule, funcManager config.FunctionManager) *Task {
	return &Task{
		Func:        function,
		Params:      params,
		Schedule:    schedule,
		FuncManager: funcManager,
	}
}

// IsDue returns a boolean indicating whether the task should execute or not
func (task *Task) IsDue() bool {
	timeNow := time.Now()
	return timeNow == task.NextRun || timeNow.After(task.NextRun)
}

// Run will execute the task and schedule it's next run.
func (task *Task) Run() {
	// https://medium.com/@vicky.kurniawan/go-call-a-function-from-string-name-30b41dcb9e12

	b := make([]interface{}, len(task.Params))
	for i := range task.Params {
		b[i] = task.Params[i]
	}

	_, err := task.FuncManager.Call(task.Func.Name, b...)
	if err != nil {
		log.Printf("Error calling function %s. Error: %s", task.Func.Name, err)
	}
}

// Hash will return the SHA1 representation of the task's data.
func (task *Task) Hash() ID {
	hash := sha1.New()
	_, _ = io.WriteString(hash, task.Func.Name)
	_, _ = io.WriteString(hash, fmt.Sprintf("%+v", task.Params))
	_, _ = io.WriteString(hash, fmt.Sprintf("%s", task.Schedule.Duration))
	_, _ = io.WriteString(hash, fmt.Sprintf("%t", task.Schedule.IsRecurring))
	return ID(fmt.Sprintf("%x", hash.Sum(nil)))
}

func (task *Task) ScheduleNextRun() {
	if !task.IsRecurring {
		return
	}

	task.LastRun = task.NextRun
	task.NextRun = task.NextRun.Add(task.Duration)
}
