package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Everything talks to the course's own server through psql over the local
// socket. It never looks for, or touches, any other Postgres.

const progressDB = "pgdemo"
const readyMark = "pgdemo:ready"

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// query runs SQL and returns unaligned, tuples-only output.
func query(db, sql string) (string, error) {
	cmd := exec.Command("psql", "-X", "-q", "-At", "-v", "ON_ERROR_STOP=1", "-d", db, "-c", sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// script runs several statements, one at a time, stopping at the first error.
func script(db, sql string) (string, error) {
	cmd := exec.Command("psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-d", db, "-f", "-")
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// waitForServer gives a just-started container a moment to accept connections.
func waitForServer() error {
	var err error
	for i := 0; i < 60; i++ {
		if _, err = query("postgres", "SELECT 1"); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("the course database did not start: %v", err)
}

func ensureProgress() error {
	if err := waitForServer(); err != nil {
		return err
	}
	exists, err := query("postgres", "SELECT 1 FROM pg_database WHERE datname = "+quote(progressDB))
	if err != nil {
		return err
	}
	if exists == "" {
		if _, err := query("postgres", "CREATE DATABASE "+progressDB); err != nil {
			return err
		}
	}
	_, err = query(progressDB, `CREATE TABLE IF NOT EXISTS progress (
  lesson       text PRIMARY KEY,
  state        text NOT NULL CHECK (state IN ('watched', 'done')),
  current_step int  NOT NULL DEFAULT 1,
  updated_at   timestamptz NOT NULL DEFAULT now()
)`)
	return err
}

type Progress struct {
	State string // "", "watched" or "done"
	Step  int    // the next step to run, starting at 1
}

func getProgress(name string) Progress {
	out, err := query(progressDB, "SELECT state, current_step FROM progress WHERE lesson = "+quote(name))
	if err != nil || out == "" {
		return Progress{Step: 1}
	}
	parts := strings.Split(out, "|")
	n, _ := strconv.Atoi(parts[1])
	return Progress{State: parts[0], Step: n}
}

func allProgress() map[string]Progress {
	m := map[string]Progress{}
	out, err := query(progressDB, "SELECT lesson, state, current_step FROM progress")
	if err != nil {
		return m
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "|")
		if len(parts) == 3 {
			n, _ := strconv.Atoi(parts[2])
			m[parts[0]] = Progress{State: parts[1], Step: n}
		}
	}
	return m
}

// lastLesson is the lesson most recently worked on, for plain `pgdemo`.
func lastLesson() string {
	out, _ := query(progressDB, "SELECT lesson FROM progress ORDER BY updated_at DESC LIMIT 1")
	return out
}

// markWatched records that steps ran. It never downgrades a done lesson.
func markWatched(name string, nextStep int) error {
	_, err := query(progressDB, fmt.Sprintf(`INSERT INTO progress (lesson, state, current_step) VALUES (%s, 'watched', %d)
ON CONFLICT (lesson) DO UPDATE SET current_step = EXCLUDED.current_step, updated_at = now()`, quote(name), nextStep))
	return err
}

// markDone is the only way a lesson becomes done: a check passed.
func markDone(name string) error {
	_, err := query(progressDB, fmt.Sprintf(`INSERT INTO progress (lesson, state) VALUES (%s, 'done')
ON CONFLICT (lesson) DO UPDATE SET state = 'done', updated_at = now()`, quote(name)))
	return err
}

func setStep(name string, step int) error {
	_, err := query(progressDB, fmt.Sprintf("UPDATE progress SET current_step = %d, updated_at = now() WHERE lesson = %s", step, quote(name)))
	return err
}

func playgroundReady(name string) bool {
	out, _ := query("postgres", "SELECT shobj_description(oid, 'pg_database') FROM pg_database WHERE datname = "+quote(playground(name)))
	return out == readyMark
}

// ensurePlayground builds the lesson's database the first time it opens,
// and keeps it, with the learner's work, until reset.
func ensurePlayground(l *Lesson) error {
	if playgroundReady(l.Name) {
		return nil
	}
	return buildPlayground(l)
}

func buildPlayground(l *Lesson) error {
	db := playground(l.Name)
	if _, err := query("postgres", "DROP DATABASE IF EXISTS "+db+" WITH (FORCE)"); err != nil {
		return err
	}
	if _, err := query("postgres", "CREATE DATABASE "+db); err != nil {
		return err
	}
	if l.Setup != "" {
		if _, err := script(db, l.Setup); err != nil {
			return fmt.Errorf("setting up lesson %s failed: %v", l.Name, err)
		}
	}
	if _, err := query("postgres", "COMMENT ON DATABASE "+db+" IS "+quote(readyMark)); err != nil {
		return err
	}
	return setStep(l.Name, 1)
}

// resetPlayground rewinds to step 1. It keeps a lesson's done status.
func resetPlayground(l *Lesson) error {
	runCleanup(l)
	return buildPlayground(l)
}

// runCleanup undoes effects that reach past the playground. Errors are
// ignored: there may be nothing left to clean.
func runCleanup(l *Lesson) {
	if l.Cleanup != "" && playgroundReady(l.Name) {
		script(playground(l.Name), l.Cleanup)
	}
}

// check runs the lesson's hidden check against the playground's committed
// state, so any correct approach passes.
func check(l *Lesson) (bool, error) {
	out, err := query(playground(l.Name), l.Check)
	if err != nil {
		return false, err
	}
	return out == "t", nil
}

// Session is one long-running psql process per lesson, so every sql step
// shares a session: BEGIN in one step and ROLLBACK in a later one works.
type Session struct {
	db    string
	cmd   *exec.Cmd
	in    *os.File
	lines chan string
	n     int
}

func startSession(db string) (*Session, error) {
	outR, outW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("psql", "-X", "-P", "pager=off", "-d", db)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = inR, outW, outW // one pipe keeps errors in order
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	inR.Close()
	outW.Close()
	s := &Session{db: db, cmd: cmd, in: inW, lines: make(chan string, 1024)}
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			s.lines <- sc.Text()
		}
		close(s.lines)
	}()
	return s, nil
}

func (s *Session) Run(sql string) (string, error) {
	s.n++
	mark := fmt.Sprintf("__pgdemo_step_%d__", s.n)
	if _, err := fmt.Fprintf(s.in, "%s\n\\echo %s\n", sql, mark); err != nil {
		return "", fmt.Errorf("the SQL session ended: %v", err)
	}
	var out []string
	for line := range s.lines {
		if line == mark {
			return strings.TrimRight(strings.Join(out, "\n"), "\n"), nil
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), fmt.Errorf("the SQL session ended")
}

func (s *Session) Close() {
	s.in.Close()
	s.cmd.Wait()
}

// runStep runs one step: sql in the shared session, sh as a script with
// psql pointed at the playground.
func runStep(s *Session, step Step) (string, error) {
	switch step.Lang {
	case "sql":
		return s.Run(step.Code)
	case "sh":
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bash", "-c", step.Code)
		cmd.Dir = os.TempDir()
		cmd.Env = append(os.Environ(), "PGDATABASE="+s.db)
		var b bytes.Buffer
		cmd.Stdout, cmd.Stderr = &b, &b
		// A background job that keeps the output open must not hang the step.
		cmd.WaitDelay = time.Second
		err := cmd.Run()
		if errors.Is(err, exec.ErrWaitDelay) {
			err = nil
		}
		return strings.TrimRight(b.String(), "\n"), err
	}
	return "", nil
}
