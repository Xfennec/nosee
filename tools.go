package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
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

func InterfaceValueToString(iv interface{}) string {
	switch iv.(type) {
	case int:
		return fmt.Sprintf("%d", iv.(int))
	case int32:
		return fmt.Sprintf("%d", iv.(int32))
	case int64:
		return strconv.FormatInt(iv.(int64), 10)
	case float32:
		return fmt.Sprintf("%f", iv.(float32))
	case float64:
		return strconv.FormatFloat(iv.(float64), 'f', -1, 64)
	case string:
		return iv.(string)
	case bool:
		return strconv.FormatBool(iv.(bool))
	}
	return "INVALID_TYPE"
}
