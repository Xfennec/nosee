package main

import (
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

type Default struct {
	Name  string
	Value interface{}
	//~ Type  string
}

type Check struct {
	Desc    string
	If      *govaluate.EvaluableExpression
	Classes []string
}

type Probe struct {
	Name        string
	Script      string
	ScriptCache *strings.Reader
	Targets     []string
	Delay       time.Duration
	Timeout     time.Duration
	Arguments   string
	Defaults    []*Default
	Checks      []*Check
}

func (probe *Probe) MissingDefaults() []string {
	missing := make(map[string]bool)
	defaults := make(map[string]bool)

	for _, def := range probe.Defaults {
		defaults[def.Name] = true
	}

	for _, check := range probe.Checks {
		for _, token := range check.If.Tokens() {
			if token.Kind == govaluate.VARIABLE {
				name := token.Value.(string)
				if IsAllUpper(name) {
					continue
				}
				if defaults[name] != true {
					missing[name] = true
				}
			}
		}
	}

	// map to slice:
	var missSlice []string
	for key, _ := range missing {
		missSlice = append(missSlice, key)
	}

	return missSlice
}
