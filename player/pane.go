package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// The Lesson pane. Positions: -2 is the Course Overview (first lesson
// only), -1 is the lesson intro, 0..n-1 are steps, and n is the challenge.

const posOverview = -2

var (
	headStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	keyStyle  = lipgloss.NewStyle().Bold(true)
	outStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("8")).PaddingLeft(1)
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	labelRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).PaddingLeft(2)
	labelRan  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).PaddingLeft(2)
)

type model struct {
	course  *Course
	entry   *Entry
	sess    *Session
	pos     int
	ran     int
	outputs map[int]string
	running bool
	msg     string
	hint    bool
	help    bool
	confirm string // "quit" or "reset" while asking y/n
	width   int
	height  int
	vp      viewport.Model
	md      *glamour.TermRenderer
	ready   bool
}

type stepDoneMsg struct {
	idx int
	out string
	err error
}
type checkDoneMsg struct {
	ok  bool
	err error
}
type lessonMsg struct {
	entry *Entry
	err   error
	what  string // what was being done, for the error message
}

func newModel(c *Course, e *Entry) (*model, error) {
	m := &model{course: c}
	if err := m.load(e); err != nil {
		return nil, err
	}
	return m, nil
}

// load starts a lesson, resuming at the saved step.
func (m *model) load(e *Entry) error {
	if m.sess != nil {
		m.sess.Close()
		m.sess = nil
	}
	if err := ensurePlayground(e.Lesson); err != nil {
		return err
	}
	s, err := startSession(playground(e.Name))
	if err != nil {
		return err
	}
	m.entry, m.sess = e, s
	m.outputs = map[int]string{}
	m.msg, m.hint, m.help, m.confirm = "", false, false, ""
	n := len(e.Lesson.Steps)
	m.ran = min(max(getProgress(e.Name).Step-1, 0), n)
	switch {
	case m.ran == 0:
		m.pos = m.start()
	case m.ran == n:
		m.pos = n
	default:
		m.pos = m.ran // resume on the next step, not yet run
	}
	return nil
}

func (m *model) Init() tea.Cmd { return nil }

// start is the first page of the lesson: the Course Overview for the first
// lesson of the course, otherwise the lesson intro.
func (m *model) start() int {
	if m.course.Overview != "" && m.entry == m.course.First() {
		return posOverview
	}
	return -1
}

func (m *model) steps() []Step { return m.entry.Lesson.Steps }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.md, _ = glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(max(m.width-4, 20)))
		if !m.ready {
			m.vp = viewport.New(m.width, m.bodyHeight())
			m.ready = true
		}
		m.vp.Width, m.vp.Height = m.width, m.bodyHeight()
		m.refresh(true)
		return m, nil

	case stepDoneMsg:
		m.running = false
		out := msg.out
		if msg.err != nil && out == "" {
			out = msg.err.Error()
		}
		m.outputs[msg.idx] = out
		m.ran = msg.idx + 1
		m.pos = msg.idx
		m.msg = m.afterRun()
		if err := markWatched(m.entry.Name, m.ran+1); err != nil {
			m.msg = warnStyle.Render("Could not save progress: " + err.Error())
		}
		m.refresh(false)
		return m, nil

	case checkDoneMsg:
		m.running = false
		switch {
		case msg.err != nil:
			m.msg = warnStyle.Render("Not yet.") + " The check hit an error: " + firstLine(msg.err.Error()) + ". Press h for a hint."
		case msg.ok:
			markDone(m.entry.Name)
			m.msg = okStyle.Render("● Done!") + " Press n for the next lesson."
		default:
			m.msg = warnStyle.Render("Not yet.") + " Press h for a hint, or s to skip."
		}
		m.refresh(false)
		return m, nil

	case lessonMsg:
		m.running = false
		if msg.err == nil {
			msg.err = m.load(msg.entry)
		}
		if msg.err != nil {
			m.msg = warnStyle.Render("Could not "+msg.what+": ") + msg.err.Error()
		} else if msg.what == "reset the lesson" {
			m.msg = okStyle.Render("Reset to the start.") + " " + m.nextHint()
		}
		m.refresh(true)
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

