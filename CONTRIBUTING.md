# Contributing to restless

## Prerequisites

- Go 1.23+
- `make`
- `git`

## Development Setup

```bash
git clone https://github.com/shahadulhaider/restless.git
cd restless
go mod download
make build
```

## Running Tests

```bash
make test       # run all tests
make vet        # run go vet
make lint       # run staticcheck (requires: go install honnef.co/go/tools/cmd/staticcheck@latest)
```

All tests must pass before submitting a PR.

## Making Changes

1. Fork the repository
2. Create a branch: `git checkout -b feat/your-feature`
3. Make your changes
4. Run `go fmt ./...` and `go vet ./...`
5. Run `make test` — all tests must pass
6. Commit your changes (see commit conventions below)
7. Push to your fork and open a Pull Request

## Commit Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scope): add new feature
fix(scope): fix a bug
docs: update documentation
build: change build config
ci: update CI workflow
chore: maintenance task
test: add or update tests
refactor: code change that neither fixes a bug nor adds a feature
```

Examples from this project:
- `feat(parser): add .http lexer and environment parser`
- `fix(engine): handle redirect loops in cookie jar`
- `docs: update README quickstart example`

## Reporting Issues

Please include:
- Your OS and architecture
- Go version (`go version`)
- restless version (`restless version`)
- Steps to reproduce
- Expected vs actual behavior

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Add tests for new behavior
- Update README if adding user-visible features
- PR title should follow commit conventions
