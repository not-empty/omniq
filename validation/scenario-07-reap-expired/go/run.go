package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK            string `json:"sdk"`
	Queue          string `json:"queue"`
	ReapedCount    int    `json:"reaped_count"`
	RetryableJobID string `json:"retryable_job_id"`
	TerminalJobID  string `json:"terminal_job_id"`
}

func main() {
	queue := getenv("QUEUE", "validation-s07-go")
	retryJobID := getenv("RETRY_JOB_ID", queue+"-retry-job-001")
	failJobID := getenv("FAIL_JOB_ID", queue+"-fail-job-001")
	baseNowMs := int64(1775260000000)
	reapNowMs := baseNowMs + 31000

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: "omniq-redis",
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: retryJobID,
		Payload: map[string]any{
			"kind": "reap-expired",
			"mode": "retry",
			"sdk":  "go",
		},
		Timeout:       30000,
		MaxAttempts:   3,
		Backoff:       5000,
		NowMsOverride: baseNowMs,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: failJobID,
		Payload: map[string]any{
			"kind": "reap-expired",
			"mode": "terminal",
			"sdk":  "go",
		},
		Timeout:       30000,
		MaxAttempts:   1,
		Backoff:       5000,
		NowMsOverride: baseNowMs,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Reserve(queue, baseNowMs)
	if err != nil {
		fail(err)
	}
	_, err = client.Reserve(queue, baseNowMs)
	if err != nil {
		fail(err)
	}

	reaped, err := client.ReapExpired(queue, 1000, reapNowMs)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:            "go",
		Queue:          queue,
		ReapedCount:    reaped,
		RetryableJobID: retryJobID,
		TerminalJobID:  failJobID,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}

	fmt.Println(string(b))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