// nextHint says what Enter does from here. The status line always shows
// it, because on a long page the same words at the end may be off screen.
func (m *model) nextHint() string {
	n := len(m.steps())
	switch {
	case m.pos == posOverview:
		return "Press Enter to begin."
	case m.pos == -1:
		return "Press Enter to see step 1."
	case m.pos < n && m.running:
		return "Running..."
	case m.pos < n && m.steps()[m.pos].Lang != "" && m.pos == m.ran:
		return "Press Enter to run it."
	case m.pos < n && m.pos < m.ran:
		return m.afterRun()
	case m.pos < n:
		return "Press Enter to continue."
	}
	return "Type your answer in the SQL pane, then press Enter here to check it."
}

func (m *model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm != "" {
		what := m.confirm
		m.confirm = ""
		m.msg = ""
		if k.String() == "y" || k.String() == "Y" {
			if what == "reset" {
				return m, m.reset()
			}
			m.leave()
			quitSession()
			return m, tea.Quit
		}
		m.refresh(false)
		return m, nil
	}
	if m.help {
		m.help = false
		m.refresh(true)
		return m, nil
	}
	switch k.String() {
	case "up", "down", "pgup", "pgdown", "k", "j", "home", "end":
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(k)
		return m, cmd
	case "f":
		// Zoom this pane to fill the window, or put the three panes back.
		// tmux resizes the pane, and Bubble Tea redraws on the resize.
		tmux("resize-pane", "-Z", "-t", tmuxSession+":Lesson.0")
		return m, nil
	case "ctrl+c":
		m.confirm = "quit"
		m.refresh(false)
		return m, nil
	}
	if m.running {
		return m, nil
	}
	n := len(m.steps())
	switch k.String() {
	case "enter":
		m.msg = ""
		switch {
		case m.pos == n:
			return m, m.check()
		case m.pos >= 0 && m.pos == m.ran && m.steps()[m.pos].Lang != "":
			return m, m.run(m.pos) // the step on screen has not run yet
		case m.pos >= 0 && m.pos == m.ran:
			m.ran++ // a closing note with no command: reading it is enough
			markWatched(m.entry.Name, m.ran+1)
		}
		m.pos++ // show the next page; a step waits for Enter before it runs
		m.refresh(true)
	case "b":
		if m.pos > m.start() {
			m.pos--
		}
		m.msg = ""
		m.refresh(true)
	case "c":
		return m, m.check()
	case "h":
		m.hint = true
		m.pos = n
		m.msg = ""
		m.refresh(true)
	case "s", "n":
		return m, m.nextLesson()
	case "p":
		return m, m.prevLesson()
	case "?":
		m.help = true
		m.refresh(true)
	case "r":
		m.confirm = "reset"
		m.refresh(false)
	case "q":
		m.confirm = "quit"
		m.refresh(false)
	}
	return m, nil
}

func (m *model) run(idx int) tea.Cmd {
	m.running = true
	m.pos = idx
	m.refresh(true)
	s, step := m.sess, m.steps()[idx]
	return func() tea.Msg {
		out, err := runStep(s, step)
		return stepDoneMsg{idx, out, err}
	}
}

// afterRun says what Enter does next, once a step has run.
func (m *model) afterRun() string {
	if m.ran == len(m.steps()) {
		return "That was the last step. Press Enter for the challenge."
	}
	return fmt.Sprintf("Press Enter for step %d.", m.ran+1)
}

// reset rebuilds the playground, reconnects the SQL and Live View panes,
// and goes back to the start of the lesson.
func (m *model) reset() tea.Cmd {
	m.running = true
	m.msg = "Resetting..."
	m.refresh(false)
	if m.sess != nil {
		m.sess.Close()
		m.sess = nil
	}
	e := m.entry
	return func() tea.Msg {
		if err := resetPlayground(e.Lesson); err != nil {
			return lessonMsg{err: err, what: "reset the lesson"}
		}
		return lessonMsg{entry: e, err: switchLesson(e), what: "reset the lesson"}
	}
}

