package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Step is one explained command: a paragraph followed by a sql or sh block.
type Step struct {
	Text string
	Lang string // "sql", "sh", or "" for a closing note with no command
	Code string
}

type Lesson struct {
	Name      string
	Title     string `yaml:"title"`
	TLDR      string `yaml:"tldr"`
	Setup     string
	View      string
	Cleanup   string
	Solution  string
	Check     string
	Intro     string // the teaching, before "## Try it"; nothing in it runs
	Steps     []Step
	Challenge string
	Hint      string
}

// parseLesson reads one lesson file. The format is plain Markdown so it
// also reads well on GitHub; fenced block names carry the meaning.
func parseLesson(name string, src []byte) (*Lesson, error) {
	fail := func(format string, a ...any) error {
		return fmt.Errorf("lessons/%s.md: %s", name, fmt.Sprintf(format, a...))
	}
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	l := &Lesson{Name: name}

	if !strings.HasPrefix(text, "---\n") {
		return nil, fail("must start with a --- front matter block holding title and tldr")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return nil, fail("front matter is not closed with ---")
	}
	if err := yaml.Unmarshal([]byte(text[4:4+end]), l); err != nil {
		return nil, fail("front matter: %v", err)
	}
	body := text[4+end+5:]

	lines := strings.Split(body, "\n")
	var prose []string
	inChallenge := false
	// With a "## Try it" heading, everything above it is the Intro, and
	// code blocks there are examples to read, not steps to run.
	inIntro := strings.Contains(body, "\n## Try it\n") || strings.HasPrefix(body, "## Try it\n")
	flushNote := func() {
		if t := strings.TrimSpace(strings.Join(prose, "\n")); t != "" {
			l.Steps = append(l.Steps, Step{Text: t})
		}
		prose = nil
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "```") {
			lang := strings.TrimSpace(line[3:])
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) != "```" {
				j++
			}
			if j == len(lines) {
				return nil, fail("a ``` block opened on line %d is never closed", i+1)
			}
			code := strings.TrimSpace(strings.Join(lines[i+1:j], "\n"))
			raw := lines[i : j+1]
			i = j
			switch lang {
			case "setup":
				l.Setup = code
			case "view":
				l.View = code
			case "cleanup":
				l.Cleanup = code
			case "solution":
				l.Solution = code
			case "check":
				l.Check = code
			case "sql", "sh":
				if inChallenge || inIntro {
					prose = append(prose, raw...)
					continue
				}
				l.Steps = append(l.Steps, Step{Text: strings.TrimSpace(strings.Join(prose, "\n")), Lang: lang, Code: code})
				prose = nil
			default:
				prose = append(prose, raw...)
			}
			continue
		}
		if inIntro && strings.TrimSpace(line) == "## Try it" {
			l.Intro = strings.TrimSpace(strings.Join(prose, "\n"))
			prose = nil
			inIntro = false
			continue
		}
		if !inChallenge && strings.TrimSpace(line) == "## Challenge" {
			flushNote()
			inChallenge = true
			continue
		}
		if inChallenge && strings.HasPrefix(line, "Hint:") {
			l.Hint = strings.TrimSpace(line[len("Hint:"):])
			continue
		}
		prose = append(prose, line)
	}
	if inChallenge {
		l.Challenge = strings.TrimSpace(strings.Join(prose, "\n"))
	} else {
		flushNote()
	}

	switch {
	case l.Title == "":
		return nil, fail("front matter needs a title")
	case l.TLDR == "":
		return nil, fail("front matter needs a tldr")
	case !hasCommand(l.Steps):
		return nil, fail("needs at least one step: a paragraph followed by a ```sql or ```sh block")
	case l.Challenge == "":
		return nil, fail("needs a ## Challenge section with a task")
	case l.Hint == "":
		return nil, fail("the challenge needs a line starting with Hint:")
	case l.Solution == "":
		return nil, fail("needs a ```solution block")
	case l.Check == "":
		return nil, fail("needs a ```check block")
	}
	for i, s := range l.Steps {
		if s.Lang == "sql" && !endsStatement(s.Code) {
			return nil, fail("step %d: end the SQL with ; so the session knows it is complete", i+1)
		}
	}
	return l, nil
}

func hasCommand(steps []Step) bool {
	for _, s := range steps {
		if s.Lang != "" {
			return true
		}
	}
	return false
}

func endsStatement(code string) bool {
	lines := strings.Split(strings.TrimSpace(code), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	return strings.HasSuffix(last, ";") || strings.HasPrefix(last, `\`)
}

// Markdown renders the written lesson for `pgdemo N info`: everything a
// reader needs, without the hidden setup and check.
func (l *Lesson) Markdown(e *Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n*%s*\n\n", e.Label(), l.TLDR)
	if e.Section.ServerWide {
		b.WriteString("> This lesson shares the whole server, not just its own playground.\n\n")
	}
	if l.Intro != "" {
		b.WriteString(l.Intro + "\n\n")
	}
	for i, s := range l.Steps {
		fmt.Fprintf(&b, "## Step %d\n\n%s\n\n", i+1, s.Text)
		if s.Lang != "" {
			fmt.Fprintf(&b, "```%s\n%s\n```\n\n", fence(s.Lang), s.Code)
		}
	}
	fmt.Fprintf(&b, "## Challenge\n\n%s\n\nHint: %s\n", l.Challenge, l.Hint)
	return b.String()
}

// fence maps a step's language to one the highlighter knows.
func fence(lang string) string {
	if lang == "sh" {
		return "bash"
	}
	return lang
}
