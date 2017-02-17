package main

import (
	"fmt"
	"strings"
	"time"
	"os"
	"os/exec"
	"regexp"
)

type HourRange struct {
	Start [2]int
	End   [2]int
}

type Alert struct {
	Name        string
	Disabled    bool
	Targets     []string
	Command     string
	Arguments   []string
	Hours       []HourRange
	Days        []int
}

type AlertMessage struct {
	Subject string
	Details string
	Classes []string
}

func (alert *Alert) Ring(msg *AlertMessage) {
	fmt.Println(alert.Name + " " + alert.Command /* + " " + strings.Join(alert.Arguments, " ") */)

	// replace $SUBJECT with the real value in the arguments
	// we should perhaps provide some other infos?
	var args []string
	reSubject := regexp.MustCompile("\\$SUBJECT")
	for _, arg := range alert.Arguments {
		arg = reSubject.ReplaceAllString(arg, msg.Subject)
		args = append(args, arg)
	}

	go func() {
		cmd := exec.Command(alert.Command, args...)

		env := os.Environ()
		env = append(env, fmt.Sprintf("SUBJECT=%s", msg.Subject))
		env = append(env, fmt.Sprintf("DETAILS=%s", msg.Details))
		cmd.Env = env

		// we also inject Details thru stdin:
		cmd.Stdin = strings.NewReader(msg.Details)

		if cmdOut, err := cmd.CombinedOutput(); err != nil {
			// how to deal with  errors? :( Launch another message to a fallback? what about loop?
			fmt.Fprintln(os.Stderr, "There was an error running '%s': ", alert.Command, err)
			fmt.Fprintf(os.Stderr, "%s\n", string(cmdOut))
		}
	}()
}

func (alert *Alert) Ringable() bool {
	now := time.Now()
	nowMins := now.Hour()*60 + now.Minute()
	nowDay := int(now.Weekday())
	hourOk := false
	for _, hourRange := range alert.Hours {
		start := hourRange.Start[0]*60 + hourRange.Start[1]
		end := hourRange.End[0]*60 + hourRange.End[1]
		if nowMins >= start && nowMins <= end {
			hourOk = true
			break
		}
	}
	dayOk := false
	for _, day := range alert.Days {
		if nowDay == day {
			dayOk = true
		}
	}
	return hourOk && dayOk
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
