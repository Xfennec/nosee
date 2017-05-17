package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// AlertMessageType definition
type AlertMessageType uint8

// AlertMessageType numeric values
const (
	AlertGood AlertMessageType = iota + 1
	AlertBad
)

// AlertMessageTypeStr stores matching strings
var AlertMessageTypeStr = [...]string{
	"GOOD",
	"BAD",
}

// AlertMessage will store the text of the error
type AlertMessage struct {
	Type     AlertMessageType
	Subject  string
	Details  string
	Classes  []string
	UniqueID string
	Hostname string
	DateTime time.Time
}

// GeneralClass is a "general" class for very important general messages
const GeneralClass = "general"

func (amt AlertMessageType) String() string {
	if amt == 0 {
		return "INVALID_TYPE"
	}
	return AlertMessageTypeStr[amt-1]
}

// AlertMessageCreateForRun creates a new AlertMessage with AlertGood or
// AlertBad type for a Run
func AlertMessageCreateForRun(aType AlertMessageType, run *Run, currentFail *CurrentFail) *AlertMessage {
	var message AlertMessage

	message.Subject = fmt.Sprintf("[%s] %s: run error(s)", aType, run.Host.Name)
	message.Type = aType
	message.UniqueID = currentFail.UniqueID
	message.Hostname = run.Host.Name
	message.DateTime = run.StartTime

	var details bytes.Buffer

	switch aType {
	case AlertBad:
		details.WriteString("A least one error occured during a run for this host. (" + run.StartTime.Format("2006-01-02 15:04:05") + ")\n")
		details.WriteString("\n")
		details.WriteString("Error(s):\n")
		for _, err := range run.Errors {
			details.WriteString(err.Error() + "\n")
		}
	case AlertGood:
		details.WriteString("No more run errors for this host. (" + run.StartTime.Format("2006-01-02 15:04:05") + ")\n")
	}

	details.WriteString("\n")
	details.WriteString("Unique failure ID: " + message.UniqueID + "\n")
	message.Details = details.String()

	message.Classes = []string{GeneralClass}

	return &message
}

// AlertMessageCreateForTaskResult creates an AlertGood or AlertBad message for a TaskResult
func AlertMessageCreateForTaskResult(aType AlertMessageType, run *Run, taskResult *TaskResult, currentFail *CurrentFail) *AlertMessage {
	var message AlertMessage

	message.Subject = fmt.Sprintf("[%s] %s: %s: task error(s)", aType, run.Host.Name, taskResult.Task.Probe.Name)
	message.Type = aType
	message.UniqueID = currentFail.UniqueID
	message.Hostname = run.Host.Name
	message.DateTime = taskResult.StartTime

	var details bytes.Buffer

	switch aType {
	case AlertBad:
		details.WriteString("A least one error occured during a task for this host. (" + taskResult.StartTime.Format("2006-01-02 15:04:05") + ")\n")
		details.WriteString("\n")
		details.WriteString("Error(s):\n")
		for _, err := range taskResult.Errors {
			details.WriteString(err.Error() + "\n")
		}
		if len(taskResult.Logs) > 0 {
			details.WriteString("\n")
			details.WriteString("Logs(s):\n")
			for _, log := range taskResult.Logs {
				details.WriteString(log + "\n")
			}
		}
	case AlertGood:
		details.WriteString("No more errors for this task on this host. (" + taskResult.StartTime.Format("2006-01-02 15:04:05") + ")\n")
	}

	details.WriteString("\n")
	details.WriteString("Unique failure ID: " + message.UniqueID + "\n")
	message.Details = details.String()

	message.Classes = []string{GeneralClass}

	return &message
}

