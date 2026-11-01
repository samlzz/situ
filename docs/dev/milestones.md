# Development milestones

situ v0.1.0 is built in six milestones. Each one ends with `make check` passing, a review, and a commit. Read-only commands come first, so that `situ list` and `situ check` can run on a real host long before situ writes anything there.

## How to run a milestone

One Claude Code session per milestone, started from a clean working tree.

1. Start Claude Code at the effort level of the milestone: `claude --effort medium` or `claude --effort high`. The project settings select `opusplan`: Opus plans, Sonnet implements.
2. Enter plan mode (`Shift+Tab` until the status bar shows plan mode).
3. Run `/milestone M<n>`.
4. Read the plan. Answer its questions. Edit it with `Ctrl+G` if needed, or ask for changes.
5. Approve with the option that clears the planning context. Sonnet implements, runs `make check` and the verification steps, and reports.
6. Run `/code-review` for an independent review of the diff. Fix what affects correctness or the spec, ignore style-only remarks.
7. Confirm the commit. Push yourself.
8. Do the maintainer steps of the milestone, if any, then `/clear` or quit.

If a session goes wrong (two corrections on the same point without progress), stop, `/clear`, and restart the milestone with a more precise instruction.

## M1: scaffold, command line, Situfile parser

Effort: `medium`.

Read: `docs/format.md` (all); `docs/spec.md`: Command line, Help topics (structure only), Exit codes, Implementation constraints.

Do:

- Go module `github.com/samlzz/situ` (`go mod init`), `cmd/situ/main.go`, internal packages.
- An error type carrying the exit code; `main` maps it to `os.Exit`.
- Command dispatch for every command of the synopsis. Commands not implemented yet exit 3 with `not implemented`.
- Global options `-C` and `--root`, `--help` everywhere, `--version`, `version`, `-h` rejected, usage errors, `situ` without arguments.
- `help` with the list of commands and topics; topics `situfile` and `exit-codes` written; `status` present as a stub.
- The Situfile parser: every rule of `docs/format.md`, all errors as `<path>:<line>: <message>`, code 2. Variable expansion as a separate function taking the variables. Source file checks (exists, regular file, not a link) relative to the module directory.

Out of scope: repository discovery, graph, host access, state.

Tests: spec cases 1 and 16 (the parts that exist: `--help`, `--version`, `-h`, usage errors, `help` topics).

Verify: `make check`; `make build`; `bin/situ --help`, `bin/situ --version`, `bin/situ -h` (exit 2), `bin/situ help situfile`.

## M2: repository, resolution, `list`

Effort: `medium`.

Read: `docs/spec.md`: Concepts, Repository, Resolution, `list`.

Do:

- Repository discovery (`-C`, walk up), module names (full and short forms), directories without Situfile.
- Current profile from the host name; `${HOST}` per profile.
- The graph: transitive `requires`, cycles, missing and ambiguous targets, profile as target, duplicate destinations and units across modules, deterministic topological order.
- Parent directory rule (reads the host under `--root`, read-only).
- `list` in its three forms, with the `Situfile:line:` output for paths.

Out of scope: state, comparison, any write to the host.

Tests: spec cases 2, 3, 4, 14.

Verify: `make check`; a fixture repository in `testdata/` reproducing `backup-core`, `etc-bundle` and a `hosts/pve-xps` profile; `go run ./cmd/situ -C testdata/<fixture> list hosts/pve-xps` and `list /etc/default/etc-bundle`.

Maintainer: on the laptop, `go run ./cmd/situ -C ~/path/to/hl-infra list hosts/pve-xps`, then a path search. Compare with the real Situfiles.

## M3: state, comparison, `check`

Effort: `high`.

Read: `docs/spec.md`: State, Comparing source, host and state, Units, `check`, Help topics (`status`), Preconditions (`check` column), Implementation constraints.

Do:

