package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type reserveView struct {
	Status            string `json:"status"`
	JobID             string `json:"job_id"`
	Payload           string `json:"payload"`
	Attempt           int    `json:"attempt"`
	MaxAttempts       int    `json:"max_attempts"`
	GID               string `json:"gid"`
	LeaseTokenPresent bool   `json:"lease_token_present"`
}

type resultView struct {
	SDK                    string       `json:"sdk"`
	Queue                  string       `json:"queue"`
	InvalidPublishRejected bool         `json:"invalid_publish_rejected"`
	JobID                  string       `json:"job_id"`
	Reserve                *reserveView `json:"reserve"`
}

func main() {
	queue := getenv("QUEUE", "validation-basic-go")
	jobID := getenv("JOB_ID", queue+"-job-001")

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: "omniq-redis",
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	invalidPublishRejected := false
	if _, err := client.Publish(omniq.PublishOpts{
		Queue:   queue,
		Payload: "raw-string",
	}); err != nil {
		invalidPublishRejected = true
	}

	publishedJobID, err := client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: jobID,
		Payload: map[string]any{
			"kind":   "basic-reserve",
			"source": "validation",
			"sdk":    "go",
			"value":  1,
		},
		Timeout:     30_000,
		MaxAttempts: 3,
		Backoff:     5_000,
	})
	if err != nil {
		fail(err)
	}

	res, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                    "go",
		Queue:                  queue,
		InvalidPublishRejected: invalidPublishRejected,
		JobID:                  publishedJobID,
		Reserve:                nil,
	}

	if job, ok := res.(omniq.ReserveJob); ok {
		out.Reserve = &reserveView{
			Status:            job.Status,
			JobID:             job.JobID,
			Payload:           job.Payload,
			Attempt:           job.Attempt,
			MaxAttempts:       job.MaxAttempts,
			GID:               job.GID,
			LeaseTokenPresent: job.LeaseToken != "",
		}
	} else if paused, ok := res.(omniq.ReservePaused); ok {
		out.Reserve = &reserveView{
			Status: paused.Status,
		}
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
