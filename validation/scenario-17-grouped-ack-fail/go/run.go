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
	AlphaFailStatus        string   `json:"alpha_fail_status"`
	BetaFailStatus         string   `json:"beta_fail_status"`
	AlphaInflightAfterFail int64    `json:"alpha_inflight_after_fail"`
	BetaInflightAfterFail  int64    `json:"beta_inflight_after_fail"`
	AlphaReadyAfterFail    bool     `json:"alpha_ready_after_fail"`
	BetaReadyAfterFail     bool     `json:"beta_ready_after_fail"`
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
	queue := getenv("QUEUE", "validation-s17-go")
	baseNowMs := int64(1775340000000)

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-001", Payload: map[string]any{"kind": "grouped-ack-fail", "slot": "alpha-1"}, GID: "alpha", GroupLimit: 1, MaxAttempts: 3, Backoff: 5000, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-002", Payload: map[string]any{"kind": "grouped-ack-fail", "slot": "alpha-2"}, GID: "alpha", GroupLimit: 1, MaxAttempts: 3, Backoff: 5000, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-001", Payload: map[string]any{"kind": "grouped-ack-fail", "slot": "beta-1"}, GID: "beta", GroupLimit: 1, MaxAttempts: 1, Backoff: 5000, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-002", Payload: map[string]any{"kind": "grouped-ack-fail", "slot": "beta-2"}, GID: "beta", GroupLimit: 1, MaxAttempts: 1, Backoff: 5000, NowMsOverride: baseNowMs + 4})

	alphaFirst := reserveJob(client, queue, baseNowMs+100)
	betaFirst := reserveJob(client, queue, baseNowMs+101)

	alphaMsg := "retryable grouped fail"
	alphaFail, err := client.AckFail(queue, alphaFirst.JobID, alphaFirst.LeaseToken, &alphaMsg, baseNowMs+150)
	if err != nil {
		fail(err)
	}
	betaMsg := "terminal grouped fail"
	betaFail, err := client.AckFail(queue, betaFirst.JobID, betaFirst.LeaseToken, &betaMsg, baseNowMs+151)
	if err != nil {
		fail(err)
	}

	alphaInflightRaw, err := client.Ops().R.Get("{" + queue + "}:g:alpha:inflight")
	if err != nil {
		fail(err)
	}
	betaInflightRaw, err := client.Ops().R.Get("{" + queue + "}:g:beta:inflight")
	if err != nil {
		fail(err)
	}
	alphaInflight := int64(0)
	betaInflight := int64(0)
	if alphaInflightRaw != nil {
		fmt.Sscanf(*alphaInflightRaw, "%d", &alphaInflight)
	}
	if betaInflightRaw != nil {
		fmt.Sscanf(*betaInflightRaw, "%d", &betaInflight)
	}

	alphaScore, err := client.Ops().R.ZScore("{"+queue+"}:groups:ready", "alpha")
	if err != nil {
		fail(err)
	}
	betaScore, err := client.Ops().R.ZScore("{"+queue+"}:groups:ready", "beta")
	if err != nil {
		fail(err)
	}

	nextOne := reserveJob(client, queue, baseNowMs+152)
	nextTwo := reserveJob(client, queue, baseNowMs+153)

	out := resultView{
		SDK:                    "go",
		Queue:                  queue,
		AlphaFailStatus:        string(alphaFail.Status),
		BetaFailStatus:         string(betaFail.Status),
		AlphaInflightAfterFail: alphaInflight,
		BetaInflightAfterFail:  betaInflight,
		AlphaReadyAfterFail:    alphaScore != nil,
		BetaReadyAfterFail:     betaScore != nil,
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
