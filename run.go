package main

import (
	"fmt"
	"time"
)

type Run struct {
	Host         *Host
	Tasks        []*Task
	StartTime    time.Time
	Duration     time.Duration
	DialDuration time.Duration
	TaskResults  []*TaskResult
	Errors       []error
}

func (run *Run) Dump() {
	fmt.Printf("-\n")
	fmt.Printf("- host: %s\n", run.Host.Name)
	fmt.Printf("- %d task(s)\n", len(run.Tasks))
	fmt.Printf("- start: %s\n", run.StartTime)
	fmt.Printf("- duration: %s\n", run.Duration)
	fmt.Printf("- ssh dial duration: %s\n", run.DialDuration)
	for _, err := range run.Errors {
		fmt.Printf("-e %s\n", err)
	}
	for _, res := range run.TaskResults {
		fmt.Printf("-- task probe: %s\n", res.Task.Probe.Name)
		fmt.Printf("-- start time: %s\n", res.StartTime)
		fmt.Printf("-- duration: %s\n", res.Duration)
		fmt.Printf("-- exit status: %d\n", res.ExitStatus)
		fmt.Printf("-- next task run: %s\n", res.Task.NextRun)
		for key, val := range res.Values {
			fmt.Printf("-v- '%s' = '%s'\n", key, val)
		}
		for _, err := range res.Errors {
			fmt.Printf("-e- %s\n", err)
		}
		for _, check := range res.FailedChecks {
			fmt.Printf("-F- %s\n", check.Desc)
		}
		for _, log := range res.Logs {
			fmt.Printf("-l- %s\n", log)
		}
	}
}

func (run *Run) addError(err error) {
	Info.Printf("Run error: %s (host '%s')", err, run.Host.Name)
	run.Errors = append(run.Errors, err)
}

func (run *Run) currentTaskResult() *TaskResult {
	if len(run.TaskResults) == 0 {
		return nil
	}
	return run.TaskResults[len(run.TaskResults)-1]
}

func (run *Run) totalErrorCount() int {
	total := len(run.Errors)
	for _, taskResult := range run.TaskResults {
		total += len(taskResult.Errors)
		total += len(taskResult.FailedChecks)
	}
	return total
}

func (run *Run) totalTaskResultErrorCount() int {
	total := 0
	for _, taskResult := range run.TaskResults {
		total += len(taskResult.Errors)
	}
	return total
}

func (run *Run) ReSchedule() {
	for _, task := range run.Tasks {
		task.NextRun = task.PrevRun
	}
	Info.Printf("re-scheduling all tasks for '%s'\n", run.Host.Name)
}

func (run *Run) ReScheduleFailedTasks() {
	for _, task := range run.Tasks {
		for _, cf := range currentFails {
			if cf.RelatedTask == task || cf.RelatedTTask == task {
				task.ReSchedule(time.Now())
				Info.Printf("re-scheduling task '%s'\n", task.Probe.Name)
			}
		}
	}
}

func (run *Run) DoChecks() {
	for _, taskResult := range run.TaskResults {
		taskResult.DoChecks()
	}
}

func (run *Run) Go() {
	const bootstrap = "bash -s --"

	timeout := time.Second * 59
	timeoutChan := time.After(timeout)

	run.StartTime = time.Now()
	defer func() {
		run.Duration = time.Now().Sub(run.StartTime)
	}()

	if err := run.Host.Connection.Connect(); err != nil {
		run.addError(err)
		return
	}
	defer run.Host.Connection.Close()

	run.DialDuration = time.Now().Sub(run.StartTime)
	if run.DialDuration > run.Host.Connection.SshConnTimeWarn {
		run.addError(fmt.Errorf("SSH connection time was too long: %s (ssh_connection_time_warn = %s)", run.DialDuration, run.Host.Connection.SshConnTimeWarn))
		return
	}

	if err := run.preparePipes(); err != nil {
		run.addError(err)
		return
	}

	ended := make(chan int, 1)

	go func() {
		if err := run.Host.Connection.Session.Run(bootstrap); err != nil {
			run.addError(err)
		}
		ended <- 1
	}()

	select {
	case <-ended:
		// nice
	case <-timeoutChan:
		run.addError(fmt.Errorf("timeout for this run, after %s", timeout))
		Trace.Println("run timeout")
	}
}
