package main

import (
	"strings"
)

type HourRange struct {
	Start [2]int
	End   [2]int
}

type Alert struct {
	Name        string
	Disabled    bool
	Targets     []string
	Script      string
	ScriptCache *strings.Reader
	Arguments   string
	Hours       []HourRange
	Days        []int
}
