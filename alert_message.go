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

func AlertMessageCreate(aType AlertMessageType, run *Run, taskRes *TaskResult, check *Check, currentFail *CurrentFail) *AlertMessage {
	var message AlertMessage

	// Host: Check (Task)
	message.Subject = fmt.Sprintf("[%s] %s: %s (%s)", aType, run.Host.Name, check.Desc, taskRes.Task.Probe.Name)
	message.Type = aType

	var details bytes.Buffer

	switch aType {
	case ALERT_BAD:
		details.WriteString("An alert **is** ringing.\n\n")
	case ALERT_GOOD:
		details.WriteString("This alert is **no more** ringing\n\n")
	}

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
		// !!!
		// what to do with this case? :(
		// !!!
		fmt.Printf("Error, no matching alert for this failure : '%s' with class(es): %s\n", msg.Subject, strings.Join(msg.Classes, ", "))
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
