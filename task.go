package main

import (
	"fmt"
	"time"
)

// Task structure holds (mainly timing) informations about a Task
// next and previous execution
type Task struct {
	Probe *Probe
	//~ LastRun        time.Time
	//~ RunCount       int
	//~ RemainingTicks int
	NextRun time.Time
	PrevRun time.Time
}

// ReSchedule is used to schedule another run for this
// task in the future
func (task *Task) ReSchedule(val time.Time) {
	task.PrevRun = task.NextRun
	task.NextRun = val
}

// Taskable returns true if the task is currently available (see RunIf expression)
func (task *Task) Taskable() (bool, error) {
	// no RunIf condition? taskable, then
	if task.Probe.RunIf == nil {
		return true, nil
	}
	res, err := task.Probe.RunIf.Evaluate(nil)
	if err != nil {
		return false, fmt.Errorf("%s (run_if expression '%s' probe)", err, task.Probe.Name)
	}
	if _, ok := res.(bool); ok == false {
		return false, fmt.Errorf("'run_if' must return a boolean value (probe '%s')", task.Probe.Name)
	}
	return res.(bool), nil
}
