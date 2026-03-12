---
name: git-rules
description: Run after each commit to check commit hygiene, diff sanity, and whether the working tree is unexpectedly dirty.
---

# Git Rules Skill

Use this skill after every commit.

Purpose:

- keep commit hygiene consistent
- catch post-commit repository drift
- make sure a commit leaves the repo in a reviewable state

Checks:

- verify the new commit is inspectable and scoped
- check for whitespace or patch-shape issues
- check whether the working tree is unexpectedly dirty after commit
- remind the user to push only intentional changes

This skill is lightweight and should run on every commit.
