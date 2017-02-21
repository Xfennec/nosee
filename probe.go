package main

import (
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

type Default struct {
	Name  string
	Value interface{}
}

type Check struct {
	Index           int
	Desc            string
	If              *govaluate.EvaluableExpression
	Classes         []string
	NeededFailures  int
	NeededSuccesses int
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
		for _, name := range check.If.Vars() {
			if IsAllUpper(name) {
				continue
			}
			if defaults[name] != true {
				missing[name] = true
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
