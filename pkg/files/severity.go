package files

import (
	"strings"

	"github.com/pkg/errors"
)

type Severity uint

const (
	UNDEFINED Severity = iota
	MINOR
	MAJOR
	CRITICAL
)

const total = 3

func AllFileSeverities() [total]Severity {
	return [total]Severity{MINOR, MAJOR, CRITICAL}
}

func SeverityFromString(s string) (Severity, error) {
	switch {
	case strings.ToUpper(s) == "MINOR":
		return MINOR, nil
	case strings.ToUpper(s) == "MAJOR":
		return MAJOR, nil
	case strings.ToUpper(s) == "CRITICAL":
		return CRITICAL, nil
	}
	return UNDEFINED, errors.Errorf("unknown severity: %s", s)
}

func (s Severity) String() string {
	switch s {
	case MINOR:
		return "MINOR"
	case MAJOR:
		return "MAJOR"
	case CRITICAL:
		return "CRITICAL"
	}
	return "unknown"
}
