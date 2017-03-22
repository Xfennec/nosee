package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type tomlDefault struct {
	Name  string
	Value interface{}
}

type tomlCheck struct {
	Desc            string
	If              string
	Classes         []string
	NeededFailures  int `toml:"needed_failures"`
	NeededSuccesses int `toml:"needed_successes"`
}

type tomlProbe struct {
	Name      string
	Disabled  bool
	Script    string
	Targets   []string
	Delay     Duration
	Timeout   Duration
	Arguments string
	Default   []tomlDefault
	Check     []tomlCheck
}

func checkTomlDefault(pDefaults map[string]interface{}, tDefaults []tomlDefault) error {
	for _, tDefault := range tDefaults {

		if tDefault.Name == "" {
			return errors.New("[[default]] with invalid or missing 'name'")
		}

		if IsAllUpper(tDefault.Name) {
			return fmt.Errorf("[[default]] name is invalid (all uppercase): %s", tDefault.Name)
		}

		valid := false
		switch tDefault.Value.(type) {
		case string:
			valid = true
		case int32:
			valid = true
		case int64:
			valid = true
		case float32:
			valid = true
		case float64:
			valid = true
		}

		if valid == false {
			return fmt.Errorf("[[default]] invalid value type for '%s'", tDefault.Name)
		}

		if _, exists := pDefaults[tDefault.Name]; exists == true {
			return fmt.Errorf("Config error: duplicate default name '%s'", tDefault.Name)
		}

		pDefaults[tDefault.Name] = tDefault.Value
	}
	return nil
}

func tomlProbeToProbe(tProbe *tomlProbe, config *Config) (*Probe, error) {
	var probe Probe

	if tProbe.Disabled == true {
		return nil, nil
	}

	if tProbe.Name == "" {
		return nil, errors.New("invalid or missing 'name'")
	}
	probe.Name = tProbe.Name

	if tProbe.Script == "" {
		return nil, errors.New("invalid or missing 'script'")
	}

	scriptPath := path.Clean(config.configPath + "/scripts/probes/" + tProbe.Script)
	stat, err := os.Stat(scriptPath)

	if err != nil {
		return nil, fmt.Errorf("invalid 'script' file '%s': %s", scriptPath, err)
	}

	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("is not a regular 'script' file '%s'", scriptPath)
	}
	probe.Script = scriptPath

	str, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("error reading script file '%s': %s", scriptPath, err)
	}
	if config.CacheScripts {
		probe.ScriptCache = strings.NewReader(string(str))
	}

	if tProbe.Targets == nil {
		return nil, errors.New("no valid 'targets' parameter found")
	}

	if len(tProbe.Targets) == 0 {
		return nil, errors.New("empty 'targets'")
	}
	// explode targets on & and check IsValidTokenName
	for _, targets := range tProbe.Targets {
		if targets == "*" {
			continue
		}
		tokens := strings.Split(targets, "&")
		for _, token := range tokens {
			token := strings.TrimSpace(token)
			if !IsValidTokenName(token) {
				return nil, fmt.Errorf("invalid 'target' class name '%s'", token)
			}
		}
	}
	probe.Targets = tProbe.Targets

	if tProbe.Delay.Duration == 0 {
		return nil, errors.New("invalid or missing 'delay'")
	}

	if tProbe.Delay.Duration < (1 * time.Minute) {
		return nil, errors.New("'delay' can't be less than a minute")
	}

	minutes := float64(tProbe.Delay.Duration) / float64(time.Minute)
	if float64(int(minutes)) != minutes {
		return nil, errors.New("'delay' granularity is in minutes (ex: 5m)")
	}
	probe.Delay = tProbe.Delay.Duration

	if tProbe.Timeout.Duration == 0 {
		//~ return nil, errors.New("invalid or missing 'timeout'")
		tProbe.Timeout.Duration = 20 * time.Second
	}

	if tProbe.Timeout.Duration < (1 * time.Second) {
		return nil, errors.New("'timeout' can't be less than 1 second")
	}
	probe.Timeout = tProbe.Timeout.Duration

	// should warn about dangerous characters? (;& …)
	probe.Arguments = tProbe.Arguments

	probe.Defaults = make(map[string]interface{})
	if err := checkTomlDefault(probe.Defaults, tProbe.Default); err != nil {
		return nil, err
	}

	for index, tCheck := range tProbe.Check {
		var check Check

		check.Index = index

		if tCheck.Desc == "" {
			return nil, errors.New("[[check]] with invalid or missing 'desc'")
		}
		check.Desc = tCheck.Desc

		if tCheck.If == "" {
			return nil, errors.New("[[check]] with invalid or missing 'if'")
		}
		expr, err := govaluate.NewEvaluableExpressionWithFunctions(tCheck.If, CheckFunctions)
		if err != nil {
			return nil, fmt.Errorf("[[check]] invalid 'if' expression: %s (\"%s\")", err, tCheck.If)
		}
		check.If = expr

		if tCheck.Classes == nil {
			return nil, errors.New("no valid 'classes' parameter found")
		}

		if len(tCheck.Classes) == 0 {
			return nil, errors.New("empty classes")
		}
		for _, class := range tCheck.Classes {
			if !IsValidTokenName(class) {
				return nil, fmt.Errorf("invalid class name '%s'", class)
			}
		}
		check.Classes = tCheck.Classes

		if tCheck.NeededFailures == 0 {
			tCheck.NeededFailures = 1
		}
		check.NeededFailures = tCheck.NeededFailures

		if tCheck.NeededSuccesses == 0 {
			tCheck.NeededSuccesses = check.NeededFailures
		}
		check.NeededSuccesses = tCheck.NeededSuccesses

		probe.Checks = append(probe.Checks, &check)
	}

	if miss := probe.MissingDefaults(); len(miss) > 0 {
		return nil, fmt.Errorf("missing defaults (used in 'if' expressions or 'arguments' parameter): %s", strings.Join(miss, ", "))
	}

	return &probe, nil
}
