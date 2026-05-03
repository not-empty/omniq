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

type seenItem struct {
	Attempt       int  `json:"attempt"`
	MaxAttempts   int  `json:"max_attempts"`
	IsLastAttempt bool `json:"is_last_attempt"`
}

type resultView struct {
	SDK        string     `json:"sdk"`
	Queue      string     `json:"queue"`
	JobID      string     `json:"job_id"`
	Seen       []seenItem `json:"seen"`
	FinalState string     `json:"final_state"`
}

func main() {
	queue := getenv("QUEUE", "validation-s28-go")
	jobID := queue + "-job-001"
	baseNowMs := int64(1775440000000)

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	inspect := newRawRedis()
	defer func() { _ = inspect.Close() }()

	seen := []seenItem{}
	sigSent := false

	_, err = client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: jobID,
		Payload: map[string]any{
			"kind": "consume-max-attempts",
			"sdk":  "go",
		},
		MaxAttempts:   3,
		Backoff:       100,
		Timeout:       30_000,
		NowMsOverride: baseNowMs + 1,
	})
	if err != nil {
		fail(err)
	}

	err = client.Consume(omniq.ConsumeOpts{
		Queue:            queue,
		PollIntervalS:    0.02,
		PromoteIntervalS: 0.05,
		ReapIntervalS:    10.0,
		Drain:            true,
		Handler: func(job omniq.JobCtx) {
			isLastAttempt := job.Attempt >= job.MaxAttempts
			seen = append(seen, seenItem{
				Attempt:       job.Attempt,
				MaxAttempts:   job.MaxAttempts,
				IsLastAttempt: isLastAttempt,
			})

			if !isLastAttempt {
				panic("Intentional failure before the last attempt")
			}

			if !sigSent {
				sigSent = true
				go func() {
					time.Sleep(50 * time.Millisecond)
					_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
				}()
			}
			time.Sleep(100 * time.Millisecond)
		},
	})
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:        "go",
		Queue:      queue,
		JobID:      jobID,
		Seen:       seen,
		FinalState: hgetString(inspect, ctx, "{"+queue+"}:job:"+jobID, "state"),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(b))
}

func hgetString(r redis.UniversalClient, ctx context.Context, key, field string) string {
	v, err := r.HGet(ctx, key, field).Result()
	if err != nil {
		return ""
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
