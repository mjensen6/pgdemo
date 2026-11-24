package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// tmux runs inside the container on its own socket, so it never meets the
// learner's tmux, even when pgdemo is started from inside one.

const tmuxSocket = "pgdemo"
const tmuxSession = "pgdemo"
const quitFlag = "/tmp/pgdemo-quit"
const exitQuit = 10 // tells the launcher to stop the database

const tmuxConf = `
set -g mouse on
set -g prefix None
set -g escape-time 0
set -g default-terminal "tmux-256color"
set -g history-limit 10000
set -g focus-events on
set -g pane-border-status top
set -g pane-border-format " #{pane_title} "
set -g pane-active-border-style "fg=cyan"
set -g pane-border-lines heavy
set -g status on
set -g status-position bottom
set -g status-style "bg=default"
set -g status-left ""
set -g status-right " #[dim]click a tab to switch "
set -g window-status-format " #W "
set -g window-status-current-format "#[reverse] #W "
set -g window-status-separator "|"
set -g allow-rename off
set -g set-clipboard on
`

func tmux(args ...string) error {
	conf := filepath.Join(os.TempDir(), "pgdemo-tmux.conf")
	os.WriteFile(conf, []byte(tmuxConf), 0o644)
	out, err := exec.Command("tmux", append([]string{"-L", tmuxSocket, "-f", conf}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	return nil
}

func self() string {
	p, err := os.Executable()
	if err != nil {
		return "pgdemo-player"
	}
	return p
}

// shellQuote makes one argument safe inside the command string tmux runs.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func paneCmd(args ...string) string {
	q := []string{shellQuote(self())}
	for _, a := range args {
		q = append(q, shellQuote(a))
	}
	return strings.Join(q, " ")
}

// fixTerm falls back to a common terminal type if the learner's terminal
// is one the container has never heard of.
func fixTerm() {
	t := os.Getenv("TERM")
	if t == "" || exec.Command("infocmp", t).Run() != nil {
		os.Setenv("TERM", "xterm-256color")
	}
}

// openLesson builds the three-pane screen for e and attaches to it.
func openLesson(c *Course, e *Entry) error {
	if err := ensurePlayground(e.Lesson); err != nil {
		return err
	}
	fixTerm()
	writePgcliConfig()
	os.Remove(quitFlag)

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 160, 45
	}
	if w < 120 || h < 30 {
		fmt.Printf("Tip: pgdemo works best in a terminal at least 120x30 (this one is %dx%d). Maximize the window if you can.\n", w, h)
		time.Sleep(2 * time.Second)
	}

	tmux("kill-session", "-t", tmuxSession)
	db := playground(e.Name)
	steps := [][]string{
		{"new-session", "-d", "-s", tmuxSession, "-n", "Lesson", "-x", fmt.Sprint(w), "-y", fmt.Sprint(h - 1), paneCmd("pane", e.Name)},
		{"split-window", "-h", "-t", tmuxSession + ":Lesson.0", "-l", "48%", paneCmd("sql-pane", db)},
		{"split-window", "-v", "-t", tmuxSession + ":Lesson.1", "-l", "40%", paneCmd("live-pane", e.Name)},
		{"select-pane", "-t", tmuxSession + ":Lesson.0", "-T", e.Label()},
		{"select-pane", "-t", tmuxSession + ":Lesson.1", "-T", "SQL"},
		{"select-pane", "-t", tmuxSession + ":Lesson.2", "-T", "Live View"},
		{"new-window", "-d", "-t", tmuxSession + ":", "-n", "Browse", paneCmd("placeholder", "Browse opens lazysql to explore the database. It arrives in the next build.")},
		{"new-window", "-d", "-t", tmuxSession + ":", "-n", "Top", paneCmd("top-pane")},
		{"select-pane", "-t", tmuxSession + ":Browse.0", "-T", "Browse"},
		{"select-pane", "-t", tmuxSession + ":Top.0", "-T", "Top: live server activity (pg_activity, q to restart)"},
		{"select-window", "-t", tmuxSession + ":Lesson"},
		{"select-pane", "-t", tmuxSession + ":Lesson.0"},
	}
	for _, s := range steps {
		if err := tmux(s...); err != nil {
			return err
		}
	}

	conf := filepath.Join(os.TempDir(), "pgdemo-tmux.conf")
	attach := exec.Command("tmux", "-L", tmuxSocket, "-f", conf, "attach-session", "-t", tmuxSession)
	attach.Env = withoutTMUX()
	attach.Stdin, attach.Stdout, attach.Stderr = os.Stdin, os.Stdout, os.Stderr
	attach.Run()

	if _, err := os.Stat(quitFlag); err == nil {
		os.Remove(quitFlag)
		os.Exit(exitQuit)
	}
	return nil
}

// withoutTMUX lets attach work even if a TMUX variable leaked in.
func withoutTMUX() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "TMUX=") {
			env = append(env, kv)
		}
	}
	return env
}

// switchLesson reconnects the SQL and Live View panes to a new lesson.
func switchLesson(e *Entry) error {
	db := playground(e.Name)
	for _, s := range [][]string{
		{"respawn-pane", "-k", "-t", tmuxSession + ":Lesson.1", paneCmd("sql-pane", db)},
		{"respawn-pane", "-k", "-t", tmuxSession + ":Lesson.2", paneCmd("live-pane", e.Name)},
		{"select-pane", "-t", tmuxSession + ":Lesson.0", "-T", e.Label()},
		{"select-pane", "-t", tmuxSession + ":Lesson.1", "-T", "SQL"},
		{"select-pane", "-t", tmuxSession + ":Lesson.2", "-T", "Live View"},
		{"select-pane", "-t", tmuxSession + ":Lesson.0"},
	} {
		if err := tmux(s...); err != nil {
			return err
		}
	}
	return nil
}

func quitSession() {
	os.WriteFile(quitFlag, nil, 0o644)
	tmux("kill-session", "-t", tmuxSession)
}

func writePgcliConfig() {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "pgcli")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config"), []byte(`[main]
use_local_timezone = False
keyring = False
less_chatty = True
multi_line = False
prompt = '\d=# '
`), 0o644)
}

// sqlPane keeps a prompt connected to the playground. If the learner
// exits it, it waits for Enter and reconnects.
func sqlPane(db string) {
	for {
		start := time.Now()
		cmd := exec.Command("pgcli", db)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := cmd.Run()
		if err != nil && time.Since(start) < 3*time.Second {
			// pgcli could not start: fall back to plain psql.
			cmd = exec.Command("psql", "-X", "-d", db)
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			cmd.Run()
		}
		fmt.Print("\nThe SQL prompt closed. Press Enter to reopen it.")
		fmt.Scanln()
	}
}

func livePane(c *Course, name string) {
	e := c.ByName(name)
	if e == nil || e.Lesson == nil || e.Lesson.View == "" {
		placeholder("This lesson has no Live View.")
		return
	}
	for {
		cmd := exec.Command("pspg", "-d", playground(name), "--query", e.Lesson.View, "--watch", "1",
			"--no-commandbar", "--no-topbar", "--no-scrollbar")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.Run()
		fmt.Print("\nThe Live View closed. Press Enter to reopen it.")
		fmt.Scanln()
	}
}

func topPane() {
	for {
		cmd := exec.Command("pg_activity", "-d", "postgres", "--no-db-size")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.Run()
		fmt.Print("\npg_activity closed. Press Enter to reopen it.")
		fmt.Scanln()
	}
}

func placeholder(msg string) {
	fmt.Print("\n  " + msg + "\n")
	for {
		time.Sleep(time.Hour)
	}
}
