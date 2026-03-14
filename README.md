# jots

Plural for *jot*.

Jot fast, tidy up later.

## Install

```
go install github.com/f13o/jots/cmd/jots@latest
```

## Take notes

```
jots new [flags] title 
```

- `title` -- becomes the filename and `{{title}}` in the template (default: template
  name)
- `-d path` -- where to write the file (default: current directory)
- `-t name` -- template to use (default: `default`)

### Examples

```
jots new                            # ./no-name-jot.md
jots new meeting                    # ./meeting.md
jots new file with long title       # ./file-with-long-title.md
jots new -d ~/notes meeting         # ~/notes/meeting.md
jots new -t standup retro           # ./retro.md using standup template
```

If the file already exists, a counter is appended: `meeting-2.md`, `meeting-3.md`.

### Jot faster with an alias

```
alias jot="jots new"
```

## Tidy up: Jots Index

Every document you crate with jot is mapped into the Jot Index (`~/.jot/index.json`) so
you can find later and tidy up as you need:

```
jots ls             # list indexed notes

jots add            # index an existing note
jots mv             # move/rename a note
jots prune          # remove stale entries from the index
```

Use `jots mv <src> <dst>` as normal `mv` UNIX command to keep the index aware.

In case you forget to use `jots mv` and moved a file with the UNIX command, you can
always re-add them with `jots add` and then `jots prune` to remove stales from the
index.

## Templates

Templates live in `~/.jot/templates/`. A default template is created automatically on
first run.

Every new jot can leverage on the basic templating that runs at jot creation.

A template that defines the following variables will have them auto-filled wherever they
appear in the document, both frontmatter and body:

| Variable       | Example                        |
| -------------- | ------------------------------ |
| `{{date}}`     | `2026-03-14`                   |
| `{{datetime}}` | `2026-03-14 10:23:45`          |
| `{{project}}`  | `ws/p/jot` (cwd relative to ~) |
| `{{title}}`    | `meeting`                      |

## Editor

Uses `$EDITOR`, falls back to `$VISUAL`.
