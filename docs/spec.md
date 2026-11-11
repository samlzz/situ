# situ specification

Version 0.1 (target: situ v0.1.0, 2026-11-01). This document defines what situ does. The Situfile syntax is defined in [format.md](format.md). Every rule here is meant to be covered by at least one automated test (see [Tests](#tests)).

**SITU Installs, Tracks & Uninstalls.** situ deploys files, directories and systemd units described in Situfiles from a git repository to the Linux host it runs on, detects drift, and removes what it installed. It is a small, local, single-host tool: it never connects to other machines.

## Contents

1. [Concepts](#concepts)
2. [Repository](#repository)
3. [Resolution](#resolution)
4. [State](#state)
5. [Comparing source, host and state](#comparing-source-host-and-state)
6. [Units](#units)
7. [Commands](#commands)
8. [Command line](#command-line)
9. [Help topics](#help-topics)
10. [Preconditions](#preconditions)
11. [Exit codes](#exit-codes)
12. [Implementation constraints](#implementation-constraints)
13. [Tests](#tests)
14. [Out of scope for v0.1](#out-of-scope-for-v01)

## Concepts

| Term | Meaning |
| --- | --- |
| Module | A directory holding a Situfile: a component or a host profile. Every entry on the host belongs to exactly one module. |
| Component | A module at `components/<category>/<component>/`, with its Situfile and source files. |
| Host profile | A module at `hosts/<host>/`, with its Situfile and host-owned source files. It behaves exactly like a component, except that nothing can require it. |
| Situfile | The manifest of a module, see [format.md](format.md). |
| Unit | A systemd unit, and nothing else. |
| Source (S) | What the repository says. |
| Host (H) | What is on the host. |
| Record (E) | What situ recorded at its last successful install, in the state. |
| Drift | Any difference between source and host. |

"Module" always means a situ module. The Go module that implements situ is always called "Go module".

Commands and their direction:

| Command | Direction | Writes to |
| --- | --- | --- |
| `install` | repository -> host | host, state |
| `check` | compares, reports | nothing |
| `adopt` | host -> repository | repository |
| `remove` | uninstalls | host, state |
| `list` | repository -> screen | nothing |

## Repository

### Layout

```
<repo>/
  components/<category>/<component>/Situfile
  hosts/<host>/Situfile
```

- Category, component and host names match `[a-z0-9][a-z0-9-]*`.
- Every directory at `components/<category>/<component>/` and `hosts/<host>/` must contain a `Situfile`. A directory at that depth without one is an error (code 2), reported only when situ needs to scan that level.
- Other files and directories of the repository are ignored.

### Finding the repository

1. With `-C <dir>`, the repository is `<dir>`. It must contain `components/` and `hosts/`, otherwise error (code 2).
2. Otherwise, situ walks up from the current directory to the first directory that contains both `components/` and `hosts/`. None found: error (code 2).

### Naming modules on the command line

| Form | Designates |
| --- | --- |
| `<category>/<component>` | that component |
| `<component>` | the only component of that name in any category; ambiguous or missing: error (code 2) |
| `hosts/<host>` | that host profile |

## Resolution

### The current profile

The current host name is the result of `os.Hostname()` up to the first `.`. The current profile is `hosts/<name>/`. There is no option to choose another profile, except as an explicit target of `list`.

### Variables

`${HOST}` is the name of the host profile the module is resolved for. A destination is expanded once per profile.

### The graph

Starting from a set of root modules, situ follows `requires` transitively. The result is the graph of the run.

Errors (code 2):

- a `requires` target that does not exist, or is ambiguous;
- a dependency cycle (the message lists the cycle);
- a host profile used as a `requires` target (also a format error);
- two modules of the graph declaring the same expanded destination, whatever their roles;
- two modules of the graph declaring the same unit, whatever their roles (the same unit in `enable` and `start` of one module is allowed).

Order: modules are processed in a topological order, dependencies first. Ties are broken by module name (`components/...` sorted lexically, then `hosts/...`), so runs are deterministic.

### Parents of destinations

The parent directory of every destination must either exist on the host as a directory, or be the destination of a `dir` or `data` entry of the graph. Otherwise the plan fails with code 2 (the Situfile is incomplete). situ never creates undeclared parent directories.

### Order of processing

The order of statements in a Situfile never changes what situ does (see [format.md](format.md)). Statements are processed by kind, always in the same order: `requires`, then entries (`dir`, `data`, `file`, `config`), then units (`enable`, `start`).

Reading (every command). Each stage starts only when the previous one has succeeded:

1. Each Situfile is read and checked on its own (every rule of [format.md](format.md) that needs no other module); all its errors are reported together, sorted by line.
2. `requires` build the graph of the run (see [The graph](#the-graph)): missing targets, ambiguity and cycles.
3. Entries: destinations are expanded for the profile, sources are checked, then duplicate destinations across modules and the [parents](#parents-of-destinations) are checked.
4. Units: duplicate units across modules are checked.

Applying (`install`), see [`install`](#install---force-module):

- Between modules: topological order, dependencies first, in every phase.
- Between kinds, inside the run: orphan teardown, then `dir` and `data`, then `file` and `config`, then `daemon-reload`, then `enable`, then `start`. A phase is finished for every module before the next one starts.
- Inside one module and one phase: the order of the Situfile, except that a `dir` or `data` entry always comes after the entry that creates its parent.

`remove` uses the exact reverse order (see [`remove`](#remove--r---purge-module)): modules in reverse topological order, units before entries, and inside a module, the reverse of the Situfile order.

## State

### Location

`<root>/var/lib/situ/`, mode `0700`, owner `root:root`. One state file per installed module, mirroring the repository layout:

```
/var/lib/situ/components/<category>/<component>/Situfile
/var/lib/situ/hosts/<host>/Situfile
/var/lib/situ/lock
```

State files have mode `0600`.

### Content

A state file is the Situfile of the module as installed, resolved, with one extra column. It uses the lexical rules of [format.md](format.md) with these differences:

- A `commit <sha>` statement follows `format`: the repository commit installed (40 hex characters).
- `requires` targets are written in full form (`<category>/<component>`).
- Destinations are written expanded: no `${...}` remains.
- `file`, `config`, `dir` and `data` lines have a sixth field: the lowercase hex SHA-256 of the content written, for `config` only; `-` for the other roles.
- `enable` and `start` lines are unchanged.

Example:

```
format    1
commit    3f2a9c1e5b7d4e8a9c217d4e5f6a8b9001234567
requires  backup/backup-core

# role    mode  owner      source               destination                      sha256
file      0755  root:root  etc-bundle           /usr/local/sbin/etc-bundle       -
file      0644  root:root  etc-bundle.service   /etc/systemd/system/etc-bundle.service  -
config    0644  root:root  etc-bundle.default   /etc/default/etc-bundle          9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
dir       0700  root:root  -                    /etc/etc-bundle                  -
data      0700  root:root  -                    /var/backups/pve-xps/etc         -
enable    etc-bundle.service
```

### Writing

- A state file is written only after every phase of the run has succeeded, for each module of the run.
- Before writing any state, situ calls `sync()` once, so that everything the run changed on the host is on disk before the state claims it. Without it, a crash could keep the new state but lose the rename of an installed file, and the next run would wrongly report it as modified on the host.
- Writing is atomic: temporary file in the same directory, `fsync`, rename, `fsync` of the directory.
- A run that fails part way writes no state. The next run recovers through the "converged" and "no record" rules below.

### Missing or broken state

- A missing state file means "no record" for every entry of that module. This is how situ takes over modules installed by hand: identical files are recorded, different ones are reported, nothing is overwritten.
- A state file that cannot be parsed is an error (code 3). The message tells the user that deleting the file makes situ rebuild it.

### Lock

`install` and `remove` take an exclusive `flock` on `<root>/var/lib/situ/lock` for their whole run. If another situ holds it, they fail at once (code 3).

## Comparing source, host and state

### `config`

S, H and E are SHA-256 digests: S of the source file, H of the host file (or absent), E of the record (or none).

| # | Record | Host | Source | Situation | `install` | `check` status |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | any | absent | — | missing | create | `missing` |
| 2 | E | H = E | S = E | up to date | nothing | `ok` |
| 3 | E | H = E | S ≠ E | repository ahead | update | `pending` |
| 4 | E | H ≠ E, H = S | — | converged | record only | `unrecorded` |
| 5 | E | H ≠ E | S = E | modified on host | **blocking** | `modified` |
| 6 | E | H ≠ E, H ≠ S | S ≠ E | conflict | **blocking** | `conflict` |
| 7 | none | H = S | — | identical, not recorded | record only | `unrecorded` |
| 8 | none | H ≠ S | — | differs, not recorded | **blocking** | `unknown` |

`install --force <module>` turns cases 5, 6 and 8 of that module into "overwrite with the source".

### `file`

| Host | `install` | `check` status |
| --- | --- | --- |
| absent | create | `missing` |
| H = S | nothing (record if no record) | `ok` |
| H ≠ S | overwrite | `differs` |

A `file` is owned by the repository: it is never blocking and no digest is recorded.

### `dir` and `data`

| Host | `install` | `check` status |
| --- | --- | --- |
| absent | create with mode and owner | `missing` |
| directory | nothing | `ok` |

For `data`, only the directory itself is checked and fixed, never its content.

### Mode and owner

For every role with a mode and owner, a host mode or owner different from the Situfile is fixed by `install` (`chmod`, `chown`) and reported by `check` as `mode` or `owner`. The repository always wins. `adopt` never copies modes or owners.

### Type mismatch

If a destination exists with the wrong type (a directory where a file is expected, a file where a directory is expected, or a symbolic link for any role), the entry is **blocking** with status `type`. `--force` does not override it: situ never deletes a directory or follows a link to make room. The user fixes it by hand.

### Orphans

An orphan is an entry present in the state file of a module but absent from its current Situfile. `install` removes the orphans of the modules in its run, before applying the rest (teardown runs in the order of `remove`):

| Role | `install` on an orphan | `check` status |
| --- | --- | --- |
| `file` | delete | `orphan` |
| `config` | keep on the host, drop from the state | `orphan` |
| `dir` | delete if empty, otherwise keep and report | `orphan` |
| `data` | keep, drop from the state, report the path | `orphan` |
| `enable` | disable | `orphan` |
| `start` | stop | `orphan` |

### Modules out of the graph

- A module with a state file but not reachable from the current profile is `unreachable`. `check` and `install` report it with the command to remove it (`situ remove <module>`). Nothing removes it automatically.
- A module of the graph without a state file and with every entry missing is `not-installed` for `check`.

## Units

### Unit directories

A path is a unit path if it is under `/etc/systemd/system/` (after the root prefix). v0.1 knows only this directory.

### `enable`

| `systemctl is-enabled` | `install` | `check` status |
| --- | --- | --- |
| `enabled` | nothing, or `reenable` if the module changed | `ok` |
| `disabled` | `enable` | `disabled` |
| anything else (`masked`, `static`, `not-found`…) | **blocking** | the reported value |

### `start`

| `systemctl is-active` | `install` | `check` status |
| --- | --- | --- |
| `active`, `activating`, `reloading` | nothing, or `restart` if the module changed | `ok` |
| `inactive`, `failed` | `start` | the reported value |

### A module "changed"

A module changed during a run if any of its entries was created, updated, overwritten, deleted, or had its mode or owner fixed. In that case its `enable` units are re-enabled (`systemctl reenable`) and its `start` units restarted (`systemctl restart`), even if they were already enabled and active. `reenable` is needed because a changed `[Install]` section leaves stale links with a plain `enable`.

### `daemon-reload`

Run once after the file phase of `install`, and once after the file phase of `remove`, if at least one unit path was created, updated, overwritten or deleted during that phase.

## Commands

All commands that touch the host first compute a **plan** (the classification of every entry), then apply it only if nothing in it is blocking. A blocked plan changes nothing and exits with code 1, after printing every blocking entry with the commands that would resolve it.

### `install [--force] [<module>...]`

Scope:

- Without modules: the graph of the current profile.
- With modules: the graph of those modules. Each must be reachable from the current profile, otherwise error (code 2).
- `--force` requires at least one module. It only affects the blocking `config` cases of the named modules (not their dependencies).

Steps:

1. Preconditions (see [Preconditions](#preconditions)).
2. Plan. Blocking entries: stop with code 1.
3. Apply, in phases. Within a phase, modules are taken in topological order (dependencies first):
   1. orphan teardown, in the order of `remove`;
   2. `dir` and `data`: create, fix mode and owner, parents before children;
   3. `file` and `config`: write, fix mode and owner;
   4. `daemon-reload` if needed;
   5. `enable` (or `reenable`);
   6. `start` (or `restart`).
4. `sync()`, then write the state of every module of the run.
5. If `/etc` is managed by etckeeper and changed: `etckeeper commit "situ: install <modules or profile>"`. This also happens after a failure part way, with `(incomplete)` appended, so that `/etc` is never left unclean.
6. Report `unreachable` modules, if any (whole-profile runs only).

Writing a file: temporary file in the destination directory, content, `chmod`, `chown`, `fsync`, rename. A destination is never written through a symbolic link.

Output: one line per action (`create`, `update`, `overwrite`, `record`, `chmod`, `chown`, `delete`, `keep`, `enable`, `reenable`, `start`, `restart`, `disable`, `stop`, `daemon-reload`), then `nothing to do` or a summary. Exit 0 on success, including when nothing had to be done.

### `check [<module>...]`

Scope: like `install`, without `--force`. No preconditions except reading the repository and the host. A clone with uncommitted changes is allowed.

Output, one block per module, then unreachable modules:

```
components/backup/backup-core
  ok
components/backup/etc-bundle
  modified  config  /etc/default/etc-bundle
            -> situ adopt /etc/default/etc-bundle
            -> situ install --force backup/etc-bundle
hosts/pve-xps
  ok
```

Suggested commands per status:

| Status | Suggestions |
| --- | --- |
| `missing`, `pending`, `unrecorded`, `orphan`, `mode`, `owner`, `disabled`, `inactive`, `failed`, `not-installed` | `situ install` |
| `differs` | `situ install`, `situ adopt <path>` |
| `modified`, `conflict`, `unknown` | `situ adopt <path>`, `situ install --force <module>` |
| `type`, `masked`, other unit states | fix by hand |
| `unreachable` | `situ remove <module>` |

Exit 0 if every entry is `ok` and nothing is unreachable, 1 otherwise.

### `adopt <path|module>...`

Copies host content back into the repository.

- A path designates the `file` or `config` entry of the current graph with that expanded destination. Another role, or no entry: error (code 2).
- A module designates every `file` and `config` entry of that module whose host content differs from its source. None: prints `nothing to adopt`, exit 0.
- For each entry, the host content replaces the source file content in the repository. The source file keeps its own mode in the repository. Modes and owners on the host are never adopted.
- Refused (code 1) if a source file to overwrite has uncommitted changes in the repository, or if the host file is missing or has the wrong type.
- situ never commits. After adopting, it prints the files changed and `git -C <repo> commit` as a reminder. It does not write the state: the next `install` records the entries (case 4 or 7).

### `remove [-r] [--purge] <module>...`

Uninstalls. Works from the state only: it does not need the repository content, only the module names.

- At least one module is required. Names are resolved against the installed modules (state), with the same short-name rules.
- Refused (code 1) if an installed module outside the set of names requires one of them. The message lists which module requires which.
- `-r`, `--recursive`: also remove, transitively, the dependencies of the named modules that no installed module outside the removed set still requires.
- `--purge`: also delete `config` files, even if modified on the host.
- There is no `--force`.

Phases, modules in reverse topological order within each phase:

1. `stop` the `start` units;
2. `disable` the `enable` units;
3. delete `file` entries; delete `config` entries with `--purge`, otherwise keep them;
4. `daemon-reload` if needed;
5. delete `dir` entries, children before parents, only if empty; a non-empty one is kept and reported;
6. `data` entries are never touched; their paths are printed;
7. `sync()`, then delete the state files;
8. etckeeper commit as for `install` (`situ: remove <modules>`).

A missing host file or unit is not an error during `remove`. A `remove` that fails part way deletes no state file: running it again completes it.

### `list [<module>|<path>]`

Reads the repository only. Never reads the host or the state, never needs root.

- Without argument: the graph of the current profile. Prints the dependency tree, then the entries of each module, expanded. No current profile: error (code 2) suggesting `situ list hosts/<host>`.
- With a module: that module, its `requires` and its entries. Expanded with the named profile, or with the current profile if there is one; otherwise `${HOST}` is printed unexpanded.
- With a path: every Situfile line whose destination, expanded for a profile of `hosts/`, is that path. The search covers the graph of every profile. Each match is printed like `grep -n`: the Situfile path relative to the repository, the line number, the line as written in the Situfile (variables unexpanded, whitespace normalized to two spaces), then the profiles it was matched for, in parentheses. A line matched for several profiles is printed once, with all of them. Exit 1 if nothing matches.

```
$ situ list /etc/default/etc-bundle
components/backup/etc-bundle/Situfile:7: config  0644  root:root  etc-bundle.default  /etc/default/etc-bundle  (hosts/pve-xps)

$ situ list /var/backups/pve-xps/etc
components/backup/etc-bundle/Situfile:9: data  0700  root:root  -  /var/backups/${HOST}/etc  (hosts/pve-xps)
```

### `version`

Prints `situ <version>`, where the version is `Main.Version` from `runtime/debug.ReadBuildInfo()`. A build that is not from a tagged Go module version (`(devel)`) also prints the `vcs.revision` if available. `situ --version` is an exact alias.

### `help [<topic>]`

- `situ help` prints the synopsis, the list of commands and the list of help topics.
- `situ help <topic>` prints that topic (see [Help topics](#help-topics)).
- An unknown topic is an error (code 2). If the topic is a command name, the message points to `situ <command> --help`.

## Command line

```
situ [-C <dir>] [--root <dir>] <command> [options] [targets...]
situ --help | --version

  install  [--force] [<module>...]
  check    [<module>...]
  adopt    <path|module>...
  remove   [-r] [--purge] <module>...
  list     [<module>|<path>]
  version
  help     [<topic>]
```

- Global options come before the command. Command options come after it.
- `-C <dir>` sets the repository, as in `git -C`.
- `--root <dir>` (default `/`): every host path (destinations, state, `/etc`) is resolved under `<dir>`. With a root other than `/`:
  - etckeeper is ignored;
  - `systemctl --root=<dir>` is used for `enable`, `disable`, `reenable` and `is-enabled`;
  - `daemon-reload`, `start`, `stop`, `restart` and `is-active` are skipped and reported as skipped;
  - user and group names are resolved from `<dir>/etc/passwd` and `<dir>/etc/group`.
- A target starting with `/` is a path; anything else is a module name. Paths are accepted by `adopt` and `list` only.

### Help and version options

- `--help` is accepted everywhere: `situ --help` prints the same as `situ help`; `situ <command> --help` prints the usage and options of that command. Output goes to stdout, exit 0, and nothing else is done.
- `--version` is accepted before the command only.
- `-h` is not an option. It is rejected with `situ: unknown option -h (use --help)`, code 2, so that the letter stays free for a future option. Go's `flag` package accepts `-h` by default: the implementation must reject it explicitly.
- A usage error (unknown command or option, missing argument, `situ` without arguments) prints a short usage to stderr and exits with code 2.

### Output

- English plain text, without color. Results go to stdout, errors to stderr.
- No environment variable changes situ's behaviour in v0.1. The prefix `SITU_` is reserved.

## Help topics

The text of each topic is compiled into the binary, so that it is available on hosts where the situ documentation is not.

| Topic | Content |
| --- | --- |
| `situfile` | A summary of [format.md](format.md): statements, roles, fields, variables. |
| `status` | Every status `check` can print, its meaning and the suggested commands (the tables of [Comparing source, host and state](#comparing-source-host-and-state), [Units](#units) and [`check`](#check-module)). |
| `exit-codes` | The table of [Exit codes](#exit-codes). |

The topics must stay consistent with this document: a test compares the list of statuses in the `status` topic with the statuses the code can produce.

## Preconditions

| Check | `install` | `remove` | `adopt` | `check` | `list` |
| --- | --- | --- | --- | --- | --- |
| Effective user is root (when `--root` is `/`) | yes, code 2 | yes, code 2 | — | — | — |
| `/etc` clean (`etckeeper unclean` false), if `/etc` is managed by etckeeper | yes, code 1 | yes, code 1 | — | — | — |
| Repository is a git work tree with no uncommitted or untracked changes (`git status --porcelain` empty) | yes, code 1 | — | — | — | — |
| Lock | yes | yes | — | — | — |

`/etc` is managed by etckeeper if `/etc/.etckeeper` exists and the `etckeeper` command is available.

Commands that read the host without being root fail with code 3 on the first permission error.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success, conforming, nothing to do, or help printed. |
| 1 | Drift or conflict reported (`check`), operation refused because an action is needed from the user (`install`, `remove`, `adopt`), path not found (`list`). |
| 2 | User error: wrong usage, unknown command, option or help topic, invalid Situfile, invalid repository, unknown module, missing profile, not root. |
| 3 | System error: I/O error, permission denied, `systemctl` or `git` failure, broken state file, lock held. |

## Implementation constraints

- Go module `github.com/samlzz/situ`, Go 1.27, binary in `cmd/situ`.
- Standard library only. Builds with `CGO_ENABLED=0` into a static binary.
- Every external command (`systemctl`, `git`, `etckeeper`) goes through a Go interface, with a fake implementation for tests. No external command is run directly from the core packages.
- Host file operations go through the root prefix. Ownership changes, user lookups and `sync()` go through an interface, so tests run as an unprivileged user and can check the call order.
- No hard-coded path of any particular infrastructure. The defaults (`/`, `/var/lib/situ`, `/etc/systemd/system`) are constants in one place.
- Errors carry their exit code; `main` is the only place that calls `os.Exit`.
- No global mutable state.

## Tests

`go test ./...`, `go vet ./...` and `gofmt -l .` (empty output) must pass. Tests never require root and never touch the real system: they use `t.TempDir()` as root and repository, and fake external commands.

Required cases, at minimum:

1. **Parser**: every rule of [format.md](format.md), valid and invalid, with the expected `path:line` message and code 2.
2. **Repository**: discovery with and without `-C`, directory without Situfile, short names (unique, ambiguous, missing).
3. **Graph**: transitive `requires`, cycle, missing target, profile as target, duplicate destination across modules, duplicate unit across modules, deterministic order.
4. **Parents**: existing parent, parent declared by `dir`, undeclared missing parent (code 2).
5. **State**: write then read gives the same model; broken file (code 3); atomic write leaves no temporary file.
6. **Reference table**: each of the 8 `config` cases, for both `check` (status, exit code) and `install` (action, resulting state); `--force` on cases 5, 6 and 8, and on a dependency (no effect).
7. **`file`, `dir`, `data`**: each row of their tables; mode and owner fixes; type mismatches (blocking, `--force` ignored); symbolic link at a destination.
8. **Orphans**: each role.
9. **Units**: each row of the `enable` and `start` tables; `reenable` and `restart` only when the module changed; `daemon-reload` only when a unit path changed; call order recorded by the fake follows the phases, and inside a phase the Situfile order (reversed for `remove`).
10. **Plan**: one blocking entry anywhere means nothing is applied.
11. **Failure part way**: an apply error leaves no state written, and the next run converges. `sync()` (through the fake) is called after the last host change and before the first state write, for both `install` and `remove`.
12. **`remove`**: refusal when required, several modules at once, `-r` keeping a dependency still required elsewhere, `--purge`, non-empty `dir` kept, `data` untouched, works with the repository absent.
13. **`adopt`**: by path and by module, refused on a dirty source, refused on a missing host file, never touches modes.
14. **`list`**: the three forms; unexpanded `${HOST}`; path search printing `Situfile:line:` with the line as written and the profiles; one line matched for two profiles printed once; path not found (exit 1).
15. **Preconditions**: not root, `/etc` unclean, dirty repository, lock held; etckeeper commit after success and after a partial failure; `remove` failing part way, then completed by a second run.
16. **Command line**: `--help` at top level and for each command (stdout, exit 0, nothing done); `--version` and `version`; `-h` rejected (code 2); `help` with each topic, an unknown topic, and a command name as topic; usage errors (code 2, stderr); `situ` without arguments.
17. **Help topics**: the statuses listed in the `status` topic are exactly the statuses the code can produce.
18. **End to end**: a fixture repository reproducing `backup-core`, `etc-bundle` and a `hosts/<host>` profile, installed under a temporary root: first install from nothing; second install does nothing; host `config` edited then `check` (exit 1), `adopt`, commit, `install` (records); component removed from the profile then `check` reports it `unreachable`; `remove -r hosts/<host>` leaves only `data` directories and kept `config` files.
19. **Takeover**: files already present and identical under the root, no state: `install` records them without writing them.

## Out of scope for v0.1

- Machine-readable output for `check` and `list` (planned option, name to be decided).
- Confirmation prompts (`-y`).
- An options column in the Situfile.
- Unit directories other than `/etc/systemd/system` (e.g. LibreELEC's `/storage/.config/system.d`).
- Other architectures than the one situ is built for: the build is the user's concern.
- Remote hosts, APIs (Proxmox, TrueNAS), templates, secrets.
