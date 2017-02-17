package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func (run *Run) AlertsForRun() {
	fmt.Println(">E alerts for run")
}

func (run *Run) AlertsForTasks() {
	fmt.Println(">E alerts for tasks")
}

func (run *Run) AlertsForChecks() {
	fmt.Println(">E alerts for checks")

	for _, taskRes := range run.TaskResults {
		for _, check := range taskRes.FailedChecks {

			hash := MD5Hash(run.Host.Name + taskRes.Task.Probe.Name + strconv.Itoa(check.Index))
			currentFail := CurrentFailGet(hash)
			if currentFail.FailCount != check.NeededFailures {
				continue // not yet / already done
			}

			var message AlertMessage

			// Host: Check (Task)
			message.Subject = fmt.Sprintf("%s: %s (%s)", run.Host.Name, check.Desc, taskRes.Task.Probe.Name)

			var details bytes.Buffer
			details.WriteString("Failure time: " + currentFail.FailStart.Format("2006-01-02 15:04:05") + "\n")
			details.WriteString("Last task time: " + taskRes.StartTime.Format("2006-01-02 15:04:05") + "\n")
			details.WriteString("Class(es): " + strings.Join(check.Classes, ", ") + "\n")
			details.WriteString("Failed condition was: " + check.If.String() + "\n")
			details.WriteString("\n")
			details.WriteString("Values:\n")
			for _, token := range check.If.Vars() {
				details.WriteString("- " + token + ": " + taskRes.Values[token] + "\n")
			}
			details.WriteString("\n")
			details.WriteString("All values for this run:\n")
			for _, tr := range run.TaskResults {
				details.WriteString("- " + tr.Task.Probe.Name + ":\n")
				for key, val := range tr.Values {
					details.WriteString("--- " + key + ": " + val + "\n")
				}
			}

			message.Details = details.String()

			message.Classes = check.Classes
			//~ message.Dump()
			message.RingAlerts()
		}
	}
}

func (run *Run) Alerts() {
	if run.totalErrorCount() == 0 { // run & tasks errors
		run.DoChecks()
		if run.totalErrorCount() == 0 { // check errors
			// OK, no errors
		} else {
			// errors (checks)
			run.ReScheduleFailedTasks()
			run.AlertsForChecks()
		}
	} else {
		// errors (general)
		if len(run.Errors) > 0 {
			run.AlertsForRun()
		} else {
			run.AlertsForTasks()
		}
	}
}
