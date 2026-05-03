package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	omniq "github.com/not-empty/omniq-go"
	"github.com/redis/go-redis/v9"
)

type resultView struct {
	SDK            string         `json:"sdk"`
	Queue          string         `json:"queue"`
	HandledJobIDs  []string       `json:"handled_job_ids"`
	FirstJobState  string         `json:"first_job_state"`
	SecondJobState string         `json:"second_job_state"`
	Stats          map[string]int `json:"stats"`
}

func main() {
	queue := getenv("QUEUE", "validation-s26-go")
	baseNowMs := int64(1775430000000)
	firstJob := queue + "-job-001"
	secondJob := queue + "-job-002"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	inspect := newRawRedis()
	defer func() { _ = inspect.Close() }()

	handled := []string{}
	sigSent := false

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: firstJob, Payload: map[string]any{"kind": "drain-true", "slot": 1}, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: secondJob, Payload: map[string]any{"kind": "drain-true", "slot": 2}, NowMsOverride: baseNowMs + 2})

	err = client.Consume(omniq.ConsumeOpts{
		Queue:            queue,
		PollIntervalS:    0.02,
		PromoteIntervalS: 10.0,
		ReapIntervalS:    10.0,
		Drain:            true,
		Handler: func(job omniq.JobCtx) {
			handled = append(handled, job.JobID)
			if job.JobID == firstJob && !sigSent {
				sigSent = true
				go func() {
					time.Sleep(100 * time.Millisecond)
					_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
				}()
			}
			time.Sleep(750 * time.Millisecond)
		},
	})
	if err != nil {
		fail(err)
	}

	statsKey := "{" + queue + "}:stats"
	out := resultView{
		SDK:            "go",
		Queue:          queue,
		HandledJobIDs:  handled,
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
