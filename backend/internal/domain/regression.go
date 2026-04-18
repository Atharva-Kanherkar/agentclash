package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRegressionSuiteStatus = errors.New("invalid regression suite status")
	ErrInvalidRegressionCaseStatus  = errors.New("invalid regression case status")
	ErrInvalidRegressionSeverity    = errors.New("invalid regression severity")
	ErrInvalidPromotionMode         = errors.New("invalid regression promotion mode")
)

type RegressionSuiteStatus string

const (
	RegressionSuiteStatusActive   RegressionSuiteStatus = "active"
	RegressionSuiteStatusArchived RegressionSuiteStatus = "archived"
)

var regressionSuiteTransitions = map[RegressionSuiteStatus]map[RegressionSuiteStatus]struct{}{
	RegressionSuiteStatusActive: {
		RegressionSuiteStatusArchived: {},
	},
	RegressionSuiteStatusArchived: {
		RegressionSuiteStatusActive: {},
	},
}

func ParseRegressionSuiteStatus(raw string) (RegressionSuiteStatus, error) {
	status := RegressionSuiteStatus(raw)
	if !status.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidRegressionSuiteStatus, raw)
	}
	return status, nil
}

func (s RegressionSuiteStatus) Valid() bool {
	_, ok := regressionSuiteTransitions[s]
	return ok
}

func (s RegressionSuiteStatus) CanTransitionTo(next RegressionSuiteStatus) bool {
	nextStatuses, ok := regressionSuiteTransitions[s]
	if !ok {
		return false
	}
	_, ok = nextStatuses[next]
	return ok
}

type RegressionCaseStatus string

const (
	RegressionCaseStatusActive   RegressionCaseStatus = "active"
	RegressionCaseStatusMuted    RegressionCaseStatus = "muted"
	RegressionCaseStatusArchived RegressionCaseStatus = "archived"
)

var regressionCaseTransitions = map[RegressionCaseStatus]map[RegressionCaseStatus]struct{}{
	RegressionCaseStatusActive: {
		RegressionCaseStatusMuted:    {},
		RegressionCaseStatusArchived: {},
	},
	RegressionCaseStatusMuted: {
		RegressionCaseStatusActive:   {},
		RegressionCaseStatusArchived: {},
	},
	RegressionCaseStatusArchived: {},
}

func ParseRegressionCaseStatus(raw string) (RegressionCaseStatus, error) {
	status := RegressionCaseStatus(raw)
	if !status.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidRegressionCaseStatus, raw)
	}
	return status, nil
}

func (s RegressionCaseStatus) Valid() bool {
	_, ok := regressionCaseTransitions[s]
	return ok
}

func (s RegressionCaseStatus) CanTransitionTo(next RegressionCaseStatus) bool {
	nextStatuses, ok := regressionCaseTransitions[s]
	if !ok {
		return false
	}
	_, ok = nextStatuses[next]
	return ok
}

type RegressionSeverity string

const (
	RegressionSeverityInfo     RegressionSeverity = "info"
	RegressionSeverityWarning  RegressionSeverity = "warning"
	RegressionSeverityBlocking RegressionSeverity = "blocking"
)

func ParseRegressionSeverity(raw string) (RegressionSeverity, error) {
	severity := RegressionSeverity(raw)
	if !severity.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidRegressionSeverity, raw)
	}
	return severity, nil
}

func (s RegressionSeverity) Valid() bool {
	switch s {
	case RegressionSeverityInfo, RegressionSeverityWarning, RegressionSeverityBlocking:
		return true
	default:
		return false
	}
}

type RegressionPromotionMode string

const (
	RegressionPromotionModeFullExecutable RegressionPromotionMode = "full_executable"
	RegressionPromotionModeOutputOnly     RegressionPromotionMode = "output_only"
	RegressionPromotionModeManual         RegressionPromotionMode = "manual"
)

func ParseRegressionPromotionMode(raw string) (RegressionPromotionMode, error) {
	mode := RegressionPromotionMode(raw)
	if !mode.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidPromotionMode, raw)
	}
	return mode, nil
}

func (m RegressionPromotionMode) Valid() bool {
	switch m {
	case RegressionPromotionModeFullExecutable, RegressionPromotionModeOutputOnly, RegressionPromotionModeManual:
		return true
	default:
		return false
	}
}
