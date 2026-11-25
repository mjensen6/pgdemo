package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Section is a themed group of lessons, as written in course.yaml.
type Section struct {
	Title      string   `yaml:"title"`
	ServerWide bool     `yaml:"server_wide"`
	Lessons    []string `yaml:"lessons"`
}

// Entry is one lesson's place in the course. Lesson is nil while the
// lesson is still "coming soon" (it has no Markdown file yet).
type Entry struct {
	Num        int
	Name       string
	SectionNum int
	Section    *Section
	Lesson     *Lesson
}

type Course struct {
	Sections []*Section `yaml:"sections"`
	Entries  []*Entry   `yaml:"-"`
	Overview string     `yaml:"-"` // lessons/overview.md, shown before the first lesson
}

var lessonName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// loadCourse reads course.yaml and every lesson file it names. Numbers are
// computed from the order, so a lesson's name is its only permanent ID.
func loadCourse(dir string) (*Course, error) {
	src, err := os.ReadFile(filepath.Join(dir, "course.yaml"))
	if err != nil {
		return nil, err
	}
	var c Course
	if err := yaml.Unmarshal(src, &c); err != nil {
		return nil, fmt.Errorf("course.yaml: %w", err)
	}
	if md, err := os.ReadFile(filepath.Join(dir, "overview.md")); err == nil {
		c.Overview = string(md)
	}
	seen := map[string]bool{}
	for si, s := range c.Sections {
		for _, name := range s.Lessons {
			if !lessonName.MatchString(name) {
				return nil, fmt.Errorf("course.yaml: %q is not a valid lesson name", name)
			}
			if seen[name] {
				return nil, fmt.Errorf("course.yaml: lesson %q is listed twice", name)
			}
			seen[name] = true
			e := &Entry{Num: len(c.Entries) + 1, Name: name, SectionNum: si + 1, Section: s}
			md, err := os.ReadFile(filepath.Join(dir, name+".md"))
			switch {
			case errors.Is(err, os.ErrNotExist):
			case err != nil:
				return nil, err
			default:
				if e.Lesson, err = parseLesson(name, md); err != nil {
					return nil, err
				}
			}
			c.Entries = append(c.Entries, e)
		}
	}
	return &c, nil
}

// Find accepts a lesson number or name.
func (c *Course) Find(arg string) (*Entry, error) {
	if n, err := strconv.Atoi(arg); err == nil {
		if n < 1 || n > len(c.Entries) {
			return nil, fmt.Errorf("there is no lesson %d. Lessons run from 1 to %d. Try: pgdemo list", n, len(c.Entries))
		}
		return c.Entries[n-1], nil
	}
	for _, e := range c.Entries {
		if e.Name == arg {
			return e, nil
		}
	}
	return nil, fmt.Errorf("there is no lesson called %q. Try: pgdemo list", arg)
}

func (c *Course) ByName(name string) *Entry {
	for _, e := range c.Entries {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// Next returns the next lesson after e that is ready to play, or nil.
func (c *Course) Next(e *Entry) *Entry {
	for _, n := range c.Entries[e.Num:] {
		if n.Lesson != nil {
			return n
		}
	}
	return nil
}

// Prev returns the nearest lesson before e that is ready to play, or nil.
func (c *Course) Prev(e *Entry) *Entry {
	for i := e.Num - 2; i >= 0; i-- {
		if c.Entries[i].Lesson != nil {
			return c.Entries[i]
		}
	}
	return nil
}

func (c *Course) First() *Entry {
	for _, e := range c.Entries {
		if e.Lesson != nil {
			return e
		}
	}
	return nil
}

// Playground is the name of the lesson's own database.
func playground(name string) string {
	return "lesson_" + strings.ReplaceAll(name, "-", "_")
}

func (e *Entry) Label() string {
	if e.Lesson == nil {
		return fmt.Sprintf("Lesson %d: %s", e.Num, e.Name)
	}
	return fmt.Sprintf("Lesson %d: %s", e.Num, e.Lesson.Title)
}
