package jrex

import (
	"fmt"
	"strconv"
	"strings"
)

type Part string

const (
	Same    Part = "same"
	Major   Part = "major"
	Minor   Part = "minor"
	Release Part = "release"
)

/**
* ToPart
* @param value string
* @return Part, bool
**/
func ToPart(value string) (Part, bool) {
	switch value {
	case "same":
		return Same, true
	case "major":
		return Major, true
	case "minor":
		return Minor, true
	case "release":
		return Release, true
	}
	return "", false
}

/**
* GetVersion
* @param version string, part Part
* @return string
**/
func GetVersion(version string, part Part) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch part {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "release":
		patch++
	default:
		return version
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}
