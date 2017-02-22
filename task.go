package main

import (
	"time"
)

type Task struct {
	Probe *Probe
	//~ LastRun        time.Time
	//~ RunCount       int
	//~ RemainingTicks int
	NextRun time.Time
	PrevRun time.Time
}

func (task *Task) ReSchedule(val time.Time) {
	task.PrevRun = task.NextRun
	task.NextRun = val
}
