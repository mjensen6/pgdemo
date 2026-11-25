package main

// The Phase 2 tools: section, browse, top, view, doctor, and help topics.
// connect and uninstall live in the host launcher, because they change the
// container itself; the player only supplies the connection details.

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// --- section ----------------------------------------------------------------

// sectionCmd lists the Sections, or opens one: its first lesson that is
// built and not done yet, or its first built lesson if all are done.
func sectionCmd(c *Course, args []string) error {
	if len(args) == 0 {
		prog := allProgress()
		last := 0
		for _, e := range c.Entries {
			if e.SectionNum == last {
				continue
			}
			last = e.SectionNum
			built, done, total := 0, 0, 0
			for _, x := range c.Entries {
				if x.SectionNum != e.SectionNum {
					continue
				}
				total++
				if x.Lesson != nil {
					built++
					if prog[x.Name].State == "done" {
						done++
					}
				}
			}
			status := fmt.Sprintf("%d of %d done", done, built)
			if built == 0 {
				status = "coming soon"
			}
			fmt.Printf("  %2d  %-26s %2d lessons   %s\n", e.SectionNum, e.Section.Title, total, status)
		}
		fmt.Println("\nOpen one with: pgdemo section 8")
		return nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 || n > len(c.Sections) {
		return fmt.Errorf("there is no Section %q. Sections run from 1 to %d. See them with: pgdemo section", args[0], len(c.Sections))
	}
	var first, todo *Entry
	var soon []string
	for _, e := range c.Entries {
		if e.SectionNum != n {
			continue
		}
		if e.Lesson == nil {
			soon = append(soon, e.Name)
			continue
		}
		if first == nil {
			first = e
		}
		if todo == nil && getProgress(e.Name).State != "done" {
			todo = e
		}
	}
	if first == nil {
		return fmt.Errorf("Section %d (%s) is coming soon: %s. See what is ready with: pgdemo list",
			n, c.Sections[n-1].Title, strings.Join(soon, ", "))
	}
	if todo == nil {
		todo = first
	}
	return openLesson(c, todo)
}

// --- browse, top, view ------------------------------------------------------

// defaultEntry is the lesson a course-wide tool should point at: the one
// named in args, or else the lesson the learner worked on last.
func defaultEntry(c *Course, args []string) (*Entry, error) {
	if len(args) > 0 {
		e, err := c.Find(args[0])
		if err != nil {
			return nil, err
		}
		if e.Lesson == nil {
			return nil, fmt.Errorf("lesson %d (%s) is coming soon. See what is ready with: pgdemo list", e.Num, e.Name)
		}
		return e, nil
	}
	if e := c.ByName(lastLesson()); e != nil && e.Lesson != nil {
		return e, nil
	}
	return c.First(), nil
}

func lazysqlCmd(db string) *exec.Cmd {
	url := "postgres://postgres@localhost:5432/" + db + "?sslmode=disable"
	cmd := exec.Command("lazysql", "--no-telemetry", url)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}

// browse opens lazysql full screen on a lesson's playground.
func browse(c *Course, args []string) error {
	e, err := defaultEntry(c, args)
	if err != nil {
		return err
	}
	if err := ensurePlayground(e.Lesson); err != nil {
		return err
	}
	fixTerm()
	fmt.Printf("Opening %s's playground in lazysql. Press ? for its keys, q to quit.\n", e.Label())
	return lazysqlCmd(playground(e.Name)).Run()
}

// browsePane keeps lazysql open in the Browse tab.
func browsePane(db string) {
	for {
		lazysqlCmd(db).Run()
		fmt.Print("\nlazysql closed. Press Enter to reopen it.")
		fmt.Scanln()
	}
}

func top() error {
	fixTerm()
	return syscall.Exec("/usr/bin/pg_activity", []string{"pg_activity", "-d", "postgres", "--no-db-size"}, os.Environ())
}

// view shows a lesson's Live View in this terminal until q.
func view(e *Entry) error {
	l := e.Lesson
	if l.View == "" {
		return fmt.Errorf("lesson %d has no Live View", e.Num)
	}
	if err := ensurePlayground(l); err != nil {
		return err
	}
	fixTerm()
	cmd := exec.Command("pspg", "-d", playground(e.Name), "--query", l.View, "--watch", "1", "--no-commandbar")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// connectInfo prints the playground a desktop app should open. The host
// launcher turns it into a connection string.
func connectInfo(c *Course, args []string) error {
	e, err := defaultEntry(c, args)
	if err != nil {
		return err
	}
	if err := ensurePlayground(e.Lesson); err != nil {
		return err
	}
	fmt.Printf("%s\t%s\n", playground(e.Name), e.Label())
	return nil
}

// --- doctor -----------------------------------------------------------------

type doctorReport struct{ problems int }

func (d *doctorReport) ok(msg string) { fmt.Println("  ✓ " + msg) }
func (d *doctorReport) bad(msg, fix string) {
	d.problems++
	fmt.Println("  ✗ " + msg)
	for _, line := range strings.Split(fix, "\n") {
		fmt.Println("      " + line)
	}
}

// The extensions the course uses, by the name CREATE EXTENSION takes.
var courseExtensions = []string{
	"pg_stat_statements", "pgstattuple", "pg_cron", "pg_partman", "vector", "postgis",
	"pgrouting", "pg_repack", "pg_squeeze", "hypopg", "pg_ivm", "hll", "pgaudit", "pgtap",
}

var coursePreload = []string{"pg_stat_statements", "pg_cron", "pg_squeeze", "pgaudit"}

var courseTools = []string{"tmux", "pgcli", "pspg", "pg_activity", "lazysql", "pgbouncer"}

// doctor checks everything inside the container and explains each problem
// in plain English, with the command that fixes it.
func doctor(c *Course) error {
	d := &doctorReport{}
	fmt.Println("Inside the course container")

	v, err := query("postgres", "SHOW server_version")
	if err != nil {
		d.bad("The database is not answering.", "Restart it with: pgdemo stop, then pgdemo")
		return fmt.Errorf("found %d problem(s)", d.problems)
	}
	d.ok("Postgres " + v + " is running")

	if _, err := query(progressDB, "SELECT count(*) FROM progress"); err != nil {
		d.bad("Your progress can't be read: "+firstLine(err.Error()), "Run pgdemo once to recreate it. Your playgrounds are kept.")
	} else {
		d.ok("Your progress is saved in the pgdemo database")
	}

	pre, _ := query("postgres", "SHOW shared_preload_libraries")
	var missingPre []string
	for _, lib := range coursePreload {
		if !strings.Contains(","+strings.ReplaceAll(pre, " ", "")+",", ","+lib+",") {
			missingPre = append(missingPre, lib)
		}
	}
	if len(missingPre) > 0 {
		d.bad("The server didn't load "+strings.Join(missingPre, ", ")+", which some lessons need.",
			"compose.yaml sets this. If you changed it, put it back, then run: pgdemo stop, then pgdemo")
	} else {
		d.ok("Server settings are in place")
	}

	avail, _ := query("postgres", "SELECT name FROM pg_available_extensions")
	have := map[string]bool{}
	for _, n := range strings.Split(avail, "\n") {
		have[n] = true
	}
	var missingExt []string
	for _, x := range courseExtensions {
		if !have[x] {
			missingExt = append(missingExt, x)
		}
	}
	if len(missingExt) > 0 {
		d.bad("These extensions are missing from the image: "+strings.Join(missingExt, ", "),
			"Rebuild the image with: pgdemo setup")
	} else {
		d.ok(fmt.Sprintf("All %d extensions the lessons use are installed", len(courseExtensions)))
	}

	var missingTools []string
	for _, t := range courseTools {
		if _, err := exec.LookPath(t); err != nil {
			missingTools = append(missingTools, t)
		}
	}
	if len(missingTools) > 0 {
		d.bad("These tools are missing from the image: "+strings.Join(missingTools, ", "), "Rebuild the image with: pgdemo setup")
	} else {
		d.ok("Terminal tools are installed: " + strings.Join(courseTools, ", "))
	}

	doctorPlaygrounds(c, d)
	doctorLeftovers(d)

	var fs syscall.Statfs_t
	if err := syscall.Statfs("/var/lib/postgresql", &fs); err == nil {
		free := fs.Bavail * uint64(fs.Bsize)
		if free < 1<<30 {
			d.bad(fmt.Sprintf("Only %d MB of disk is free for the database.", free>>20),
				"Free some space on this computer. Docker's own cleanup is: docker system prune")
		} else {
			d.ok(fmt.Sprintf("%d GB of disk is free for the database", free>>30))
		}
	}

	if d.problems > 0 {
		return fmt.Errorf("found %d problem(s) inside the container. Each one says how to fix it", d.problems)
	}
	return nil
}

func doctorPlaygrounds(c *Course, d *doctorReport) {
	out, err := query("postgres", `SELECT datname, coalesce(shobj_description(oid, 'pg_database'), '')
FROM pg_database WHERE datname LIKE 'lesson\_%' ORDER BY 1`)
	if err != nil {
		return
	}
	byDB := map[string]*Entry{}
	for _, e := range c.Entries {
		byDB[playground(e.Name)] = e
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		db, mark, found := strings.Cut(line, "|")
		if !found {
			continue
		}
		count++
		e := byDB[db]
		switch {
		case e == nil || e.Lesson == nil:
			d.bad("Database "+db+" doesn't belong to any lesson in this course.",
				"It's harmless. To remove it, run pgdemo psql, then: DROP DATABASE "+db+";")
		case mark != readyMark:
			d.bad(e.Label()+" has a half-built playground, probably from an interrupted setup.",
				fmt.Sprintf("Rebuild it with: pgdemo %d reset", e.Num))
		}
	}
	d.ok(fmt.Sprintf("%d lesson playground(s) checked", count))
}

// doctorLeftovers finds things server-wide lessons create that outlive the
// lesson: cron jobs, replication slots, and subscriptions.
func doctorLeftovers(d *doctorReport) {
	type leftover struct{ what, fix string }
	var found []leftover

	if jobs, err := query("postgres", "SELECT jobid || ' ' || jobname || ' (' || database || ')' FROM cron.job ORDER BY 1"); err == nil && jobs != "" {
		for _, j := range strings.Split(jobs, "\n") {
			id, _, _ := strings.Cut(j, " ")
			found = append(found, leftover{"Scheduled cron job " + j + " is still running.",
				"Remove it with pgdemo psql, then: SELECT cron.unschedule(" + id + ");"})
		}
	}
	if slots, err := query("postgres", "SELECT slot_name FROM pg_replication_slots ORDER BY 1"); err == nil && slots != "" {
		for _, s := range strings.Split(slots, "\n") {
			found = append(found, leftover{"Replication slot " + s + " is left over. A slot nobody reads keeps old WAL forever and slowly fills the disk.",
				"Remove it with pgdemo psql, then: SELECT pg_drop_replication_slot('" + s + "');"})
		}
	}
	if subs, err := query("postgres", "SELECT s.subname || '|' || d.datname FROM pg_subscription s JOIN pg_database d ON d.oid = s.subdbid ORDER BY 1"); err == nil && subs != "" {
		for _, line := range strings.Split(subs, "\n") {
			name, db, _ := strings.Cut(line, "|")
			found = append(found, leftover{"Subscription " + name + " in " + db + " is left over.",
				"Remove it with pgdemo psql, then: \\c " + db + "  and  DROP SUBSCRIPTION " + name + ";"})
		}
	}
	if len(found) == 0 {
		d.ok("No leftovers from server-wide lessons (cron jobs, replication slots, subscriptions)")
		return
	}
	sort.Slice(found, func(i, j int) bool { return found[i].what < found[j].what })
	for _, f := range found {
		d.bad(f.what, f.fix+"\nIf you're in the middle of a Server-Wide lesson, this is expected. Leaving or resetting it cleans up.")
	}
}
