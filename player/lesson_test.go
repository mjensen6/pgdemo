package main

import "testing"

func TestCourseLoads(t *testing.T) {
	c, err := loadCourse("../lessons")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Entries) != 43 {
		t.Fatalf("want 43 lessons, got %d", len(c.Entries))
	}
	want := map[string]int{"mvcc-bloat": 1, "transactional-ddl": 2, "repack": 31}
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
