package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	tmplName := flag.String("t", "default", "")
	flag.Parse()

	args := flag.Args()
	title := *tmplName
	if len(args) >= 1 {
		title = args[0]
	}
	outputDir := "."
	if len(args) >= 2 {
		outputDir = args[1]
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fatal("cannot determine home directory: %v", err)
	}

	content, err := loadTemplate(home, *tmplName)
	if err != nil {
		fatal("%v", err)
	}

	now := time.Now()
	cwd, _ := os.Getwd()
	project := strings.TrimPrefix(cwd, home+"/")
	if cwd == home {
		project = ""
	}

	vars := map[string]string{
		"date":     now.Format("2006-01-02"),
		"datetime": now.Format("2006-01-02 15:04:05"),
		"project":  project,
		"title":    title,
	}
	result := replaceVars(string(content), vars)

	slug := slugify(title)
	outPath := resolveFilename(outputDir, now.Format("2006-01-02"), slug)

	if err := os.WriteFile(outPath, []byte(result), 0644); err != nil {
		fatal("cannot write file: %v", err)
	}

	editor := editorCmd()
	if editor == "" {
		fatal("no editor found: set $EDITOR or $VISUAL")
	}

	editorPath, err := exec.LookPath(editor)
	if err != nil {
		fatal("editor %q not found in PATH: %v", editor, err)
	}

	syscall.Exec(editorPath, []string{editor, outPath}, os.Environ())
}

var defaultTemplate = `---
created: {{datetime}}
project: {{project}}
title: {{title}}
---

`

func loadTemplate(home, name string) ([]byte, error) {
	name = strings.TrimSuffix(name, ".md")
	dir := filepath.Join(home, ".jot", "templates")

	path := filepath.Join(dir, name+".md")
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}

	if name != "default" {
		fallback := filepath.Join(dir, "default.md")
		if data, err := os.ReadFile(fallback); err == nil {
			return data, nil
		}
		return nil, fmt.Errorf("template %q not found", name)
	}

	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "default.md"), []byte(defaultTemplate), 0644)
	return []byte(defaultTemplate), nil
}

func replaceVars(content string, vars map[string]string) string {
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

func slugify(title string) string {
	var b strings.Builder
	prev := false
	for _, c := range strings.ToLower(title) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			prev = false
		} else if !prev && b.Len() > 0 {
			b.WriteByte('-')
			prev = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func resolveFilename(dir, date, slug string) string {
	base := filepath.Join(dir, date+"-"+slug+".md")
	if _, err := os.Stat(base); err != nil {
		return base
	}
	for i := 2; ; i++ {
		name := filepath.Join(dir, fmt.Sprintf("%s-%s-%d.md", date, slug, i))
		if _, err := os.Stat(name); err != nil {
			return name
		}
	}
}

func editorCmd() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return os.Getenv("VISUAL")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
