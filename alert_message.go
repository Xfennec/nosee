package main

import (
	"bytes"
	"fmt"
	"strings"
)

type AlertMessageType uint8

const (
	ALERT_GOOD AlertMessageType = iota + 1
	ALERT_BAD
)

var AlertMessageTypeStr = [...]string{
	"GOOD",
	"BAD",
}

type AlertMessage struct {
	Type    AlertMessageType
	Subject string
	Details string
	Classes []string
}

func (amt AlertMessageType) String() string {
	if amt == 0 {
		return "INVALID_TYPE"
	}
	return AlertMessageTypeStr[amt-1]
}

func AlertMessageCreateForRun(aType AlertMessageType, run *Run) *AlertMessage {
	var message AlertMessage

	message.Subject = fmt.Sprintf("[%s] %s: run error(s)", aType, run.Host.Name)
	message.Type = aType

	var details bytes.Buffer

	switch aType {
	case ALERT_BAD:
		details.WriteString("A least one error occured during a run for this host. (" + run.StartTime.Format("2006-01-02 15:04:05") + ")\n")
		details.WriteString("\n")
		details.WriteString("Error(s):\n")
		for _, err := range run.Errors {
			details.WriteString(err.Error() + "\n")
		}
	case ALERT_GOOD:
		details.WriteString("No more run errors for this host. (" + run.StartTime.Format("2006-01-02 15:04:05") + ")\n")
	}

	message.Details = details.String()

	message.Classes = []string{"general"}

	return &message
}

// taskResult may be nil for GOOD messages
func AlertMessageCreateForTaskResult(aType AlertMessageType, run *Run, taskResult *TaskResult) *AlertMessage {
	var message AlertMessage

	message.Subject = fmt.Sprintf("[%s] %s: %s: task error(s)", aType, run.Host.Name, taskResult.Task.Probe.Name)
	message.Type = aType

	var details bytes.Buffer

	switch aType {
	case ALERT_BAD:
		details.WriteString("A least one error occured during a task for this host. (" + taskResult.StartTime.Format("2006-01-02 15:04:05") + ")\n")
		details.WriteString("\n")
		details.WriteString("Error(s):\n")
		for _, err := range taskResult.Errors {
			details.WriteString(err.Error() + "\n")
		}
	case ALERT_GOOD:
		details.WriteString("No more errors for this task on this host. (" + taskResult.StartTime.Format("2006-01-02 15:04:05") + ")\n")
	}

	message.Details = details.String()

	message.Classes = []string{"general"}

	return &message
}

func AlertMessageCreateForCheck(aType AlertMessageType, run *Run, taskRes *TaskResult, check *Check, currentFail *CurrentFail) *AlertMessage {
	var message AlertMessage

	// Host: Check (Task)
	message.Subject = fmt.Sprintf("[%s] %s: %s (%s)", aType, run.Host.Name, check.Desc, taskRes.Task.Probe.Name)
	message.Type = aType

	var details bytes.Buffer

	switch aType {
	case ALERT_BAD:
		details.WriteString("An alert **is** ringing.\n\n")
	case ALERT_GOOD:
		details.WriteString("This alert is **no more** ringing.\n\n")
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
	message.Details = details.String()

	message.Classes = check.Classes

	return &message
}

func (msg *AlertMessage) Dump() {
	fmt.Printf("---\n")
	fmt.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("%s\n---\n", msg.Details)
}

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
		if len(msg.Classes) == 1 && msg.Classes[0] == "general" {
			Error.Printf("unable to ring an alert : can't match the 'general' class!\n")
			return
		}

		Warning.Printf("no matching alert for this failure: '%s' with class(es): %s\n", msg.Subject, strings.Join(msg.Classes, ", "))

		// forward the alert to 'general' class:
		msg.Subject = msg.Subject + " (Fwd)"
		prepend := "WARNING: This alert is re-routed to the 'general' class, because no alert matches its orginial classes (" + strings.Join(msg.Classes, ", ") + ")\n\n"
		msg.Details = prepend + msg.Details
		msg.Classes = []string{"general"}
		msg.RingAlerts()
	}
}

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
