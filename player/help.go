package main

import (
	"fmt"
	"sort"
	"strings"
)

const usage = `pgdemo: learn modern PostgreSQL in one-minute lessons.

Start
  pgdemo                  Start, or resume where you left off
  pgdemo list             All lessons by Section, with progress
  pgdemo section 8        Start a Section (pgdemo section lists them)
  pgdemo 5                Open lesson 5 (a name works too: pgdemo repack)

One lesson
  pgdemo 5 tldr           What it is and why it matters. Runs nothing
  pgdemo 5 info           The full written lesson. Runs nothing
  pgdemo 5 run            Run every step non-stop
  pgdemo 5 view           Reopen the Live View
  pgdemo 5 reset          Wipe the playground and return to step 1
  pgdemo 5 check          Verify the lesson was completed
  pgdemo 5 hint           Hint for the challenge
  pgdemo 5 solution       Show the answer

Explore
  pgdemo browse           Explore the database with lazysql
  pgdemo top              See live server activity with pg_activity
  pgdemo psql             Open plain psql
  pgdemo connect          Optional: connect a desktop app like TablePlus

Course
  pgdemo check            Re-verify every lesson you have opened
  pgdemo setup            Check Docker and prepare everything
  pgdemo doctor           Diagnose problems in plain English
  pgdemo stop             Stop the database
  pgdemo uninstall        Remove the container, image, and saved data
  pgdemo help check       One command explained, with an example
`

type helpTopic struct{ usage, about, example string }

// helpTopics explains one command each, for pgdemo help <command>.
var helpTopics = map[string]helpTopic{
	"list": {"pgdemo list",
		"Shows every lesson, grouped by Section, with your progress: ○ not started, ◐ watched, ● done. Lessons that aren't written yet show as coming soon.",
		"pgdemo list"},
	"section": {"pgdemo section [N]",
		"With no number, lists the Sections and how far you are in each. With a number, opens that Section at its first lesson you haven't finished.",
		"pgdemo section 8"},
	"open": {"pgdemo N   or   pgdemo NAME",
		"Opens a lesson in the three-pane screen: the Lesson pane, your SQL prompt, and the Live View. Every lesson command takes a number or a name.",
		"pgdemo 31   (same as: pgdemo repack)"},
	"tldr": {"pgdemo N tldr",
		"What the lesson is about and why it matters, in a few lines. Runs nothing.",
		"pgdemo 1 tldr"},
	"info": {"pgdemo N info",
		"Prints the whole written lesson, including its steps and challenge, without running anything. Handy for reading ahead.",
		"pgdemo 2 info"},
	"run": {"pgdemo N run",
		"Runs every step of a lesson non-stop and prints the output. It counts as watched, not done. Only solving the challenge marks a lesson done.",
		"pgdemo 1 run"},
	"view": {"pgdemo N view",
		"Opens a lesson's Live View in this terminal, refreshing every second. Use it if you closed the Live View pane, or want it in a second window. Press q to close it.",
		"pgdemo 1 view"},
	"reset": {"pgdemo N reset",
		"Wipes the lesson's playground database, rebuilds it, and goes back to step 1. A lesson you finished stays done.",
		"pgdemo 1 reset"},
	"check": {"pgdemo check   or   pgdemo N check",
		"Checks whether a challenge is solved, by looking at the playground, not at what you typed, so any correct approach passes. With no lesson, re-checks every lesson you have opened.",
		"pgdemo 2 check"},
	"hint": {"pgdemo N hint",
		"Shows the hint for a lesson's challenge.",
		"pgdemo 1 hint"},
	"solution": {"pgdemo N solution",
		"Shows the answer to a lesson's challenge. Try the hint first.",
		"pgdemo 1 solution"},
	"browse": {"pgdemo browse [N]",
		"Opens lazysql, a full-screen database explorer, on a lesson's playground: the one you worked on last, or lesson N. You can also click the Browse tab inside a lesson. Press ? in lazysql for its keys.",
		"pgdemo browse 2"},
	"top": {"pgdemo top",
		"Opens pg_activity, which shows what the server is doing right now: running queries, how long they've taken, and what's waiting on locks. Same as the Top tab inside a lesson. Press q to quit.",
		"pgdemo top"},
	"psql": {"pgdemo psql",
		"Opens plain psql, connected to the postgres database. Use \\c lesson_mvcc_bloat to switch to a lesson's playground.",
		"pgdemo psql"},
	"connect": {"pgdemo connect [N]   or   pgdemo connect off",
		"Lets a desktop app like TablePlus or DBeaver connect to the course. The first time, it opens port 5439 on this computer only (127.0.0.1), so no other machine can reach it, and there's no password. It prints a connection string for a lesson's playground. pgdemo connect off closes the port again.",
		"pgdemo connect 1"},
	"setup": {"pgdemo setup",
		"Checks Docker, builds the course image, and starts the database. Plain pgdemo does this for you when needed, so you only need it after changing the Dockerfile.",
		"pgdemo setup"},
	"doctor": {"pgdemo doctor",
		"Checks Docker, the course image, the database, the extensions and tools, your playgrounds, and leftovers from server-wide lessons. Each problem comes with the command that fixes it.",
		"pgdemo doctor"},
	"stop": {"pgdemo stop",
		"Stops the database. Your progress and playgrounds are kept. Run pgdemo to start again.",
		"pgdemo stop"},
	"uninstall": {"pgdemo uninstall",
		"Removes the course container, its image, and the saved data, including your progress and every playground. It asks first. The files in this folder stay, so ./pgdemo can set it all up again.",
		"pgdemo uninstall"},
	"help": {"pgdemo help [COMMAND]",
		"Lists every command, or explains one with an example.",
		"pgdemo help check"},
}

func help(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	t, ok := helpTopics[args[0]]
	if !ok {
		var names []string
		for n := range helpTopics {
			names = append(names, n)
		}
		sort.Strings(names)
		return fmt.Errorf("there is no command %q. Help covers: %s", args[0], strings.Join(names, ", "))
	}
	fmt.Printf("%s\n\n%s\n\nExample:\n  %s\n", t.usage, wrap(t.about, 76), t.example)
	return nil
}

func wrap(s string, width int) string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	return strings.Join(append(lines, line), "\n")
}
