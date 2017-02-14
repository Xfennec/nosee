package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type TaskResult struct {
	Task         *Task
	Values       map[string]string
	ExitStatus   int
	StartTime    time.Time
	Duration     time.Duration
	Logs         []string
	Errors       []error
	FailedChecks []*Check
}

func (result *TaskResult) addError(err error) {
	result.Errors = append(result.Errors, err)
}

func (result *TaskResult) addLog(line string) {
	result.Logs = append(result.Logs, line)
}

func (result *TaskResult) DoChecks() {
	// build parameter map (with values and defaults)
	params := make(map[string]interface{})

	for key, val := range result.Values {
		var err error
		if match, _ := regexp.MatchString("^[0-9]+$", val); match == true {
			params[key], err = strconv.Atoi(val)
			if err != nil {
				result.addError(fmt.Errorf("can't convert '%s' to an int (%s)", val, err))
			}
			continue
		}
		if match, _ := regexp.MatchString("^[0-9]+\\.[0-9]+$", val); match == true {
			params[key], err = strconv.ParseFloat(val, 64)
			if err != nil {
				result.addError(fmt.Errorf("can't convert '%s' to a float64 (%s)", val, err))
			}
			continue
		}
		// string
		params[key] = val
	}

	for _, def := range result.Task.Probe.Defaults {
		params[def.Name] = def.Value
	}

	for _, check := range result.Task.Probe.Checks {
		res, err := check.If.Evaluate(params)
		//~ fmt.Printf("%s: %s (err: %s)\n", check.Desc, res, err)
		if err != nil {
			result.addError(fmt.Errorf("%s (expression '%s' in '%s' check)", err, check.If, check.Desc))
			continue
		}
		if _, ok := res.(bool); ok == false {
			result.addError(fmt.Errorf("[[check]] 'if' must return a boolean value (expression '%s' in '%s' check)", check.If, check.Desc))
			continue
		}

		if res == true {
			result.FailedChecks = append(result.FailedChecks, check)
		}
	}
}
