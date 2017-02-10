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
}
