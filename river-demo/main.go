// river-demo is a tiny job queue app for the River lesson. River stores its
// jobs in Postgres tables, so this program only needs a database.
//
//	river-demo -d DB migrate        create River's tables
//	river-demo -d DB enqueue N      add N "send email" jobs
//	river-demo -d DB work SECONDS   run 3 workers for a while
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
)

// EmailArgs is one job's arguments. River stores them as JSON in river_job.args.
type EmailArgs struct {
	To string `json:"to"`
}

func (EmailArgs) Kind() string { return "send_email" }

type EmailWorker struct {
	river.WorkerDefaults[EmailArgs]
}

// Work pretends to send an email. Addresses containing "flaky" fail on the
// first attempt, so the lesson can show River retrying a job.
func (w *EmailWorker) Work(ctx context.Context, job *river.Job[EmailArgs]) error {
	time.Sleep(200 * time.Millisecond)
	if strings.Contains(job.Args.To, "flaky") && job.Attempt == 1 {
		fmt.Printf("worker: job %d to %s failed on attempt %d, River will retry it\n", job.ID, job.Args.To, job.Attempt)
		return errors.New("mail server timed out")
	}
	fmt.Printf("worker: job %d sent email to %s (attempt %d)\n", job.ID, job.Args.To, job.Attempt)
	return nil
}

// Retry after 2 seconds instead of River's usual growing backoff, so the
// lesson doesn't have to wait.
type quickRetry struct{}

func (quickRetry) NextRetry(job *rivertype.JobRow) time.Time { return time.Now().Add(2 * time.Second) }

func main() {
	db := flag.String("d", os.Getenv("PGDATABASE"), "database name")
	flag.Parse()
	if err := run(*db, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "river-demo:", err)
		os.Exit(1)
	}
}

func run(db string, args []string) error {
	if db == "" || len(args) == 0 {
		return errors.New("usage: river-demo -d DB migrate | enqueue N | work SECONDS")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres:///"+db+"?host=/var/run/postgresql&user=postgres")
	if err != nil {
		return err
	}
	defer pool.Close()
	driver := riverpgxv5.New(pool)

	switch args[0] {
	case "migrate":
		m, err := rivermigrate.New(driver, nil)
		if err != nil {
			return err
		}
		res, err := m.Migrate(ctx, rivermigrate.DirectionUp, nil)
		if err != nil {
			return err
		}
		fmt.Printf("River's tables are ready (%d migrations applied).\n", len(res.Versions))
		return nil

	case "enqueue":
		n := 5
		if len(args) > 1 {
			if n, err = strconv.Atoi(args[1]); err != nil {
				return err
			}
		}
		client, err := river.NewClient(driver, &river.Config{})
		if err != nil {
			return err
		}
		for i := 1; i <= n; i++ {
			to := fmt.Sprintf("user%d@example.com", i)
			if i == 2 {
				to = "flaky@example.com"
			}
			res, err := client.Insert(ctx, EmailArgs{To: to}, nil)
			if err != nil {
				return err
			}
			fmt.Printf("enqueued job %d: send email to %s\n", res.Job.ID, to)
		}
		return nil

	case "work":
		secs := 5
		if len(args) > 1 {
			if secs, err = strconv.Atoi(args[1]); err != nil {
				return err
			}
		}
		workers := river.NewWorkers()
		river.AddWorker(workers, &EmailWorker{})
		client, err := river.NewClient(driver, &river.Config{
			Queues:            map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 3}},
			Workers:           workers,
			RetryPolicy:       quickRetry{},
			FetchPollInterval: 200 * time.Millisecond,
			Logger:            nil,
		})
		if err != nil {
			return err
		}
		if err := client.Start(ctx); err != nil {
			return err
		}
		fmt.Printf("3 workers running for %d seconds...\n", secs)
		time.Sleep(time.Duration(secs) * time.Second)
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return client.Stop(stopCtx)
	}
	return fmt.Errorf("unknown command %q", args[0])
}
