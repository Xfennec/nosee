package main

import (
	"crypto/md5"
	"encoding/hex"
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

func MD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
