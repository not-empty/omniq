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
	ReapedCount            int      `json:"reaped_count"`
	AlphaInflightAfterReap int64    `json:"alpha_inflight_after_reap"`
	BetaInflightAfterReap  int64    `json:"beta_inflight_after_reap"`
	AlphaReadyAfterReap    bool     `json:"alpha_ready_after_reap"`
	BetaReadyAfterReap     bool     `json:"beta_ready_after_reap"`
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
	queue := getenv("QUEUE", "validation-s18-go")
	baseNowMs := int64(1775350000000)
	reapNowMs := baseNowMs + 31_000

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-001", Payload: map[string]any{"kind": "grouped-reap-expired", "slot": "alpha-1"}, GID: "alpha", GroupLimit: 1, MaxAttempts: 3, Timeout: 30000, Backoff: 5000, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-002", Payload: map[string]any{"kind": "grouped-reap-expired", "slot": "alpha-2"}, GID: "alpha", GroupLimit: 1, MaxAttempts: 3, Timeout: 30000, Backoff: 5000, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-001", Payload: map[string]any{"kind": "grouped-reap-expired", "slot": "beta-1"}, GID: "beta", GroupLimit: 1, MaxAttempts: 1, Timeout: 30000, Backoff: 5000, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-002", Payload: map[string]any{"kind": "grouped-reap-expired", "slot": "beta-2"}, GID: "beta", GroupLimit: 1, MaxAttempts: 1, Timeout: 30000, Backoff: 5000, NowMsOverride: baseNowMs + 4})

	reserveJob(client, queue, baseNowMs+100)
	reserveJob(client, queue, baseNowMs+101)

	reapedCount, err := client.ReapExpired(queue, 1000, reapNowMs)
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

	nextOne := reserveJob(client, queue, reapNowMs+1)
	nextTwo := reserveJob(client, queue, reapNowMs+2)

	out := resultView{
		SDK:                    "go",
		Queue:                  queue,
		ReapedCount:            reapedCount,
		AlphaInflightAfterReap: alphaInflight,
		BetaInflightAfterReap:  betaInflight,
		AlphaReadyAfterReap:    alphaScore != nil,
		BetaReadyAfterReap:     betaScore != nil,
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
