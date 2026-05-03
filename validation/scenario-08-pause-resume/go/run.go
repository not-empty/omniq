package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                string `json:"sdk"`
	Queue              string `json:"queue"`
	PausedBefore       bool   `json:"paused_before"`
	PausedAfterPause   bool   `json:"paused_after_pause"`
	PausedAfterResume  bool   `json:"paused_after_resume"`
	PausedReserveStatus string `json:"paused_reserve_status"`
	FirstReservedJobID string `json:"first_reserved_job_id"`
	SecondReservedJobID string `json:"second_reserved_job_id"`
}

func main() {
	queue := getenv("QUEUE", "validation-s08-go")
	firstJob := queue + "-job-001"
	secondJob := queue + "-job-002"

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: getenv("REDIS_HOST", "omniq-redis"),
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue:   queue,
		JobID:   firstJob,
		Payload: map[string]any{"kind": "pause-resume", "n": 1},
	})
	if err != nil {
		fail(err)
	}
	_, err = client.Publish(omniq.PublishOpts{
		Queue:   queue,
		JobID:   secondJob,
		Payload: map[string]any{"kind": "pause-resume", "n": 2},
	})
	if err != nil {
		fail(err)
	}

	pausedBefore, err := client.IsPaused(queue)
	if err != nil {
		fail(err)
	}

	firstRes, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}
	first, ok := firstRes.(omniq.ReserveJob)
	if !ok {
		fail(fmt.Errorf("unexpected first reserve: %#v", firstRes))
	}

	_, err = client.Pause(queue)
	if err != nil {
		fail(err)
	}
	pausedAfterPause, err := client.IsPaused(queue)
	if err != nil {
		fail(err)
	}

	pausedRes, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}
	pausedStatus := ""
	if paused, ok := pausedRes.(omniq.ReservePaused); ok {
		pausedStatus = paused.Status
	}

	_, err = client.Resume(queue)
	if err != nil {
		fail(err)
	}
	pausedAfterResume, err := client.IsPaused(queue)
	if err != nil {
		fail(err)
	}

	secondRes, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}
	second, ok := secondRes.(omniq.ReserveJob)
	if !ok {
		fail(fmt.Errorf("unexpected second reserve: %#v", secondRes))
	}

	out := resultView{
		SDK:                 "go",
		Queue:               queue,
		PausedBefore:        pausedBefore,
		PausedAfterPause:    pausedAfterPause,
		PausedAfterResume:   pausedAfterResume,
		PausedReserveStatus: pausedStatus,
		FirstReservedJobID:  first.JobID,
		SecondReservedJobID: second.JobID,
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
