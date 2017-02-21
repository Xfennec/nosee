package main

import (
	"fmt"
	"strconv"
)

func (run *Run) AlertsForRun() {
	fmt.Println(">E alerts for run")
	fmt.Println(run.Errors)
}

func (run *Run) AlertsForTasks() {
	fmt.Println(">E alerts for tasks")
}

func (run *Run) AlertsForChecks() {
	// Failures
	for _, taskRes := range run.TaskResults {
		for _, check := range taskRes.FailedChecks {

			hash := MD5Hash(run.Host.Name + taskRes.Task.Probe.Name + strconv.Itoa(check.Index))
			currentFail := CurrentFailGetAndInc(hash)
			currentFail.RelatedTask = taskRes.Task
			if currentFail.FailCount != check.NeededFailures {
				continue // not yet / already done
			}

			message := AlertMessageCreate(ALERT_BAD, run, taskRes, check, currentFail)
			//~ message.Dump()
			message.RingAlerts()
		}
	}

	// Successes
	for _, taskRes := range run.TaskResults {
		for _, check := range taskRes.SuccessfulChecks {
			hash := MD5Hash(run.Host.Name + taskRes.Task.Probe.Name + strconv.Itoa(check.Index))
			// we had a failure for that?
			if currentFail := CurrentFailGetAndDec(hash); currentFail != nil {
				if currentFail.OkCount == check.NeededSuccesses {
					// send the good news and delete this currentFail
					message := AlertMessageCreate(ALERT_GOOD, run, taskRes, check, currentFail)
					//~ message.Dump()
					message.RingAlerts()
					CurrentFailDelete(hash)
				}
			}
		}
	}
}

func (run *Run) Alerts() {
	if run.totalErrorCount() == 0 { // run & tasks errors
		run.DoChecks()
		run.AlertsForChecks()
		run.ReScheduleFailedTasks()
	} else {
		// errors (general)
		if len(run.Errors) > 0 {
			run.AlertsForRun()
		} else {
			run.AlertsForTasks()
		}
	}
}