func (m *model) check() tea.Cmd {
	m.running = true
	m.msg = "Checking..."
	m.refresh(false)
	l := m.entry.Lesson
	return func() tea.Msg {
		ok, err := check(l)
		return checkDoneMsg{ok, err}
	}
}

// leave runs a server-wide lesson's cleanup when the learner moves on.
func (m *model) leave() {
	if m.entry.Section.ServerWide {
		runCleanup(m.entry.Lesson)
	}
}

func (m *model) nextLesson() tea.Cmd {
	next := m.course.Next(m.entry)
	if next == nil {
		m.msg = "That is the last lesson in the course. See your progress with: pgdemo list"
		m.refresh(false)
		return nil
	}
	return m.openLesson(next, "open the next lesson")
}

func (m *model) prevLesson() tea.Cmd {
	prev := m.course.Prev(m.entry)
	if prev == nil {
		m.msg = "This is the first lesson."
		m.refresh(false)
		return nil
	}
	return m.openLesson(prev, "open the previous lesson")
}

func (m *model) openLesson(e *Entry, what string) tea.Cmd {
	m.running = true
	m.msg = "Opening " + e.Label() + "..."
	m.refresh(false)
	m.leave()
	return func() tea.Msg {
		if err := ensurePlayground(e.Lesson); err != nil {
			return lessonMsg{err: err, what: what}
		}
		return lessonMsg{entry: e, err: switchLesson(e), what: what}
	}
}

const footerLines = 3

func (m *model) bodyHeight() int { return max(m.height-1-footerLines, 3) }

// refresh rebuilds the pane body. top scrolls back to the start; otherwise
// it follows the output to the bottom.
func (m *model) refresh(top bool) {
	if !m.ready {
		return
	}
	m.vp.SetContent(m.body())
	if top {
		m.vp.GotoTop()
	} else {
		m.vp.GotoBottom()
	}
}

