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
	JobID          string `json:"job_id"`
	ScheduledDueMS int64  `json:"scheduled_due_ms"`
	PromotedCount  int    `json:"promoted_count"`
}

func main() {
	queue := getenv("QUEUE", "validation-s06-go")
	jobID := getenv("JOB_ID", queue+"-job-001")
	baseNowMs := int64(1775250000000)
	dueMs := baseNowMs + 5000

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: "omniq-redis",
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: jobID,
		Payload: map[string]any{
			"kind":   "promote-delayed",
			"source": "validation",
			"sdk":    "go",
			"value":  6,
		},
		Timeout:       30000,
		MaxAttempts:   3,
		Backoff:       5000,
		DueMs:         dueMs,
		NowMsOverride: baseNowMs,
	})
	if err != nil {
		fail(err)
	}

	promoted, err := client.PromoteDelayed(queue, 1000, dueMs)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:            "go",
		Queue:          queue,
		JobID:          jobID,
		ScheduledDueMS: dueMs,
		PromotedCount:  promoted,
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
