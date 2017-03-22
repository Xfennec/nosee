package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// HourRange hold a Start and an End in the form of int arrays ([0] = hours, [1] = minutes)
type HourRange struct {
	Start [2]int
	End   [2]int
}

// Alert is the final form of alerts.d files
type Alert struct {
	Name      string
	Disabled  bool
	Targets   []string
	Command   string
	Arguments []string
	Hours     []HourRange
	Days      []int
}

// Ring will send an AlertMessage using this Alert, executing the
// configured command
func (alert *Alert) Ring(msg *AlertMessage) {
	Info.Println("ring: " + alert.Name + ", " + alert.Command /* + " " + strings.Join(alert.Arguments, " ") */)

	// replace $SUBJECT with the real value in the arguments
	// we should perhaps provide some other infos?
	// TODO: should use StringExpandVariables() here
	var args []string
	reSubject := regexp.MustCompile("\\$SUBJECT")
	reType := regexp.MustCompile("\\$TYPE")
	for _, arg := range alert.Arguments {
		arg = reSubject.ReplaceAllString(arg, msg.Subject)
		arg = reType.ReplaceAllString(arg, msg.Type.String())
		args = append(args, arg)
	}

	go func() {
		cmd := exec.Command(alert.Command, args...)

		env := os.Environ()
		env = append(env, fmt.Sprintf("SUBJECT=%s", msg.Subject))
		env = append(env, fmt.Sprintf("DETAILS=%s", msg.Details))
		env = append(env, fmt.Sprintf("TYPE=%s", msg.Type))
		cmd.Env = env

		// we also inject Details thru stdin:
		cmd.Stdin = strings.NewReader(msg.Details)

		if cmdOut, err := cmd.CombinedOutput(); err != nil {
			if len(msg.Classes) == 1 && msg.Classes[0] == GeneralClass {
				Error.Printf("unable to ring an alert to general class! error: %s (%s)\n", err, alert.Command)
				return
			}

			Warning.Printf("There was an error running '%s': %s", alert.Command, err)

			msg.Subject = msg.Subject + " (Fwd)"
			prepend := fmt.Sprintf("WARNING: This alert is re-routed to the 'general' class, because\noriginal alert failed with the following error: %s (%s)\nOutput:%s\n\n", err.Error(), alert.Command, string(cmdOut))
			msg.Details = prepend + msg.Details
			msg.Classes = []string{GeneralClass}
			msg.RingAlerts()
		}
	}()
}

// Ringable will return true if this Alert is currently able to ring
// (no matching day or hour limit)
func (alert *Alert) Ringable() bool {
	now := time.Now()
	nowMins := now.Hour()*60 + now.Minute()
	nowDay := int(now.Weekday())
	hourOk := len(alert.Hours) == 0
	for _, hourRange := range alert.Hours {
		start := hourRange.Start[0]*60 + hourRange.Start[1]
		end := hourRange.End[0]*60 + hourRange.End[1]
		if nowMins >= start && nowMins <= end {
			hourOk = true
			break
		}
	}
	dayOk := len(alert.Days) == 0
	for _, day := range alert.Days {
		if nowDay == day {
			dayOk = true
		}
	}
	return hourOk && dayOk
}
