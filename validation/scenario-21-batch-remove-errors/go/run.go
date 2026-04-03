package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type statsView struct {
	Waiting      int `json:"waiting"`
	GroupWaiting int `json:"group_waiting"`
	WaitingTotal int `json:"waiting_total"`
	Active       int `json:"active"`
	Delayed      int `json:"delayed"`
	GroupsReady  int `json:"groups_ready"`
}

type existsView struct {
	WaitJob        int64 `json:"wait_job"`
	GroupedWaitJob int64 `json:"grouped_wait_job"`
	ActiveJob      int64 `json:"active_job"`
	DelayedJob     int64 `json:"delayed_job"`
}

type resultView struct {
	SDK               string              `json:"sdk"`
	Queue             string              `json:"queue"`
	BatchRemoveResults []omniq.BatchResult `json:"batch_remove_results"`
	JobHashExists     existsView          `json:"job_hash_exists"`
	Stats             statsView           `json:"stats"`
	WaitLen           int64               `json:"wait_len"`
	IdxWait           int64               `json:"idx_wait"`
	GroupWaitLen      int64               `json:"group_wait_len"`
	GroupsReady       int64               `json:"groups_ready"`
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
	queue := getenv("QUEUE", "validation-s21-go")
	baseNowMs := int64(1775380000000)

	waitJob := queue + "-wait-job-001"
	groupedWaitJob := queue + "-grouped-wait-job-001"
	activeJob := queue + "-active-job-001"
	delayedJob := queue + "-delayed-job-001"
	missingJob := queue + "-missing-job-001"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: activeJob, Payload: map[string]any{"kind": "batch-remove-errors", "slot": "active"}, MaxAttempts: 3, NowMsOverride: baseNowMs + 1})

	activeRes := reserveJob(client, queue, baseNowMs+100)
	if activeRes.JobID != activeJob {
		fail(fmt.Errorf("expected active job %s, got %s", activeJob, activeRes.JobID))
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: waitJob, Payload: map[string]any{"kind": "batch-remove-errors", "slot": "wait"}, MaxAttempts: 3, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: groupedWaitJob, Payload: map[string]any{"kind": "batch-remove-errors", "slot": "gwait"}, MaxAttempts: 3, GID: "alpha", GroupLimit: 1, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: delayedJob, Payload: map[string]any{"kind": "batch-remove-errors", "slot": "delayed"}, MaxAttempts: 3, DueMs: baseNowMs + 100000, NowMsOverride: baseNowMs + 4})

	batchRemoveResults, err := client.RemoveJobsBatch(queue, "wait", []string{waitJob, missingJob, groupedWaitJob, activeJob, delayedJob})
	if err != nil {
		fail(err)
	}

	statsKey := "{" + queue + "}:stats"
	waitingRaw, err := client.Ops().R.HGet(statsKey, "waiting")
	if err != nil {
		fail(err)
	}
	groupWaitingRaw, err := client.Ops().R.HGet(statsKey, "group_waiting")
	if err != nil {
		fail(err)
	}
	waitingTotalRaw, err := client.Ops().R.HGet(statsKey, "waiting_total")
	if err != nil {
		fail(err)
	}
	activeRaw, err := client.Ops().R.HGet(statsKey, "active")
	if err != nil {
		fail(err)
	}
	delayedRaw, err := client.Ops().R.HGet(statsKey, "delayed")
	if err != nil {
		fail(err)
	}
	groupsReadyRaw, err := client.Ops().R.HGet(statsKey, "groups_ready")
	if err != nil {
		fail(err)
	}

	waitExists, err := client.Ops().R.Exists("{" + queue + "}:job:" + waitJob)
	if err != nil {
		fail(err)
	}
	groupedWaitExists, err := client.Ops().R.Exists("{" + queue + "}:job:" + groupedWaitJob)
	if err != nil {
		fail(err)
	}
	activeExists, err := client.Ops().R.Exists("{" + queue + "}:job:" + activeJob)
	if err != nil {
		fail(err)
	}
	delayedExists, err := client.Ops().R.Exists("{" + queue + "}:job:" + delayedJob)
	if err != nil {
		fail(err)
	}

	waitLen, err := client.Ops().R.LLen("{" + queue + "}:wait")
	if err != nil {
		fail(err)
	}
	idxWait, err := client.Ops().R.ZCard("{" + queue + "}:idx:wait")
	if err != nil {
		fail(err)
	}
	groupWaitLen, err := client.Ops().R.LLen("{" + queue + "}:g:alpha:wait")
	if err != nil {
		fail(err)
	}
	groupsReady, err := client.Ops().R.ZCard("{" + queue + "}:groups:ready")
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                "go",
		Queue:              queue,
		BatchRemoveResults: batchRemoveResults,
		JobHashExists: existsView{
			WaitJob:        waitExists,
			GroupedWaitJob: groupedWaitExists,
			ActiveJob:      activeExists,
			DelayedJob:     delayedExists,
		},
		Stats: statsView{
			Waiting:      atoiDefault(deref(waitingRaw)),
			GroupWaiting: atoiDefault(deref(groupWaitingRaw)),
			WaitingTotal: atoiDefault(deref(waitingTotalRaw)),
			Active:       atoiDefault(deref(activeRaw)),
			Delayed:      atoiDefault(deref(delayedRaw)),
			GroupsReady:  atoiDefault(deref(groupsReadyRaw)),
		},
		WaitLen:      waitLen,
		IdxWait:      idxWait,
		GroupWaitLen: groupWaitLen,
		GroupsReady:  groupsReady,
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
