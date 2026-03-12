---
name: go
description: Run after each commit when the repository contains Go code to perform lightweight Go-specific hygiene checks.
---

# Go Skill

Use this skill after every commit when the repository contains Go code.

Purpose:

- verify Go-specific hygiene after changes
- run Go checks only when a Go module or Go files are present

Checks:

- detect whether `go.mod` exists
- detect whether any `*.go` files exist
- if both exist and `go` is installed, run `go test ./...`
- otherwise skip without failing the repository workflow

This lets the skill folder stay useful even when the repo is documentation-only.
