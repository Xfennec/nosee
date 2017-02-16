package main

import (
	"fmt"
	"strings"
)

type HourRange struct {
	Start [2]int
	End   [2]int
}

type Alert struct {
	Name        string
	Disabled    bool
	Targets     []string
	Script      string
	ScriptCache *strings.Reader
	Arguments   string
	Hours       []HourRange
	Days        []int
}

type AlertMessage struct {
	Subject string
	Details string
	Classes []string
}

func (alert *Alert) Ring(msg *AlertMessage) {
	fmt.Println(alert.Name)
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
			alert.Ring(msg)
			ringCount++
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
