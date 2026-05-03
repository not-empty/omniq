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
	SingleRetryState  string              `json:"single_retry_state"`
	SingleRetryAttempt int                `json:"single_retry_attempt"`
	BatchRetryResults []omniq.BatchResult `json:"batch_retry_results"`
	RemoveActiveError string              `json:"remove_active_error"`
	BatchRemoveResults []omniq.BatchResult `json:"batch_remove_results"`
	DelayedRemoveResult string            `json:"delayed_remove_result"`
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
	queue := getenv("QUEUE", "validation-s10-go")
	baseNowMs := int64(1775280000000)

	activeJob := queue + "-active-job-001"
	singleRetryJob := queue + "-single-retry-job-001"
	batchRetryJob := queue + "-batch-retry-job-001"
	waitingRemoveJob := queue + "-waiting-remove-job-001"
	delayedRemoveJob := queue + "-delayed-remove-job-001"

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: getenv("REDIS_HOST", "omniq-redis"),
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: activeJob, Payload: map[string]any{"kind": "admin", "slot": "active"}, MaxAttempts: 3, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: singleRetryJob, Payload: map[string]any{"kind": "admin", "slot": "single-retry"}, MaxAttempts: 1, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: batchRetryJob, Payload: map[string]any{"kind": "admin", "slot": "batch-retry"}, MaxAttempts: 1, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: waitingRemoveJob, Payload: map[string]any{"kind": "admin", "slot": "waiting-remove"}, MaxAttempts: 3, NowMsOverride: baseNowMs + 4})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: delayedRemoveJob, Payload: map[string]any{"kind": "admin", "slot": "delayed-remove"}, MaxAttempts: 3, DueMs: baseNowMs + 100000, NowMsOverride: baseNowMs + 5})

	activeRes := reserveJob(client, queue, baseNowMs+100)
	singleFailedRes := reserveJob(client, queue, baseNowMs+101)
	batchFailedRes := reserveJob(client, queue, baseNowMs+102)

	singleErr := "single retry setup"
	_, err = client.AckFail(queue, singleFailedRes.JobID, singleFailedRes.LeaseToken, &singleErr, baseNowMs+150)
	if err != nil {
		fail(err)
	}
	batchErr := "batch retry setup"
	_, err = client.AckFail(queue, batchFailedRes.JobID, batchFailedRes.LeaseToken, &batchErr, baseNowMs+151)
	if err != nil {
		fail(err)
	}

	err = client.Ops().RetryFailed(queue, singleRetryJob, baseNowMs+200)
	if err != nil {
		fail(err)
	}

	batchRetryResults, err := client.Ops().RetryFailedBatch(queue, []string{batchRetryJob, waitingRemoveJob}, baseNowMs+201)
	if err != nil {
		fail(err)
	}

	removeActiveError := "NO_ERROR"
	_, err = client.RemoveJob(queue, activeJob, "wait")
	if err != nil {
		removeActiveError = err.Error()
	}

	batchRemoveResults, err := client.RemoveJobsBatch(queue, "wait", []string{waitingRemoveJob, delayedRemoveJob})
	if err != nil {
		fail(err)
	}

	delayedRemoveResult, err := client.RemoveJob(queue, delayedRemoveJob, "delayed")
	if err != nil {
		fail(err)
	}

	stateRaw, err := client.Ops().R.HGet("{"+queue+"}:job:"+singleRetryJob, "state")
	if err != nil {
		fail(err)
	}
	attemptRaw, err := client.Ops().R.HGet("{"+queue+"}:job:"+singleRetryJob, "attempt")
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                "go",
		Queue:              queue,
		SingleRetryState:   deref(stateRaw),
		SingleRetryAttempt: atoiDefault(deref(attemptRaw)),
		BatchRetryResults:  batchRetryResults,
		RemoveActiveError:  removeActiveError,
		BatchRemoveResults: batchRemoveResults,
		DelayedRemoveResult: delayedRemoveResult,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	_ = activeRes
	fmt.Println(string(b))
}

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) {
	_, err := client.Publish(opts)
	if err != nil {
		fail(err)
	}
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func atoiDefault(v string) int {
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil {
		return 0
	}
	return n
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
