# SITU

**SITU Installs, Tracks & Uninstalls.**

`situ` keeps configuration files *in situ*: in their place. It deploys the files, directories and systemd units described in a git repository to the Linux host it runs on, tells you when something on the host has drifted from the repository, and removes cleanly what it installed.

It is a small, local tool for a personal homelab: one static binary, no daemon, no agent, no remote connection, plain-text manifests.

> **Status:** in development, towards v0.1.0. Personal tool, no warranty: try it with `--root` on a scratch directory first.
>
> **Vibe-coded:** the code is written by Claude Code from a specification I wrote. Automated tests check it against that specification, and I read it as I learn Go, without mastering every line.

## Why SITU

I needed a simple way to deploy hand-written configuration from a git repository to the machines of my homelab: copies rather than links, with drift detection and clean removal. No existing tool fit: GNU Stow and Nix system-manager rely on symbolic links, chezmoi is designed for the home directory, Ansible is heavy for a single host and has no simple way to take a change made on the host back into the repository or to uninstall, and a `.deb` package is a lot of machinery for a component of one or two files, unusable anyway on some targets such as LibreELEC.

I needed it quickly, so this first version is written by AI. Whether I take it over by hand later will depend on how it holds up in daily use. Meanwhile, it is how I learn Go, and how a deployment tool works from the inside.

## How it works

A repository holds **modules**: components and host profiles. Each module is a directory with a `Situfile` that says where things must sit:

```
format    1
requires  backup-core

# role    mode  owner      source               destination
file      0755  root:root  etc-bundle           /usr/local/sbin/etc-bundle
file      0644  root:root  etc-bundle.service   /etc/systemd/system/etc-bundle.service
config    0644  root:root  etc-bundle.default   /etc/default/etc-bundle
dir       0700  root:root  -                    /etc/etc-bundle
data      0700  root:root  -                    /var/backups/${HOST}/etc
enable    etc-bundle.service
```

```
<repo>/
  components/<category>/<component>/Situfile   what a component installs
  hosts/<host>/Situfile                         what a host gets: requires components, adds host-owned files
```

On a host, situ picks `hosts/$(hostname -s)/`, follows `requires`, and compares three things for every entry: the repository, the host, and what it recorded at its last install. That is how it tells "the repository changed" from "someone edited the file on the host", and never overwrites a host change silently.

## Commands

| Command | What it does |
| --- | --- |
| `situ install` | Brings the host in line with the repository. Refuses, without changing anything, if a protected file was modified on the host. |
| `situ check` | Reports drift and suggests the command that fixes each case. Changes nothing. |
| `situ adopt <path>` | Copies a host change back into the repository, when the host is right. |
| `situ remove <module>` | Uninstalls a module. Never touches data directories. |
| `situ list [<module>\|<path>]` | Shows what the repository declares, or which Situfile owns a path. |
| `situ help <topic>` | Offline help: `situfile`, `status`, `exit-codes`. |

## Install

```
go install github.com/samlzz/situ/cmd/situ@latest
```

`situ` is built with `CGO_ENABLED=0` into a static binary: copy it to the host, nothing else is needed there.

## Documentation

- [Situfile format](docs/format.md)
- [Specification](docs/spec.md): state, drift rules, commands, exit codes
- [Development milestones](docs/dev/milestones.md)

## License

[GPL-3.0-or-later](LICENSE). situ is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
