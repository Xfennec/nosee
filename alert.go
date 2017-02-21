package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type HourRange struct {
	Start [2]int
	End   [2]int
}

type Alert struct {
	Name      string
	Disabled  bool
	Targets   []string
	Command   string
	Arguments []string
	Hours     []HourRange
	Days      []int
}

func (alert *Alert) Ring(msg *AlertMessage) {
	fmt.Println(alert.Name + " " + alert.Command /* + " " + strings.Join(alert.Arguments, " ") */)

	// replace $SUBJECT with the real value in the arguments
	// we should perhaps provide some other infos?
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
			// how to deal with  errors? :( Launch another message to a fallback? what about loop?
			fmt.Fprintf(os.Stderr, "There was an error running '%s': %s\n", alert.Command, err)
			fmt.Fprintf(os.Stderr, "%s\n", string(cmdOut))
		}
	}()
}

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
