package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK               string              `json:"sdk"`
	Queue             string              `json:"queue"`
	BatchRetryResults []omniq.BatchResult `json:"batch_retry_results"`
	RetriedJobState   string              `json:"retried_job_state"`
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

func main() {
	queue := getenv("QUEUE", "validation-s20-go")
	baseNowMs := int64(1775370000000)

	failedJob := queue + "-failed-job-001"
	waitingJob := queue + "-waiting-job-001"
	missingJob := queue + "-missing-job-001"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: failedJob, Payload: map[string]any{"kind": "batch-retry-errors", "slot": "failed"}, MaxAttempts: 1, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: waitingJob, Payload: map[string]any{"kind": "batch-retry-errors", "slot": "waiting"}, MaxAttempts: 3, NowMsOverride: baseNowMs + 2})

	failedRes := reserveJob(client, queue, baseNowMs+100)
	msg := "make failed"
	if _, err := client.AckFail(queue, failedRes.JobID, failedRes.LeaseToken, &msg, baseNowMs+150); err != nil {
		fail(err)
	}

	batchRetryResults, err := client.Ops().RetryFailedBatch(queue, []string{failedJob, missingJob, waitingJob}, baseNowMs+200)
	if err != nil {
		fail(err)
	}

	retriedJobStateRaw, err := client.Ops().R.HGet("{"+queue+"}:job:"+failedJob, "state")
	if err != nil {
		fail(err)
	}
	retriedJobState := ""
	if retriedJobStateRaw != nil {
		retriedJobState = *retriedJobStateRaw
	}

	out := resultView{
		SDK:               "go",
		Queue:             queue,
		BatchRetryResults: batchRetryResults,
		RetriedJobState:   retriedJobState,
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
