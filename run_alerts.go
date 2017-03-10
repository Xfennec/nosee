package main

import (
	"bytes"
	"strconv"
)

func (run *Run) AlertsForRun() {
	var bbuf bytes.Buffer
	bbuf.WriteString(run.Host.Name)
	for _, err := range run.Errors {
		bbuf.WriteString(err.Error())
	}
	hash := MD5Hash(bbuf.String())

	currentFail := CurrentFailGetAndInc(hash)
	currentFail.RelatedHost = run.Host

	if currentFail.FailCount > 1 {
		return
	}

	message := AlertMessageCreateForRun(ALERT_BAD, run)
	message.RingAlerts()
}

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

			message := AlertMessageCreateForTaskResult(ALERT_BAD, run, taskRes)
			message.RingAlerts()
		}
	}
}

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

			message := AlertMessageCreateForCheck(ALERT_BAD, run, taskRes, check, currentFail)
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
						message := AlertMessageCreateForCheck(ALERT_GOOD, run, taskRes, check, currentFail)
						message.RingAlerts()
					}
					CurrentFailDelete(hash)
				}
			}
		}
	}
}

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

func (run *Run) ClearAnyCurrentRunFails() {
	found := 0
	for hash, cf := range currentFails {
		if cf.RelatedHost == run.Host {
			found++
			CurrentFailDelete(hash)
		}
	}

	if found > 0 {
		message := AlertMessageCreateForRun(ALERT_GOOD, run)
		message.RingAlerts()
	}
}

func (run *Run) ClearAnyCurrentTasksFails() {
	for _, taskRes := range run.TaskResults {
		if len(taskRes.Errors) == 0 {
			found := 0
			for hash, cf := range currentFails {
				if taskRes.Task == cf.RelatedTTask {
					found++
					CurrentFailDelete(hash)
				}
			}
			if found > 0 {
				message := AlertMessageCreateForTaskResult(ALERT_GOOD, run, taskRes)
				message.RingAlerts()
			}
		}
	}
}
