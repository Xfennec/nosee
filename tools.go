package main

import (
	"regexp"
	"strings"
)

func IsValidTokenName(token string) bool {
	match, _ := regexp.MatchString("^[A-Za-z0-9_]+$", token)
	return match
}

func IsAllUpper(str string) bool {
	return str == strings.ToUpper(str)
}
