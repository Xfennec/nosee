package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Knetic/govaluate"
)

// CheckFunctions will hold all custom govaluate functions for Check 'If'
// expressions
var CheckFunctions map[string]govaluate.ExpressionFunction

// CheckFunctionsInit will initialize CheckFunctions global variable
func CheckFunctionsInit() {
	CheckFunctions = map[string]govaluate.ExpressionFunction{

		"strlen": func(args ...interface{}) (interface{}, error) {
			length := len(args[0].(string))
			return (float64)(length), nil
		},

		"ping": func(args ...interface{}) (interface{}, error) {
			if len(args) > 0 {
				return nil, fmt.Errorf("ping function: too much arguments")
			}
			return (string)("pong"), nil
		},

		"date": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("date function: wrong argument count (1 required)")
			}
			format := args[0].(string)
			now := time.Now()
			switch format {
			case "hour":
				return (float64)(now.Hour()), nil
			case "minute":
				return (float64)(now.Minute()), nil
			case "time":
				return (float64)((float64)(now.Hour()) + (float64)(now.Minute())/60.0), nil
			case "dow", "day-of-week":
				// Sunday = 0
				return (float64)(now.Weekday()), nil
			case "dom", "day-of-month":
				return (float64)(now.Day()), nil
			case "now":
				return (float64)(now.Unix()), nil
			}

			if match, _ := regexp.MatchString("^[0-9]{1,2}:[0-9]{2}$", format); match == true {
				t, err := alertCheckHour(format)
				if err != nil {
					return nil, fmt.Errorf("date function: invalid hour '%s': %s", format, err)
				}
				return (float64)((float64)(t[0]) + (float64)(t[1])/60.0), nil
			}

			return nil, fmt.Errorf("date function: invalid format '%s'", format)
		},
	}
}
