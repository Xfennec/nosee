package main

import (
	"regexp"
)

func IsValidTokenName(token string) bool {
	match, _ := regexp.MatchString("^[A-Za-z0-9_]+$", token)
	return match
}
