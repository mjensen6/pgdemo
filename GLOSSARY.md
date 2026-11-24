# pgdemo

A terminal-based course that teaches modern PostgreSQL through short, runnable lessons.

## Language

**Course**:
The complete set of lessons.
_Avoid_: Demo, tutorial

**Section**:
A themed group of lessons within the course.
_Avoid_: Part, module, chapter

**Lesson**:
One topic, taking about a minute to work through.
_Avoid_: Demo, exercise

**Intro**:
The teaching part at the start of a lesson, before its steps. It explains the idea and runs nothing.
_Avoid_: Overview, preamble

**Step**:
One explained command inside a lesson.
_Avoid_: Stage, instruction

**Challenge**:
The small task a learner types themselves at the end of a lesson.
_Avoid_: Quiz, test, exercise

**Playground**:
The database belonging to a single lesson, which the learner is free to break.
_Avoid_: Sandbox, lesson database

**Course Overview**:
The first page of the first lesson, describing the screen before any lesson starts.
_Avoid_: Welcome screen, intro, tutorial

**Lesson Pane**:
The left pane, which explains each step and runs it when the learner presses Enter.
_Avoid_: Main pane, guide

**SQL Pane**:
The learner's own prompt, connected to the current lesson's playground.
_Avoid_: Console, terminal, shell

**Live View**:
A continuously refreshing display of the numbers a lesson wants the learner to watch.
_Avoid_: Monitor, dashboard

**Check**:
A verification of a playground's current state that confirms a lesson or challenge was completed.
_Avoid_: Test, grade, validation

**Progress**:
A learner's state for each lesson: not started, watched, or done.
_Avoid_: Score, completion

**Watched**:
A lesson whose steps have run but whose check has not passed.
_Avoid_: Viewed, partial

**Done**:
A lesson whose check has passed.
_Avoid_: Complete, finished

**Server-Wide Lesson**:
A lesson whose effects reach beyond its playground to the whole server.
_Avoid_: Global lesson, shared lesson
