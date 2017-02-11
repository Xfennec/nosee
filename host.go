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
	Tasks      []*Task
	// probably needs a channel too, right ?
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
		// start := time
		// run tasks (if any needed)
		for _, task := range host.Tasks {
			now := time.Now()
			if now.After(task.NextRun) || now.Equal(task.NextRun) {
				task.NextRun = time.Now().Add(task.Probe.Delay)
				fmt.Printf("%s: run task '%s' on host '%s'\n", time.Now().Format("15:04:05"), task.Probe.Name, host.Name)
			}
		}
		// end := time
		// wait (if any time left)
		time.Sleep(time.Second * 5)
	}
	fmt.Printf("end of %s\n", host.Name)
}
