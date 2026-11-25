// pgdemo-player runs inside the course container. The host launcher,
// ./pgdemo, handles Docker and forwards every other command here.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pgdemo: "+err.Error())
		os.Exit(1)
	}
}

func lessonsDir() string {
	if d := os.Getenv("PGDEMO_LESSONS"); d != "" {
		return d
	}
	return "/opt/pgdemo/lessons"
}

func run(args []string) error {
	home := filepath.Join(os.TempDir(), "pgdemo-home")
	os.MkdirAll(home, 0o755)
	os.Setenv("HOME", home)

	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	// help and readme need no database, so they work before setup.
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		return help(args[min(1, len(args)):])
	}
	if cmd == "readme" {
		if len(args) < 2 {
			return fmt.Errorf("usage: pgdemo-player readme <repo-root> [--check]")
		}
		return readme(args[1], len(args) > 2 && args[2] == "--check")
	}

	c, err := loadCourse(lessonsDir())
	if err != nil {
		return err
	}

	switch cmd {
	case "sql-pane":
		sqlPane(args[1])
		return nil
	case "live-pane":
		livePane(c, args[1])
		return nil
	case "top-pane":
		topPane()
		return nil
	case "browse-pane":
		browsePane(args[1])
		return nil
	case "placeholder":
		placeholder(args[1])
		return nil
	}

	if err := ensureProgress(); err != nil {
		return err
	}

	switch cmd {
	case "":
		return openLesson(c, resumeEntry(c))
	case "pane":
		return lessonPane(c, args[1])
	case "list":
		list(c)
		return nil
	case "check":
		return checkAll(c)
	case "psql":
		return syscall.Exec("/usr/bin/psql", []string{"psql", "-X", "-d", "postgres"}, os.Environ())
	case "test":
		return courseTest(c, len(args) > 1 && args[1] == "-v")
	case "section":
		return sectionCmd(c, args[1:])
	case "browse":
		return browse(c, args[1:])
	case "top":
		return top()
	case "doctor":
		return doctor(c)
	case "connect-info": // used by the launcher's pgdemo connect
		return connectInfo(c, args[1:])
	case "connect", "uninstall", "setup", "stop":
		return fmt.Errorf("pgdemo %s runs on your computer, not in the container. Run it from your terminal", cmd)
	}

	e, err := c.Find(cmd)
	if err != nil {
		return err
	}
	if e.Lesson == nil {
		return fmt.Errorf("lesson %d (%s) is coming soon. See what is ready with: pgdemo list", e.Num, e.Name)
	}
	sub := "open"
	if len(args) > 1 {
		sub = args[1]
	}
	l := e.Lesson
	switch sub {
	case "open":
		return openLesson(c, e)
	case "tldr":
		fmt.Printf("%s\n%s\nTry it: pgdemo %d\n", e.Label(), l.TLDR, e.Num)
	case "info":
		fmt.Println(renderMarkdown(l.Markdown(e)))
	case "hint":
		fmt.Println("Hint: " + l.Hint)
	case "solution":
		fmt.Println(renderMarkdown("```sql\n" + l.Solution + "\n```"))
	case "reset":
		if err := resetPlayground(l); err != nil {
			return err
		}
		fmt.Printf("Lesson %d's playground is rebuilt and back at step 1. Your progress is kept.\n", e.Num)
	case "check":
		return checkOne(e)
	case "run":
		return runAll(e)
	case "view":
		return view(e)
	default:
		return fmt.Errorf("unknown lesson command %q. Try: pgdemo help", sub)
	}
	return nil
}

// resumeEntry picks up where the learner left off: the last lesson they
// worked on, or the one after it if they finished it.
func resumeEntry(c *Course) *Entry {
	if e := c.ByName(lastLesson()); e != nil && e.Lesson != nil {
		if getProgress(e.Name).State == "done" {
			if n := c.Next(e); n != nil {
				return n
			}
		}
		return e
	}
	return c.First()
}

