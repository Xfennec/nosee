package main

import (
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