func (m *model) render(md string) string {
	if m.md == nil {
		return md
	}
	out, err := m.md.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

func (m *model) body() string {
	l, e := m.entry.Lesson, m.entry
	n := len(l.Steps)
	if m.help {
		return m.render(helpText)
	}
	var b strings.Builder
	switch {
	case m.pos == posOverview:
		b.WriteString(m.render(m.course.Overview))
	case m.pos == -1:
		md := fmt.Sprintf("# %s\n\n*%s*\n\n", l.Title, l.TLDR)
		if e.Section.ServerWide {
			md += "> This lesson shares the whole server, not just its own playground.\n\n"
		}
		if l.Intro != "" {
			md += l.Intro + "\n\n---\n\n"
		}
		md += fmt.Sprintf("**Try it:** %d steps, then a challenge. Each step shows its command before it runs. Press **Enter** to see step 1.", n)
		b.WriteString(m.render(md))
	case m.pos < n:
		s := l.Steps[m.pos]
		b.WriteString(m.render(s.Text))
		if s.Lang == "" {
			break
		}
		what := "SQL"
		if s.Lang == "sh" {
			what = "shell script"
		}
		done := m.pos < m.ran
		label := labelRun.Render("▶ Press Enter to run this " + what + ":")
		switch {
		case m.running:
			label = labelRun.Render("Running this " + what + ":")
		case done:
			label = labelRan.Render("✓ You ran:")
		}
		b.WriteString("\n\n" + label + "\n")
		b.WriteString(m.render(fmt.Sprintf("```%s\n%s\n```", fence(s.Lang), s.Code)))
		if !done && !m.running {
			break
		}
		b.WriteString("\n\n" + labelRan.Render("Result:") + "\n")
		switch out, ok := m.outputs[m.pos]; {
		case m.running:
			b.WriteString(dimStyle.Render("  Running..."))
		case !ok:
			b.WriteString(dimStyle.Render("  This step ran on an earlier visit. Press r to reset the lesson and run it again."))
		case out == "":
			b.WriteString(dimStyle.Render("  (no output)"))
		default:
			b.WriteString("  " + strings.ReplaceAll(outStyle.Render(out), "\n", "\n  "))
		}
	default:
		md := "**Your turn.** " + l.Challenge + "\n\nType your answer in the **SQL** pane, then press **Enter** here to check it."
		if m.hint {
			md += "\n\n**Hint:** " + l.Hint
		}
		b.WriteString(m.render(md))
	}
	return b.String()
}

func (m *model) View() string {
	if !m.ready {
		return ""
	}
	l, e := m.entry.Lesson, m.entry
	n := len(l.Steps)
	where := "Intro"
	switch {
	case m.pos >= 0 && m.pos < n:
		where = fmt.Sprintf("Step %d of %d", m.pos+1, n)
	case m.pos == n:
		where = "Challenge"
	}
	head := headStyle.Render(fmt.Sprintf(" Section %d · %s", e.SectionNum, e.Section.Title)) + dimStyle.Render("  "+where)
	if m.pos == posOverview {
		head = headStyle.Render(" Course Overview")
	}
	if p := getProgressCached(e.Name); p == "done" {
		head += okStyle.Render("  ● done")
	}

	k := func(key, label string) string { return keyStyle.Render(key) + " " + label }
	enter := "next"
	if m.pos >= 0 && m.pos < n && m.pos == m.ran && l.Steps[m.pos].Lang != "" {
		enter = "run"
	}
	keys1 := strings.Join([]string{k("Enter", enter), k("b", "back"), k("r", "reset"), k("c", "check"), k("h", "hint"), k("s", "skip")}, "  ")
	keys2 := strings.Join([]string{k("p", "prev lesson"), k("n", "next lesson"), k("f", "full screen"), k("?", "help"), k("q", "quit")}, "  ")
	status := m.msg
	if status == "" {
		status = m.nextHint()
	}
	// On a long page, say there's more, but never at the cost of the hint.
	if more := keyStyle.Render("↓ More below") + dimStyle.Render(", ↑ ↓ to scroll · "); !m.vp.AtBottom() &&
		lipgloss.Width(more+status) <= m.width {
		status = more + status
	}
	switch m.confirm {
	case "quit":
		status = warnStyle.Render("Quit and stop the database? y/n")
	case "reset":
		status = warnStyle.Render("Reset this lesson? Your work in its playground is wiped. y/n")
	}
	return head + "\n" + m.vp.View() + "\n" + truncate(status, m.width) + "\n " + keys1 + "\n " + keys2
}

// getProgressCached avoids a query on every frame for the header badge.
var stateCache = map[string]string{}

func getProgressCached(name string) string {
	if v, ok := stateCache[name]; ok && v == "done" {
		return v
	}
	v := getProgress(name).State
	stateCache[name] = v
	return v
}

func truncate(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

const helpText = "## Keys\n\n" +
	"| Key | Action |\n|---|---|\n" +
	"| Enter | Run the step on screen, then move to the next one. On the challenge, check it |\n" +
	"| b | Back one step |\n| r | Reset the lesson: rebuild its playground and start again |\n| c | Check |\n| h | Hint |\n| s | Skip the challenge |\n" +
	"| p | Previous lesson |\n| n | Next lesson |\n| f | Full screen: show only this pane. Press again for all three |\n| ? | This help |\n| q | Quit |\n" +
	"| ↑ ↓ | Scroll this pane |\n\n" +
	"Click a pane to type in it. Click **Browse** or **Top** at the bottom to switch tabs.\n\n" +
	"## Commands\n\nRun these in your terminal when you are not in a lesson:\n\n" +
	"```\npgdemo list          All lessons, with progress\npgdemo 5             Open lesson 5\n" +
	"pgdemo 5 info        The full written lesson\n" +
	"pgdemo help          Every command\n```\n\nPress any key to go back."
