package main

import (
	"fmt"
	"strings"
	"time"
)

type Host struct {
	Name       string
	Classes    []string
	Connection *Connection
	Defaults   map[string]interface{}
	Tasks      []*Task
}

func (host *Host) HasClass(class string) bool {
	if class == "*" {
		return true
	}

	for _, hClass := range host.Classes {
		if hClass == class {
			return true
		}
	}
	return false
}

func (host *Host) MatchProbeTargets(probe *Probe) bool {
	for _, pTargets := range probe.Targets {
		tokens := strings.Split(pTargets, "&")
		matched := 0
		mustMatch := len(tokens)
		for _, token := range tokens {
			token := strings.TrimSpace(token)
			if host.HasClass(token) {
				matched++
			}
		}
		if matched == mustMatch {
			return true
		}
	}
	return false
}

func (host *Host) Schedule() {
	for {
		start := time.Now()

		var run Run
		run.Host = host
		run.StartTime = start

		for _, task := range host.Tasks {
			if start.After(task.NextRun) || start.Equal(task.NextRun) {
				task.ReSchedule(start.Add(task.Probe.Delay))
				Info.Printf("host '%s', running task '%s'\n", host.Name, task.Probe.Name)
				run.Tasks = append(run.Tasks, task)
			}
		}

		if len(run.Tasks) > 0 {
			run.Go()
			run.Alerts()
			Trace.Printf("currentFails count = %d\n", len(currentFails))
		}

		end := time.Now()
		dur := end.Sub(start)

		if dur < time.Minute {
			remains := time.Minute - dur
			time.Sleep(remains)
		} else {
			run.addError(fmt.Errorf("run duration was too long (%s)", run.Duration))
		}
		Trace.Printf("(loop %s)\n", host.Name)
	}
}
