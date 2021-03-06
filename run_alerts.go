package main

import (
	"bytes"
	"strconv"
)

// AlertsForRun creates a currentFail entry for this Run (if not already done)
// and rings corresponding alerts
func (run *Run) AlertsForRun() {
	var bbuf bytes.Buffer
	bbuf.WriteString(run.Host.Name)
	// We now limit to one Fail per host, otherwise we may flood
	// the user with Errors (ex: "alert, ssh connection 11s", then the same
	// with 11.5s, etc). If there's an issue with a host, you have to fix it
	// to get the others (if any left), it makes sense.
	/*for _, err := range run.Errors {
		bbuf.WriteString(err.Error())
	}*/
	hash := MD5Hash(bbuf.String())

	currentFail := CurrentFailGetAndInc(hash)
	currentFail.RelatedHost = run.Host

	if currentFail.FailCount > 1 {
		return
	}

	message := AlertMessageCreateForRun(AlertBad, run, currentFail)
	message.RingAlerts()
}

// AlertsForTasks creates currentFail entries for each failed TaskResults
// (if not already done) and rings corresponding alerts
func (run *Run) AlertsForTasks() {
	for _, taskRes := range run.TaskResults {
		if len(taskRes.Errors) > 0 {
			var bbuf bytes.Buffer
			bbuf.WriteString(run.Host.Name + taskRes.Task.Probe.Name)
			for _, err := range taskRes.Errors {
				bbuf.WriteString(err.Error())
			}
			hash := MD5Hash(bbuf.String())

			currentFail := CurrentFailGetAndInc(hash)
			currentFail.RelatedTTask = taskRes.Task
			if currentFail.FailCount > 1 {
				return
			}

			message := AlertMessageCreateForTaskResult(AlertBad, run, taskRes, currentFail)
			message.RingAlerts()
		}
	}
}

// AlertsForChecks creates currentFail entries for every FailedChecks of
// every TaskResults (if not already done) and rings corresponding alerts
func (run *Run) AlertsForChecks() {
	// Failures
	for _, taskRes := range run.TaskResults {
		for _, check := range taskRes.FailedChecks {
			Info.Printf("task '%s', check '%s' failed (%s)\n", taskRes.Task.Probe.Name, check.Desc, run.Host.Name)

			hash := MD5Hash(run.Host.Name + taskRes.Task.Probe.Name + strconv.Itoa(check.Index))
			currentFail := CurrentFailGetAndInc(hash)
			currentFail.RelatedTask = taskRes.Task
			if currentFail.FailCount != check.NeededFailures {
				continue // not yet / already done
			}

			message := AlertMessageCreateForCheck(AlertBad, run, taskRes, check, currentFail)
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
					Info.Printf("task '%s', check '%s' is now OK (%s)\n", taskRes.Task.Probe.Name, check.Desc, run.Host.Name)
					// send the good news (if the bad one was sent) and delete this currentFail
					if currentFail.FailCount >= check.NeededFailures {
						message := AlertMessageCreateForCheck(AlertGood, run, taskRes, check, currentFail)
						message.RingAlerts()
					}
					CurrentFailDelete(hash)
				}
			}
		}
	}
}

// Alerts checks for Run failures, Task failures and Check
// failures and call corresponding AlertsFor*() functions
func (run *Run) Alerts() {
	run.ClearAnyCurrentTasksFails()

	if run.totalErrorCount() == 0 {
		run.ClearAnyCurrentRunFails()
		run.DoChecks()
		if run.totalTaskResultErrorCount() > 0 {
			Info.Printf("found some 'tasks' error(s) (post-checks)\n")
			run.AlertsForTasks()
		} else {
			// ideal path, let's see if there's any check errors ?
			run.AlertsForChecks()
		}
	} else { // run & tasks errors
		if len(run.Errors) > 0 {
			Info.Printf("found some 'run' error(s)\n")
			run.AlertsForRun()
			run.ReSchedule()
		} else {
			Info.Printf("found some 'tasks' error(s)\n")
			run.AlertsForTasks()
		}
	}

	run.ReScheduleFailedTasks()
}

// ClearAnyCurrentRunFails deletes any currentFail for the Run (same Host)
// and then rings GOOD alerts
func (run *Run) ClearAnyCurrentRunFails() {
	for hash, cf := range currentFails {
		if cf.RelatedHost == run.Host {
			// there was a time when we were only ringing one message
			// for the whole host, but it's compliant with UniqueID idea
			message := AlertMessageCreateForRun(AlertGood, run, cf)
			message.RingAlerts()
			CurrentFailDelete(hash)
		}
	}
}

// ClearAnyCurrentTasksFails deletes any currentFail for Run Tasks
// and then rings GOOD alerts
func (run *Run) ClearAnyCurrentTasksFails() {
	for _, taskRes := range run.TaskResults {
		if len(taskRes.Errors) == 0 {
			for hash, cf := range currentFails {
				if taskRes.Task == cf.RelatedTTask {
					message := AlertMessageCreateForTaskResult(AlertGood, run, taskRes, cf)
					message.RingAlerts()
					CurrentFailDelete(hash)
				}
			}
		}
	}
}
