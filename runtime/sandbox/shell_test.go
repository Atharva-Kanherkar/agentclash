package sandbox

import (
	"strings"
	"testing"
)

func TestIsShellCommand(t *testing.T) {
	cases := map[string]bool{
		"sh":                 true,
		"bash":               true,
		"ash":                true,
		"zsh":                true,
		"dash":               true,
		"/bin/sh":            true,
		"/usr/bin/bash":      true,
		"env sh -c echo hi":  true,
		"env -i sh -c echo":  true,
		"env bash -lc echo":  true,
		"env SHELL=sh sh -c": true,
		"busybox sh":         true,
		"python3":            false,
		"echo":               false,
		"timeout":            false,
		"env python3":        false,
	}
	for cmd, want := range cases {
		if got := IsShellCommand(strings.Fields(cmd)); got != want {
			t.Errorf("IsShellCommand(%q) = %v, want %v", cmd, got, want)
		}
	}
	if IsShellCommand(nil) {
		t.Error("IsShellCommand(nil) = true")
	}
}
