package main

import (
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

// Check holds final informations about a check of a probes.d file
type Check struct {
	Index           int
	Desc            string
	If              *govaluate.EvaluableExpression
	Classes         []string
	NeededFailures  int
	NeededSuccesses int
}

// Probe is the final form of probes.d files
type Probe struct {
	Name        string
	Script      string
	ScriptCache *strings.Reader
	Targets     []string
	Delay       time.Duration
	Timeout     time.Duration
	Arguments   string
	Defaults    map[string]interface{}
	Checks      []*Check
	RunIf       *govaluate.EvaluableExpression
}

// MissingDefaults return a slice with names of defaults used in Check 'If'
// expressions and Probe script arguments. The slice length is 0 if no
// missing default were found.
func (probe *Probe) MissingDefaults() []string {
	missing := make(map[string]bool)

	for _, check := range probe.Checks {
		for _, name := range check.If.Vars() {
			if IsAllUpper(name) {
				continue
			}
			if _, ok := probe.Defaults[name]; ok != true {
				missing[name] = true
			}
		}
	}

	vars := StringFindVariables(probe.Arguments)
	for _, name := range vars {
		if _, ok := probe.Defaults[name]; ok != true {
			missing[name] = true
		}
	}

	// map to slice:
	var missSlice []string
	for key := range missing {
		missSlice = append(missSlice, key)
	}

	return missSlice
}
