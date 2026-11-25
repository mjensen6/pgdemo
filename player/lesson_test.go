package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCourseLoads(t *testing.T) {
	c, err := loadCourse("../lessons")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Entries) != 43 {
		t.Fatalf("want 43 lessons, got %d", len(c.Entries))
	}
	want := map[string]int{"mvcc-bloat": 1, "transactional-ddl": 2, "merge": 5, "jsonb": 10,
		"skip-locked": 19, "partitioning": 26, "repack": 31, "postgis": 36, "pgbouncer": 43}
	for name, num := range want {
		e := c.ByName(name)
		if e == nil || e.Num != num || e.Lesson == nil {
			t.Fatalf("%s: want built lesson %d, got %+v", name, num, e)
		}
	}
	if c.Entries[42].Section.ServerWide != true {
		t.Fatal("pgbouncer should be in the server-wide section")
	}
}

func TestParseLesson(t *testing.T) {
	src := "---\ntitle: T\ntldr: Short.\n---\n\n```setup\nCREATE TABLE a (id int);\n```\n\nFirst.\n\n```sql\nBEGIN;\n```\n\nSecond.\n\n```sh\necho hi\n```\n\n## Challenge\n\nDo it.\n\nHint: Like this.\n\n```solution\nCOMMIT;\n```\n\n```check\nSELECT true;\n```\n"
	l, err := parseLesson("x", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Steps) != 2 || l.Steps[0].Text != "First." || l.Steps[1].Lang != "sh" {
		t.Fatalf("steps: %+v", l.Steps)
	}
	if l.Challenge != "Do it." || l.Hint != "Like this." || l.Solution != "COMMIT;" || l.Setup == "" {
		t.Fatalf("lesson: %+v", l)
	}
	if _, err := parseLesson("x", []byte("---\ntitle: T\n---\n")); err == nil {
		t.Fatal("a lesson without a tldr should be rejected")
	}
}

func TestIntroBeforeTryIt(t *testing.T) {
	src := "---\ntitle: T\ntldr: Short.\n---\n\nTeaching here.\n\n```sql\nSELECT 'example only';\n```\n\n## Try it\n\nRun this.\n\n```sql\nSELECT 1;\n```\n\n## Challenge\n\nDo it.\n\nHint: H.\n\n```solution\nSELECT 1;\n```\n\n```check\nSELECT true;\n```\n"
	l, err := parseLesson("x", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(l.Intro, "Teaching here.") || !strings.Contains(l.Intro, "example only") {
		t.Fatalf("intro: %q", l.Intro)
	}
	if len(l.Steps) != 1 || l.Steps[0].Code != "SELECT 1;" || l.Steps[0].Text != "Run this." {
		t.Fatalf("an example in the intro must not become a step: %+v", l.Steps)
	}
}

func TestPrevSkipsComingSoon(t *testing.T) {
	// A made-up course where lesson 2 is coming soon (it has no file).
	dir := t.TempDir()
	lesson := "---\ntitle: T\ntldr: Short.\n---\n\nStep.\n\n```sql\nSELECT 1;\n```\n\n## Challenge\n\nDo it.\n\nHint: This.\n\n```solution\nSELECT 1;\n```\n\n```check\nSELECT true;\n```\n"
	files := map[string]string{
		"course.yaml": "sections:\n  - title: S\n    lessons: [one, two, three]\n",
		"one.md":      lesson,
		"three.md":    lesson,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loadCourse(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p := c.Prev(c.ByName("one")); p != nil {
		t.Fatalf("first lesson has a previous lesson: %s", p.Name)
	}
	if p := c.Prev(c.ByName("three")); p == nil || p.Name != "one" {
		t.Fatalf("Prev(three) = %v, want one, skipping the coming-soon two", p)
	}
	if n := c.Next(c.ByName("one")); n == nil || n.Name != "three" {
		t.Fatalf("Next(one) = %v, want three", n)
	}
}

func TestExpectAttachesToStep(t *testing.T) {
	src := "---\ntitle: T\ntldr: Short.\n---\n\n## Try it\n\nFirst.\n\n```sql\nSELECT 1;\n```\n\n```expect\nhello\n\n  world  \n```\n\nSecond.\n\n```sql\nSELECT 2;\n```\n\n## Challenge\n\nDo it.\n\nHint: This.\n\n```solution\nSELECT 1;\n```\n\n```check\nSELECT true;\n```\n"
	l, err := parseLesson("x", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Steps) != 2 || strings.Join(l.Steps[0].Expect, "|") != "hello|world" || l.Steps[1].Expect != nil {
		t.Fatalf("expect lines landed wrong: %+v", l.Steps)
	}
	bad := strings.Replace(src, "First.\n\n```sql\nSELECT 1;\n```\n\n```expect", "```expect", 1)
	if _, err := parseLesson("x", []byte(bad)); err == nil {
		t.Fatal("an expect block before any step should be refused")
	}
}
