# Restless Plan — Decisions

## Architecture
- `.http` format: JetBrains spec (NOT Bruno .bru, NOT Hurl)
- Auth: static tokens only (Bearer, API keys, Basic via env vars)
- No response handler scripts, no VS Code extensions, no OAuth2
- History stored in `.restless/history/` (gitignored)
- Cookies: in-memory per environment only, not persisted

## Variable Resolution Priority
1. Chain context variables
2. Environment variables
3. Dynamic variables ($uuid, $timestamp, $randomInt)
