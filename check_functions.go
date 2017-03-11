package main

import (
	"fmt"
	"time"

	"github.com/Knetic/govaluate"
)

var CheckFunctions map[string]govaluate.ExpressionFunction

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
			case "hours":
				return (float64)((float64)(now.Hour()) + (float64)(now.Minute())/60.0), nil
			case "dow", "day-of-week":
				return (float64)(now.Weekday()), nil
			case "dom", "day-of-month":
				return (float64)(now.Day()), nil
				// Disabled until I sort out this timezone offset issues
			case "now":
				// *remove* timezone information, since govaluate use time.Parse() (always UTC)
				t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
				return (float64)(t.Unix()), nil
			}
			return nil, fmt.Errorf("date function: invalid format '%s'", format)
		},
	}
}
