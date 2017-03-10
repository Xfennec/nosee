package main

import (
	"fmt"
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
	}
}