- Interfaces and fakes for `systemctl`, `git`, `etckeeper`, ownership, user lookup and `sync()`. Real implementations behind them (read-only uses only in this milestone).
- User and group resolution from `<root>/etc/passwd` and `<root>/etc/group` when `--root` is not `/`.
- State file reading and writing (atomic, `fsync`), as a library: `check` only reads it, but the writer and its tests belong here.
- Classification of every entry for every role, including the 8 `config` cases, mode and owner, type mismatches, orphans, unit states, `unreachable` and `not-installed` modules.
- `check` output, suggested commands and exit code. The `status` help topic, generated from or checked against the statuses the code can produce.

Out of scope: any write to the host.

Tests: spec case 5; the `check` side of cases 6, 7, 8 and 9; case 17.

Verify: `make check`; under a temporary root built from the fixture: `check` before any install (`missing` everywhere, exit 1); files copied by hand then `check` (`unrecorded`); one `config` edited (`unknown`).

Maintainer: build a static binary (`make build`), copy it by hand to `<clone>/bin/situ` on pve-xps, and run `situ check` there. No state exists yet: expect `unrecorded` for identical files, and investigate any `unknown`, `type` or unit status before M4.

## M4: `install`

Effort: `high`.

Read: `docs/spec.md`: State (Writing, Lock), Comparing source, host and state, Units, Commands (introduction and `install`), Command line (`--root`), Preconditions, Exit codes.

Do:

- Preconditions: root (when `--root` is `/`), etckeeper clean, clean git work tree, lock.
- Plan with blocking entries; nothing applied if anything blocks; `--force` rules.
- Apply phases in order: orphan teardown, `dir`/`data`, `file`/`config`, `daemon-reload`, `enable`/`reenable`, `start`/`restart`. Atomic file writes, never through links.
- "Module changed" tracking, `daemon-reload` only for unit paths.
- `sync()` then state writing; no state after a failure; etckeeper commit, including `(incomplete)`.
- `--root` other than `/`: `systemctl --root=`, skipped runtime operations reported.
- Report `unreachable` modules.

Out of scope: `remove`, `adopt`.

Tests: the `install` side of cases 6, 7, 8 and 9; cases 10, 11, 15 (except `remove`) and 19.

Verify: `make check`; under a temporary root: install from nothing, then a second install prints `nothing to do`; takeover of identical files records them without writing.

Maintainer: copy the new binary to pve-xps. Run `situ check`, then `situ install`. On the first run, it must only `record` (takeover): if the plan shows any `create`, `update` or `overwrite` you did not expect, it will not stop by itself, so read `situ check` first. Then `situ check` must exit 0, and `git -C /etc log -1` must show the etckeeper commit.

## M5: `remove`, `adopt`, end to end

Effort: `high`.

Read: `docs/spec.md`: `remove`, `adopt`, Preconditions, Tests (cases 12, 13, 18).

Do:

- `remove` from the state only: name resolution against installed modules, dependency refusal, several modules at once, `-r`, `--purge`, phases in reverse order, `sync()` before deleting state files, etckeeper commit.
- `adopt` by path and by module: refusal on a dirty source or a missing host file, content only, reminder to commit.

Tests: cases 12, 13, the `remove` part of 15, and the end-to-end case 18.

Verify: `make check`; the end-to-end test passes; its log is shown.

## M6: release v0.1.0

Effort: `medium`.

Read: `README.md`, `docs/spec.md` (Out of scope), this file.

Do:

- Review the whole spec against the code: list every deviation, fix or report it.
- README: status, install instructions, examples checked against the real output.
- Check the static build: `make build`, then `file bin/situ` and `ldd bin/situ` (must report a static binary).
- `situ version` from a build inside the repository prints `(devel)` and the revision.

Maintainer (outside Claude Code):

1. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`. The tag is in the Go module proxy forever once fetched.
2. In hl-infra: `tools/situ.version` holds `v0.1.0`; run `tools/deploy-situ pve-xps`. It prints `situ v0.1.0` from the host.
3. On pve-xps: `situ check` exits 0.
