# situ

Go CLI that deploys files, directories and systemd units described in Situfiles, tracks drift, and removes what it installed. Local, single host.

## Source of truth

- `docs/spec.md` (behaviour) and `docs/format.md` (Situfile syntax) define situ. Read the sections relevant to your task before changing behaviour.
- If the code and the spec disagree, or the spec is ambiguous or silent, stop and ask. Never change the spec or invent behaviour silently.
- Work is organised in milestones: `docs/dev/milestones.md`, run with `/milestone <id>`. Never implement what a milestone lists as out of scope.

## Commands

- `make check`: gofmt, go vet, go test. Must pass before any commit.
- `make build`: static binary in `bin/situ`.

## Rules

- Go 1.27, standard library only. Never add a dependency.
- Every external command (`systemctl`, `git`, `etckeeper`), ownership change, user lookup and `sync()` goes through an interface with a fake for tests.
- Errors carry their exit code (0 to 3, see the spec). Only `main` calls `os.Exit`.
- No global mutable state. No hard-coded infrastructure path: defaults live in one place.
- Every Go file starts with `// SPDX-License-Identifier: GPL-3.0-or-later`.
- English everywhere: code, comments, messages, docs.

## Safety

- IMPORTANT: tests and manual runs never touch the real system. Use `t.TempDir()` for the repository and `--root`, and fake external commands. Never run `situ install`, `remove` or `adopt` without `--root` pointing to a temporary directory.
- Never use `sudo`, `systemctl` or `etckeeper` directly.

## Git

- Conventional Commits in English, scope = package or document: `feat(situfile): …`, `test(state): …`, `docs(spec): …`.
- Commit only when the maintainer confirms. Never push, never tag: tags are published to the Go module proxy forever.
