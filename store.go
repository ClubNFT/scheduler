package scheduler

import (
	"github.com/ClubNFT/scheduler/config"
	"strconv"
	"time"

	"github.com/ClubNFT/scheduler/storage"
	"github.com/ClubNFT/scheduler/task"
)

type storeBridge struct {
	store       storage.TaskStore
	funcManager config.FunctionManager
}

func (sb *storeBridge) Add(task *task.Task) error {
	attributes, err := sb.getTaskAttributes(task)
	if err != nil {
		return err
	}
	return sb.store.Add(attributes)
}

func (sb *storeBridge) Update(task *task.Task) error {
	attributes, err := sb.getTaskAttributes(task)
	if err != nil {
		return err
	}
	return sb.store.Update(attributes)
}

func (sb *storeBridge) Fetch() ([]*task.Task, error) {
	storedTasks, err := sb.store.Fetch()
	if err != nil {
		return []*task.Task{}, err
	}
	var tasks []*task.Task
	for _, storedTask := range storedTasks {
		lastRun, err := time.Parse(time.RFC3339, storedTask.LastRun)
		if err != nil {
			return nil, err
		}

		nextRun, err := time.Parse(time.RFC3339, storedTask.NextRun)
		if err != nil {
			return nil, err
		}

		duration, err := time.ParseDuration(storedTask.Duration)
		if err != nil {
			return nil, err
		}

		isRecurring, err := strconv.Atoi(storedTask.IsRecurring)
		if err != nil {
			return nil, err
		}

		t := task.NewWithSchedule(task.FunctionMeta{Name: storedTask.Name}, storedTask.Params, task.Schedule{
			IsRecurring: isRecurring == 1,
			Duration:    time.Duration(duration),
			LastRun:     lastRun,
			NextRun:     nextRun,
		}, sb.funcManager)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (sb *storeBridge) Remove(task *task.Task) error {
	attributes, err := sb.getTaskAttributes(task)
	if err != nil {
		return err
	}
	return sb.store.Remove(attributes)
}

func (sb *storeBridge) getTaskAttributes(task *task.Task) (storage.TaskAttributes, error) {
	isRecurring := 0
	if task.IsRecurring {
		isRecurring = 1
	}

	return storage.TaskAttributes{
		Hash:        string(task.Hash()),
		Name:        task.Func.Name,
		LastRun:     task.LastRun.Format(time.RFC3339),
		NextRun:     task.NextRun.Format(time.RFC3339),
		Duration:    task.Duration.String(),
		IsRecurring: strconv.Itoa(isRecurring),
		Params:      task.Params,
	}, nil
}
