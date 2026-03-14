# jots

Plural for *jot*.

Jot fast, tidy up later.

## Install

```
go install github.com/f13o/jots/cmd/jots@latest
```

Alias for quick access:

```
alias jot='jots new'
```

## Usage

```
jots new [flags] [title] [output-dir]
```

- `title` -- becomes the filename and `{{title}}` in the template (default: template
  name)
- `output-dir` -- where to write the file (default: current directory)
- `-t name` -- template to use (default: `default`)

## Examples (with the alias)

```
jot                        # ./default.md
jot meeting                # ./meeting.md
jot meeting ~/notes        # ~/notes/meeting.md
jot -t standup retro       # ./retro.md using standup template
```

If the file already exists, a counter is appended: `meeting-2.md`, `meeting-3.md`.

## Templates

Templates live in `~/.jot/templates/`. A default template is created automatically on
first run.

Available variables:

| Variable       | Example                        |
| -------------- | ------------------------------ |
| `{{date}}`     | `2026-03-14`                   |
| `{{datetime}}` | `2026-03-14 10:23:45`          |
| `{{project}}`  | `ws/p/jot` (cwd relative to ~) |
| `{{title}}`    | `meeting`                      |

## Editor

Uses `$EDITOR`, falls back to `$VISUAL`.