// AlertMessageCreateForCheck creates a AlertGood or AlertBad message for a Check
func AlertMessageCreateForCheck(aType AlertMessageType, run *Run, taskRes *TaskResult, check *Check, currentFail *CurrentFail) *AlertMessage {
	var message AlertMessage

	// Host: Check (Task)
	message.Subject = fmt.Sprintf("[%s] %s: %s (%s)", aType, run.Host.Name, check.Desc, taskRes.Task.Probe.Name)
	message.Type = aType
	message.UniqueID = currentFail.UniqueID
	message.Hostname = run.Host.Name

	var details bytes.Buffer

	switch aType {
	case AlertBad:
		details.WriteString("An alert **is** ringing.\n\n")
		message.DateTime = currentFail.FailStart
	case AlertGood:
		details.WriteString("This alert is **no more** ringing.\n\n")
		message.DateTime = taskRes.StartTime
	}

	details.WriteString("Failure time: " + currentFail.FailStart.Format("2006-01-02 15:04:05") + "\n")
	details.WriteString("Last task time: " + taskRes.StartTime.Format("2006-01-02 15:04:05") + "\n")
	details.WriteString("Class(es): " + strings.Join(check.Classes, ", ") + "\n")
	details.WriteString("Failed condition was: " + check.If.String() + "\n")
	details.WriteString("\n")
	details.WriteString("Values:\n")
	for _, token := range check.If.Vars() {
		if IsAllUpper(token) {
			details.WriteString("- " + token + ": " + taskRes.Values[token] + "\n")
		} else {
			val := InterfaceValueToString(taskRes.Task.Probe.Defaults[token])
			if _, exists := taskRes.Host.Defaults[token]; exists == true {
				val = InterfaceValueToString(taskRes.Host.Defaults[token])
			}
			details.WriteString("- " + token + ": " + val + "\n")
		}
	}
	details.WriteString("\n")
	details.WriteString(fmt.Sprintf("All values for this run (%s):\n", run.Duration))
	for _, tr := range run.TaskResults {
		details.WriteString(fmt.Sprintf("- %s (%s):\n", tr.Task.Probe.Name, tr.Duration))
		for key, val := range tr.Values {
			details.WriteString("--- " + key + ": " + val + "\n")
		}
	}
	details.WriteString("\n")
	details.WriteString("Unique failure ID: " + message.UniqueID + "\n")
	message.Details = details.String()

	message.Classes = check.Classes

	return &message
}

// Dump prints AlertMessage informations on the screen for debugging purposes
func (msg *AlertMessage) Dump() {
	fmt.Printf("---\n")
	fmt.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("%s\n---\n", msg.Details)
}

// RingAlerts will search and ring all alerts for this AlertMessage
func (msg *AlertMessage) RingAlerts() {
	ringCount := 0
	for _, alert := range globalAlerts {
		if msg.MatchAlertTargets(alert) {
			if alert.Ringable() {
				alert.Ring(msg)
				ringCount++
			}
		}
	}

	if ringCount == 0 {
		// if class is already "general", we're f*cked :(
		if len(msg.Classes) == 1 && msg.Classes[0] == GeneralClass {
			Error.Printf("unable to ring an alert : can't match the 'general' class!\n")
			return
		}

		Warning.Printf("no matching alert for this failure: '%s' with class(es): %s\n", msg.Subject, strings.Join(msg.Classes, ", "))

		// forward the alert to 'general' class:
		msg.Subject = msg.Subject + " (Fwd)"
		prepend := "WARNING: This alert is re-routed to the 'general' class, because no alert matches its orginial classes (" + strings.Join(msg.Classes, ", ") + ")\n\n"
		msg.Details = prepend + msg.Details
		msg.Classes = []string{GeneralClass}
		msg.RingAlerts()
	}
}

// HasClass returns true if this AlertMessage has this class
func (msg *AlertMessage) HasClass(class string) bool {
	if class == "*" {
		return true
	}

	for _, hClass := range msg.Classes {
		if hClass == class {
			return true
		}
	}
	return false
}

// MatchAlertTargets returns true if this AlertMessage matches alert's classes
func (msg *AlertMessage) MatchAlertTargets(alert *Alert) bool {
	for _, pTargets := range alert.Targets {
		tokens := strings.Split(pTargets, "&")
		matched := 0
		mustMatch := len(tokens)
		for _, token := range tokens {
			token := strings.TrimSpace(token)
			if msg.HasClass(token) {
				matched++
			}
		}
		if matched == mustMatch {
			return true
		}
	}
	return false
}
