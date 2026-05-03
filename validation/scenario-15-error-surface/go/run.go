package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK           string `json:"sdk"`
	Queue         string `json:"queue"`
	TokenMismatch string `json:"token_mismatch"`
	NotActive     string `json:"not_active"`
	BatchLimit    string `json:"batch_limit"`
	InvalidPublish string `json:"invalid_publish"`
	LaneMismatch  string `json:"lane_mismatch"`
}

func reserveJob(client *omniq.Client, queue string, nowMs int64) omniq.ReserveJob {
	res, err := client.Reserve(queue, nowMs)
	if err != nil {
		fail(err)
	}
	job, ok := res.(omniq.ReserveJob)
	if !ok {
		fail(fmt.Errorf("unexpected reserve response: %#v", res))
	}
	return job
}

func capture(fn func() error) string {
	if err := fn(); err != nil {
		return err.Error()
	}
	return "NO_ERROR"
}

func main() {
	queue := getenv("QUEUE", "validation-s15-go")
	baseNowMs := int64(1775320000000)

	jobID := queue + "-job-001"
	delayedJob := queue + "-delayed-001"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	invalidPublish := capture(func() error {
		_, err := client.Publish(omniq.PublishOpts{
			Queue:   queue,
			JobID:   queue + "-bad-publish",
			Payload: "raw-string",
		})
		return err
	})

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: jobID, Payload: map[string]any{"kind": "error-surface"}, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: delayedJob, Payload: map[string]any{"kind": "error-surface", "slot": "delayed"}, DueMs: baseNowMs + 100000, NowMsOverride: baseNowMs + 2})

	reserved := reserveJob(client, queue, baseNowMs+100)

	tokenMismatch := capture(func() error {
		return client.AckSuccess(queue, reserved.JobID, "bad-token", baseNowMs+110)
	})

	_, err = client.Ops().R.Eval("return redis.call('ZREM', KEYS[1], ARGV[1])", 1, "{"+queue+"}:active", reserved.JobID)
	if err != nil {
		fail(err)
	}

	notActive := capture(func() error {
		return client.AckSuccess(queue, reserved.JobID, reserved.LeaseToken, baseNowMs+112)
	})

	batchLimit := capture(func() error {
		ids := make([]string, 101)
		for i := 0; i < 101; i++ {
			ids[i] = fmt.Sprintf("%s-x-%03d", queue, i)
		}
		_, err := client.Ops().RetryFailedBatch(queue, ids, baseNowMs+120)
		return err
	})

	laneMismatch := capture(func() error {
		_, err := client.RemoveJob(queue, delayedJob, "wait")
		return err
	})

	out := resultView{
		SDK:            "go",
		Queue:          queue,
		TokenMismatch:  tokenMismatch,
		NotActive:      notActive,
		BatchLimit:     batchLimit,
		InvalidPublish: invalidPublish,
		LaneMismatch:   laneMismatch,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(b))
}

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) {
	_, err := client.Publish(opts)
	if err != nil {
		fail(err)
	}
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
