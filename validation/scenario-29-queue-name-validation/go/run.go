package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type invalidResult struct {
	Queue           string `json:"queue"`
	PublishRejected bool   `json:"publish_rejected"`
	StatsRejected   bool   `json:"stats_rejected"`
}

type resultView struct {
	SDK            string          `json:"sdk"`
	Queue          string          `json:"queue"`
	ValidJobID     string          `json:"valid_job_id"`
	InvalidResults []invalidResult `json:"invalid_results"`
}

func main() {
	queue := getenv("QUEUE", "validation-s29-go")
	invalidNames := []string{"", " bad", "bad ", "bad:name", "{manual-tag}", "bad/name", "bad\\name", "bad name"}

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	validJobID, err := client.Publish(omniq.PublishOpts{
		Queue:   queue,
		JobID:   queue + "-job-001",
		Payload: map[string]any{"kind": "queue-name-validation", "sdk": "go"},
	})
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:        "go",
		Queue:      queue,
		ValidJobID: validJobID,
	}

	for _, name := range invalidNames {
		publishRejected := false
		statsRejected := false

		_, err = client.Publish(omniq.PublishOpts{
			Queue:   name,
			Payload: map[string]any{"kind": "invalid"},
		})
		if err != nil {
			publishRejected = true
		}

		func() {
			defer func() {
				if recover() != nil {
					statsRejected = true
				}
			}()
			_ = monitor.Stats(name)
		}()

		out.InvalidResults = append(out.InvalidResults, invalidResult{
			Queue:           name,
			PublishRejected: publishRejected,
			StatsRejected:   statsRejected,
		})
	}

	if out.ValidJobID == "" {
		fail(fmt.Errorf("valid queue did not publish a job id"))
	}
	for _, row := range out.InvalidResults {
		if !row.PublishRejected || !row.StatsRejected {
			fail(fmt.Errorf("invalid queue name was not rejected consistently: %q", row.Queue))
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
