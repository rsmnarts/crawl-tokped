package main

import (
	"regexp"
	"strings"
)

func extractNumberStr(s, sep string) string {
	re := regexp.MustCompile("[0-9]+")
	match := re.FindAllString(s, -1)
	return strings.Join(match, sep)
}
