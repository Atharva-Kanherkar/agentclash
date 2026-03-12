---
name: ci
description: Run after each commit to detect CI-relevant entrypoints and surface whether repository checks should be verified or automated.
---

# CI Skill

Use this skill after every commit.

Purpose:

- detect whether the repository has CI-relevant entrypoints
- remind the user to run or verify those checks
- act as the place to grow commit-time automation later

Current behavior:

- if no CI entrypoints are present, skip cleanly
- if CI entrypoints exist, surface that they should be run or automated

This skill is intentionally conservative for now.
