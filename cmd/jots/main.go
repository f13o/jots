package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: jots <command> [args]\n\ncommands:\n  new      create a note from a template\n  add      index an existing note\n  mv       move/rename a note\n  ls       list indexed notes\n  prune    remove stale entries from the index\n  reindex  re-index notes at a given path")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "add":
		cmdAdd(os.Args[2:])
	case "mv":
		cmdMv(os.Args[2:])
	case "ls":
		cmdLs(os.Args[2:])
	case "prune":
		cmdPrune()
	case "reindex":
		cmdReindex()
	default:
		fatal("unknown command: %s", os.Args[1])
	}
}

func cmdNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	tmplName := fs.String("t", "default", "")
	fs.Parse(args)

	posArgs := fs.Args()
	title := *tmplName
	if len(posArgs) >= 1 {
		title = posArgs[0]
	}
	outputDir := "."
	if len(posArgs) >= 2 {
		outputDir = posArgs[1]
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
	project := projectFromDir(cwd, home)

	vars := map[string]string{
		"date":     now.Format("2006-01-02"),
		"datetime": now.Format("2006-01-02 15:04:05"),
		"project":  project,
		"title":    title,
	}
	result := replaceVars(string(content), vars)

	slug := slugify(title)
	outPath := resolveFilename(outputDir, slug)
	absPath, _ := filepath.Abs(outPath)

	if err := os.WriteFile(outPath, []byte(result), 0644); err != nil {
		fatal("cannot write file: %v", err)
	}

	index := loadIndex(home)
	index = append(index, indexEntry{
		Path:    absPath,
		Project: project,
		Title:   title,
		Created: now.Format("2006-01-02 15:04:05"),
	})
	saveIndex(home, index)

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

func cmdAdd(args []string) {
	if len(args) < 1 {
		fatal("usage: jots add <path>")
	}

	home, _ := os.UserHomeDir()
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		fatal("invalid path: %v", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		fatal("file not found: %s", absPath)
	}

	index := loadIndex(home)
	for _, e := range index {
		if e.Path == absPath {
			fatal("already indexed: %s", absPath)
		}
	}

	dir := filepath.Dir(absPath)
	project := projectFromDir(dir, home)
	title := strings.TrimSuffix(filepath.Base(absPath), ".md")

	info, _ := os.Stat(absPath)
	index = append(index, indexEntry{
		Path:    absPath,
		Project: project,
		Title:   title,
		Created: info.ModTime().Format("2006-01-02 15:04:05"),
	})
	saveIndex(home, index)
	fmt.Println(displayPath(absPath, home))
}

func cmdMv(args []string) {
	if len(args) < 2 {
		fatal("usage: jots mv <src> <dst>")
	}

	src, err := filepath.Abs(args[0])
	if err != nil {
		fatal("invalid source path: %v", err)
	}
	dst, err := filepath.Abs(args[1])
	if err != nil {
		fatal("invalid destination path: %v", err)
	}

	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}

	if err := os.Rename(src, dst); err != nil {
		fatal("move failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	index := loadIndex(home)
	for i, e := range index {
		if e.Path == src {
			index[i].Path = dst
			index[i].Project = projectFromDir(filepath.Dir(dst), home)
			index[i].Title = strings.TrimSuffix(filepath.Base(dst), ".md")
			break
		}
	}
	saveIndex(home, index)
	fmt.Println(displayPath(dst, home))
}

func cmdLs(args []string) {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	pattern := fs.String("re", "", "")
	fs.Parse(args)

	home, _ := os.UserHomeDir()
	index := loadIndex(home)

	var re *regexp.Regexp
	if *pattern != "" {
		var err error
		re, err = regexp.Compile(*pattern)
		if err != nil {
			fatal("invalid regexp: %v", err)
		}
	}

	for _, e := range index {
		if re != nil && !re.MatchString(e.Path) && !re.MatchString(e.Title) && !re.MatchString(e.Project) {
			continue
		}
		fmt.Println(displayPath(e.Path, home))
	}
}

func cmdPrune() {
	home, _ := os.UserHomeDir()
	index := loadIndex(home)

	var stale []indexEntry
	for _, e := range index {
		if _, err := os.Stat(e.Path); err != nil {
			stale = append(stale, e)
		}
	}

	if len(stale) == 0 {
		fmt.Println("nothing to prune")
		return
	}

	fmt.Printf("stale entries (%d):\n", len(stale))
	for _, e := range stale {
		fmt.Printf("  %s\n", displayPath(e.Path, home))
	}
	fmt.Print("\nremove these entries? [y/N] ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
		fmt.Println("aborted")
		return
	}

	var clean []indexEntry
	for _, e := range index {
		if _, err := os.Stat(e.Path); err == nil {
			clean = append(clean, e)
		}
	}
	saveIndex(home, clean)
	fmt.Printf("removed %d entries, %d remaining\n", len(stale), len(clean))
}

func cmdReindex() {
	args := os.Args[2:]
	if len(args) < 1 {
		fatal("usage: jots reindex <path>")
	}

	dir, err := filepath.Abs(args[0])
	if err != nil {
		fatal("invalid path: %v", err)
	}

	home, _ := os.UserHomeDir()
	index := loadIndex(home)

	indexed := make(map[string]bool)
	for _, e := range index {
		indexed[e.Path] = true
	}

	added := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if indexed[path] {
			return nil
		}
		index = append(index, indexEntry{
			Path:    path,
			Project: projectFromDir(filepath.Dir(path), home),
			Title:   strings.TrimSuffix(filepath.Base(path), ".md"),
			Created: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		indexed[path] = true
		added++
		return nil
	})

	saveIndex(home, index)
	fmt.Printf("added %d entries, %d total\n", added, len(index))
}

type indexEntry struct {
	Path    string `json:"path"`
	Project string `json:"project"`
	Title   string `json:"title"`
	Created string `json:"created"`
}

func loadIndex(home string) []indexEntry {
	path := filepath.Join(home, ".jot", "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var index []indexEntry
	json.Unmarshal(data, &index)
	return index
}

func saveIndex(home string, index []indexEntry) {
	dir := filepath.Join(home, ".jot")
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(index, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "index.json"), data, 0644); err != nil {
		fatal("cannot write index: %v", err)
	}
}

func projectFromDir(dir, home string) string {
	rel := strings.TrimPrefix(dir, home+"/")
	if dir == home {
		return ""
	}
	return rel
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

func resolveFilename(dir, slug string) string {
	base := filepath.Join(dir, slug+".md")
	if _, err := os.Stat(base); err != nil {
		return base
	}
	for i := 2; ; i++ {
		name := filepath.Join(dir, fmt.Sprintf("%s-%d.md", slug, i))
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

func displayPath(path, home string) string {
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
