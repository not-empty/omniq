package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                    string   `json:"sdk"`
	Queue                  string   `json:"queue"`
	PromotedCount          int      `json:"promoted_count"`
	AlphaReadyAfterPromote bool     `json:"alpha_ready_after_promote"`
	BetaReadyAfterPromote  bool     `json:"beta_ready_after_promote"`
	GroupWaitingAfterPromote int64  `json:"group_waiting_after_promote"`
	NextJobIDs             []string `json:"next_job_ids"`
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
	queue := getenv("QUEUE", "validation-s19-go")
	baseNowMs := int64(1775360000000)
	dueMs := baseNowMs + 5000

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-001", Payload: map[string]any{"kind": "grouped-promote-delayed", "slot": "alpha-1"}, GID: "alpha", GroupLimit: 1, DueMs: dueMs, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-001", Payload: map[string]any{"kind": "grouped-promote-delayed", "slot": "beta-1"}, GID: "beta", GroupLimit: 1, DueMs: dueMs, NowMsOverride: baseNowMs + 2})

	promotedCount, err := client.PromoteDelayed(queue, 1000, dueMs)
	if err != nil {
		fail(err)
	}

	alphaScore, err := client.Ops().R.ZScore("{"+queue+"}:groups:ready", "alpha")
	if err != nil {
		fail(err)
	}
	betaScore, err := client.Ops().R.ZScore("{"+queue+"}:groups:ready", "beta")
	if err != nil {
		fail(err)
	}
	statsRaw, err := client.Ops().R.HGetAll("{" + queue + "}:stats")
	if err != nil {
		fail(err)
	}
	groupWaitingAfterPromote := int64(0)
	fmt.Sscanf(statsRaw["group_waiting"], "%d", &groupWaitingAfterPromote)

	nextOne := reserveJob(client, queue, dueMs+1)
	nextTwo := reserveJob(client, queue, dueMs+2)

	out := resultView{
		SDK:                    "go",
		Queue:                  queue,
		PromotedCount:          promotedCount,
		AlphaReadyAfterPromote: alphaScore != nil,
		BetaReadyAfterPromote:  betaScore != nil,
		GroupWaitingAfterPromote: groupWaitingAfterPromote,
		NextJobIDs:             []string{nextOne.JobID, nextTwo.JobID},
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
