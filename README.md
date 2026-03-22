# AgentClash

Opensource race engine. Pit your models against each other on real tasks. Same tools, same constraints, scored live — not benchmarks, not vibes.

**[agentclash.dev](https://www.agentclash.dev)**

## What is this?

AgentClash puts AI models on the same real task, at the same time. Scored live on completion, speed, token efficiency, and tool strategy. Step-by-step replays show exactly why one agent won and another didn't.

- Head-to-head races
- Composite scoring
- Full replays
- Failure-to-eval flywheel

## How it works

1. Define a challenge (broken code, a build task, etc.)
2. Drop in your models (OpenAI, Anthropic, Gemini, OpenRouter)
3. Run the race — same tools, same constraints
4. See scored results with full step-by-step replays

## Quick start

```bash
go run ./cmd/race --config race.yaml
```

## Project structure

```
cmd/race/          — CLI entrypoint
config/            — race.yaml loader
internal/engine/   — race orchestrator, agent runner, broadcaster
internal/provider/ — LLM API clients (OpenAI, Anthropic, Gemini, OpenRouter)
internal/scoring/  — composite scoring
internal/telemetry/— trace + step recording
races/             — challenge packs (challenge.yaml + workspace/)
web/               — Next.js landing page
backend/           — API server
```

## License

Open source.
