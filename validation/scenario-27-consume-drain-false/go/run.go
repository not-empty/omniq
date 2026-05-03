package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	omniq "github.com/not-empty/omniq-go"
	"github.com/redis/go-redis/v9"
)

type resultView struct {
	SDK            string         `json:"sdk"`
	Queue          string         `json:"queue"`
	ChildExitCode  int            `json:"child_exit_code"`
	HandlerStarted bool           `json:"handler_started"`
	HandlerDone    bool           `json:"handler_done"`
	FirstJobState  string         `json:"first_job_state"`
	SecondJobState string         `json:"second_job_state"`
	Stats          map[string]int `json:"stats"`
}

func childMain() {
	queue := os.Getenv("QUEUE")
	markerStarted := os.Getenv("MARKER_STARTED")
	markerDone := os.Getenv("MARKER_DONE")

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	marker := newRawRedis()
	defer func() { _ = marker.Close() }()

	err = client.Consume(omniq.ConsumeOpts{
		Queue:            queue,
		PollIntervalS:    0.02,
		PromoteIntervalS: 10.0,
		ReapIntervalS:    10.0,
		Drain:            false,
		Handler: func(job omniq.JobCtx) {
			_ = marker.Set(ctx, markerStarted, "1", 0).Err()
			time.Sleep(1500 * time.Millisecond)
			_ = marker.Set(ctx, markerDone, "1", 0).Err()
		},
	})
	if err != nil {
		fail(err)
	}
}

func parentMain() {
	queue := getenv("QUEUE", "validation-s27-go")
	baseNowMs := int64(1775440000000)
	firstJob := queue + "-job-001"
	secondJob := queue + "-job-002"
	markerStarted := "{" + queue + "}:marker:started"
	markerDone := "{" + queue + "}:marker:done"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	inspect := newRawRedis()
	defer func() { _ = inspect.Close() }()

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: firstJob, Payload: map[string]any{"kind": "drain-false", "slot": 1}, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: secondJob, Payload: map[string]any{"kind": "drain-false", "slot": 2}, NowMsOverride: baseNowMs + 2})

	binaryPath := filepath.Join(os.TempDir(), "omniq-s27-go-child")
	buildCmd := exec.Command("/usr/local/go/bin/go", "build", "-buildvcs=false", "-o", binaryPath, ".")
	buildCmd.Dir = "."
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fail(err)
	}

	cmd := exec.Command(binaryPath, "child")
	cmd.Env = append(os.Environ(),
		"QUEUE="+queue,
		"MARKER_STARTED="+markerStarted,
		"MARKER_DONE="+markerDone,
	)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fail(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v, _ := inspect.Get(ctx, markerStarted).Result(); v == "1" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = cmd.Process.Signal(syscall.SIGINT)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	exitCode := 0
	select {
	case err := <-done:
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1
			}
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		if err := <-done; err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1
			}
		}
	}

	statsKey := "{" + queue + "}:stats"
	out := resultView{
		SDK:            "go",
		Queue:          queue,
		ChildExitCode:  exitCode,
		HandlerStarted: getString(inspect, ctx, markerStarted) == "1",
		HandlerDone:    getString(inspect, ctx, markerDone) == "1",
		FirstJobState:  hgetString(inspect, ctx, "{"+queue+"}:job:"+firstJob, "state"),
		SecondJobState: hgetString(inspect, ctx, "{"+queue+"}:job:"+secondJob, "state"),
		Stats: map[string]int{
			"waiting":        hgetInt(inspect, ctx, statsKey, "waiting"),
			"waiting_total":  hgetInt(inspect, ctx, statsKey, "waiting_total"),
			"active":         hgetInt(inspect, ctx, statsKey, "active"),
			"completed_kept": hgetInt(inspect, ctx, statsKey, "completed_kept"),
		},
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(b))
}

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) {
	if _, err := client.Publish(opts); err != nil {
		fail(err)
	}
}

func getString(r redis.UniversalClient, ctx context.Context, key string) string {
	v, err := r.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return v
}

func hgetString(r redis.UniversalClient, ctx context.Context, key, field string) string {
	v, err := r.HGet(ctx, key, field).Result()
	if err != nil {
		return ""
	}
	return v
}

func hgetInt(r redis.UniversalClient, ctx context.Context, key, field string) int {
	v, err := r.HGet(ctx, key, field).Int()
	if err != nil {
		return 0
	}
	return v
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newRawRedis() redis.UniversalClient {
	if getenv("REDIS_MODE", "standalone") == "cluster" {
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{getenv("REDIS_HOST", "omniq-redis") + ":6379"},
		})
	}

	return redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{getenv("REDIS_HOST", "omniq-redis") + ":6379"},
		DB:    0,
	})
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		childMain()
		return
	}
	parentMain()
}
