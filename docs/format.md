# Situfile format

Format version: **1**. This document defines the syntax of a `Situfile`. What situ does with each entry is defined in [spec.md](spec.md).

A Situfile describes where things must sit on a host: the files a module installs, the directories it creates, the systemd units it enables or starts, and the components it needs first. A module is a directory holding exactly one file named `Situfile`: a component (`components/<category>/<component>/`) or a host profile (`hosts/<host>/`).

## Example

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

A host profile is a Situfile like any other:

```
format    1
requires  backup/etc-bundle

# role    mode  owner      source           destination
file      0644  root:root  recipients.txt   /etc/etc-bundle/recipients.txt
```

## Lexical rules

- The file is UTF-8 text with LF line endings. A carriage return anywhere is an error.
- One statement per line. Blank lines and lines holding only whitespace are ignored.
- Fields are separated by one or more spaces or tabs. Leading and trailing whitespace is ignored. Columns do not need to be aligned.
- A field that starts with `#` starts a comment: it and the rest of the line are ignored. A `#` inside a field is a literal character (`a#b` is one field).
- There is no quoting and no escaping. A field never contains whitespace.
- Keywords are lowercase and case-sensitive.

## Statements

| Keyword | Fields | Occurrences |
| --- | --- | --- |
| `format` | `format <version>` | exactly once, first statement |
| `requires` | `requires <target>` | zero or more, one target per line |
| `file` | `file <mode> <owner> <source> <destination>` | zero or more |
| `config` | `config <mode> <owner> <source> <destination>` | zero or more |
| `dir` | `dir <mode> <owner> - <destination>` | zero or more |
| `data` | `data <mode> <owner> - <destination>` | zero or more |
| `enable` | `enable <unit>` | zero or more |
| `start` | `start <unit>` | zero or more |

- `format` must be the first statement (comments and blank lines may precede it). Its version must be `1`. Any other value is an error: this version of situ does not read it.
- After `format`, statements may appear in any order. situ never executes them in file order (see the phases in [spec.md](spec.md)). The recommended order is the one in the example: `format`, `requires`, then entries.
- A statement with too few or too many fields is an error. Extra fields are reserved for possible future versions of the format.

## Roles

| Role | Meaning |
| --- | --- |
| `file` | A file owned by the repository. Changes made on the host are overwritten by `install`. |
| `config` | A file the host may legitimately edit. situ protects it: a change made on the host is never overwritten silently. |
| `dir` | A directory created and owned by the component. Removed only if empty. |
| `data` | A directory whose content belongs to programs, not to situ. Created if missing, never deleted. |
| `enable` | A systemd unit that is enabled (`systemctl enable`). |
| `start` | A systemd unit that is started (`systemctl start`). Independent of `enable`. |

Choosing between `file` and `config`: if you may want to edit the file on the host, it is a `config`; otherwise it is a `file`.

## Fields

### `<version>`

A decimal integer. Only `1` is defined.

### `<mode>`

Exactly four octal digits: `0644`, `0755`, `2775`. The first digit carries the setuid, setgid and sticky bits.

### `<owner>`

`<user>:<group>`. Each part is either a name (`[a-z_][a-z0-9_-]*`, optionally ending with `$`) or a decimal ID. Both parts are required. Names are resolved on the host at install time; a name that does not exist on the host is an error.

### `<source>`

A path relative to the directory of the Situfile: `etc-bundle`, `files/app.conf`.

- It must not start with `/`, must not contain `.` or `..` components, empty components (`//`) or a trailing `/`.
- It must name a regular file. A missing file, a directory or a symbolic link is an error.
- For `dir` and `data`, the source field must be `-`.

### `<destination>`

An absolute path on the host: `/etc/default/etc-bundle`.

- It must start with `/`, must not be `/` itself, and must not contain `.` or `..` components, empty components or a trailing `/`.
- It may contain variables (see below). The rules above are checked again after expansion.

### `<unit>`

A systemd unit name: one or more of `A-Z a-z 0-9 : _ . @ \ -`, ending with one of `.service`, `.socket`, `.timer`, `.target`, `.path`, `.mount`, `.automount`, `.swap`, `.slice`. Template instances (`foo@bar.service`) are allowed. A unit name never contains `/`.

### `<target>`

The component a `requires` line depends on: `<component>` or `<category>/<component>`. Names match `[a-z0-9][a-z0-9-]*`. The short form is accepted only if exactly one category holds a component of that name. A host profile can never be a target.

## Variables

- `${NAME}` is replaced in `<destination>` fields only. `NAME` matches `[A-Z][A-Z0-9_]*`.
- Defined in format 1: `HOST`, the name of the host profile directory (`hosts/<host>/`).
- An undefined variable is an error.
- A `$` that does not start a valid `${NAME}` is an error. There is no other expansion and no shell evaluation.

## Rules within one Situfile

These are errors:

- two entries with the same destination (after expansion);
- the same unit listed twice under the same role;
- a target listed twice in `requires`, or a Situfile requiring itself.

Rules that span several modules (a destination owned by two modules, dependency cycles…) are defined in [spec.md](spec.md).

## Errors

A Situfile that breaks any rule of this document is rejected before anything is done on the host. situ reports every error it finds as `<path>:<line>: <message>` on stderr and exits with code 2.

## Versioning

The `format` line makes the syntax versioned. A change that an older situ would misread requires a new format version. A possible evolution is an options column after the last field (`key=value,key=value`, in the style of fstab); it is not part of format 1.