func lessonPane(c *Course, name string) error {
	e := c.ByName(name)
	if e == nil || e.Lesson == nil {
		return fmt.Errorf("no lesson %q", name)
	}
	m, err := newModel(c, e)
	if err != nil {
		placeholder(err.Error())
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func list(c *Course) {
	prog := allProgress()
	last := 0
	for _, e := range c.Entries {
		if e.SectionNum != last {
			last = e.SectionNum
			note := ""
			if e.Section.ServerWide {
				note = "  (these lessons share the whole server)"
			}
			fmt.Printf("\n%d %s%s\n", e.SectionNum, e.Section.Title, note)
		}
		if e.Lesson == nil {
			fmt.Printf("  ·  %2d  %-24s coming soon\n", e.Num, e.Name)
			continue
		}
		sym := "○"
		switch prog[e.Name].State {
		case "watched":
			sym = "◐"
		case "done":
			sym = "●"
		}
		fmt.Printf("  %s  %2d  %-24s %s\n", sym, e.Num, e.Name, e.Lesson.Title)
	}
	fmt.Println("\n○ not started   ◐ watched   ● done        Open one with: pgdemo 1")
}

func checkOne(e *Entry) error {
	if !playgroundReady(e.Name) {
		return fmt.Errorf("lesson %d has not been opened yet. Start it with: pgdemo %d", e.Num, e.Num)
	}
	ok, err := check(e.Lesson)
	switch {
	case err != nil:
		fmt.Printf("Not yet. The check hit an error: %s\nHint: pgdemo %d hint\n", firstLine(err.Error()), e.Num)
	case ok:
		markDone(e.Name)
		fmt.Printf("● %s is done.\n", e.Label())
	default:
		fmt.Printf("Not yet. For a hint: pgdemo %d hint\n", e.Num)
	}
	return nil
}

func checkAll(c *Course) error {
	prog := allProgress()
	any := false
	for _, e := range c.Entries {
		if _, opened := prog[e.Name]; !opened || e.Lesson == nil {
			continue
		}
		any = true
		if err := checkOne(e); err != nil {
			return err
		}
	}
	if !any {
		fmt.Println("You have not opened any lessons yet. Start with: pgdemo")
	}
	return nil
}

// runAll plays every step non-stop. It counts as watched, never done.
func runAll(e *Entry) error {
	l := e.Lesson
	if err := ensurePlayground(l); err != nil {
		return err
	}
	s, err := startSession(playground(e.Name))
	if err != nil {
		return err
	}
	defer s.Close()
	fmt.Println(renderMarkdown("# " + e.Label()))
	for i, st := range l.Steps {
		md := fmt.Sprintf("**Step %d of %d.** %s", i+1, len(l.Steps), st.Text)
		if st.Lang != "" {
			md += fmt.Sprintf("\n\n```%s\n%s\n```", fence(st.Lang), st.Code)
		}
		fmt.Println(renderMarkdown(md))
		out, _ := runStep(s, st)
		if out != "" {
			fmt.Println(out)
		}
	}
	markWatched(e.Name, len(l.Steps)+1)
	fmt.Printf("\nNow try the challenge: pgdemo %d\n", e.Num)
	return nil
}

func renderMarkdown(md string) string {
	w, style := 100, "notty" // plain text when piped
	if term.IsTerminal(int(os.Stdout.Fd())) {
		style = "dark"
		if tw, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			w = min(tw, 100)
		}
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(w-4))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

// courseTest plays every built lesson from a clean start: each step, then
// the check must fail, then the solution, then the check must pass.
func courseTest(c *Course, verbose bool) error {
	failed := 0
	for _, e := range c.Entries {
		if e.Lesson == nil {
			continue
		}
		if err := testLesson(e, verbose); err != nil {
			failed++
			fmt.Printf("FAIL %2d %s: %v\n", e.Num, e.Name, err)
		} else {
			fmt.Printf("ok   %2d %s\n", e.Num, e.Name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d lesson(s) failed", failed)
	}
	return nil
}

func testLesson(e *Entry, verbose bool) error {
	l := e.Lesson
	if err := resetPlayground(l); err != nil {
		return err
	}
	defer resetPlayground(l)
	if ok, _ := check(l); ok {
		return fmt.Errorf("the check passes on a fresh playground, before any step")
	}
	s, err := startSession(playground(e.Name))
	if err != nil {
		return err
	}
	defer s.Close()
	for i, st := range l.Steps {
		out, err := runStep(s, st)
		if verbose {
			fmt.Printf("--- %s step %d\n%s\n%s\n", e.Name, i+1, st.Code, out)
		}
		if err != nil {
			return fmt.Errorf("step %d: %v\n%s", i+1, err, out)
		}
	}
	if ok, _ := check(l); ok {
		return fmt.Errorf("the check passes before the challenge is solved")
	}
	out, err := s.Run(l.Solution)
	if verbose {
		fmt.Printf("--- %s solution\n%s\n", e.Name, out)
	}
	if err != nil {
		return fmt.Errorf("solution: %v", err)
	}
	ok, err := check(l)
	if err != nil {
		return fmt.Errorf("check: %v", err)
	}
	if !ok {
		return fmt.Errorf("the check fails after the solution\n%s", out)
	}
	return nil
}

const readmeStart = "<!-- lessons:start (generated from lessons/course.yaml) -->"
const readmeEnd = "<!-- lessons:end -->"

// readme regenerates the numbered lesson list in README.md.
func readme(root string, checkOnly bool) error {
	c, err := loadCourse(filepath.Join(root, "lessons"))
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(readmeStart + "\n")
	last := 0
	for _, e := range c.Entries {
		if e.SectionNum != last {
			last = e.SectionNum
			note := ""
			if e.Section.ServerWide {
				note = " (shares the whole server)"
			}
			fmt.Fprintf(&b, "\n**%d. %s**%s\n\n", e.SectionNum, e.Section.Title, note)
		}
		if e.Lesson == nil {
			fmt.Fprintf(&b, "%d. `%s` _(coming soon)_\n", e.Num, e.Name)
		} else {
			fmt.Fprintf(&b, "%d. [%s](lessons/%s.md): %s\n", e.Num, e.Lesson.Title, e.Name, e.Lesson.TLDR)
		}
	}
	b.WriteString("\n" + readmeEnd)

	path := filepath.Join(root, "README.md")
	cur, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	i, j := bytes.Index(cur, []byte(readmeStart)), bytes.Index(cur, []byte(readmeEnd))
	if i < 0 || j < i {
		return fmt.Errorf("README.md needs the lines %s and %s", readmeStart, readmeEnd)
	}
	next := append(append(append([]byte{}, cur[:i]...), b.String()...), cur[j+len(readmeEnd):]...)
	if bytes.Equal(cur, next) {
		return nil
	}
	if checkOnly {
		return fmt.Errorf("README.md's lesson list is out of date. Run: ./scripts/readme")
	}
	return os.WriteFile(path, next, 0o644)
}
