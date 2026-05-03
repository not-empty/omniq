package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                   string `json:"sdk"`
	Queue                 string `json:"queue"`
	FirstJobID            string `json:"first_job_id"`
	SecondJobID           string `json:"second_job_id"`
	GroupReadyAfterAck    bool   `json:"group_ready_after_ack"`
	GroupInflightAfterAck int64  `json:"group_inflight_after_ack"`
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
	queue := getenv("QUEUE", "validation-s16-go")
	baseNowMs := int64(1775330000000)
	gid := "alpha"
	firstJob := queue + "-alpha-job-001"
	secondJob := queue + "-alpha-job-002"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{
		Queue:         queue,
		JobID:         firstJob,
		Payload:       map[string]any{"kind": "grouped-ack-success", "slot": "first"},
		GID:           gid,
		GroupLimit:    1,
		NowMsOverride: baseNowMs + 1,
	})
	mustPublish(client, omniq.PublishOpts{
		Queue:         queue,
		JobID:         secondJob,
		Payload:       map[string]any{"kind": "grouped-ack-success", "slot": "second"},
		GID:           gid,
		GroupLimit:    1,
		NowMsOverride: baseNowMs + 2,
	})

	first := reserveJob(client, queue, baseNowMs+100)
	if err := client.AckSuccess(queue, first.JobID, first.LeaseToken, baseNowMs+150); err != nil {
		fail(err)
	}

	score, err := client.Ops().R.ZScore("{"+queue+"}:groups:ready", gid)
	if err != nil {
		fail(err)
	}
	groupReadyAfterAck := score != nil

	inflightRaw, err := client.Ops().R.Get("{" + queue + "}:g:" + gid + ":inflight")
	if err != nil {
		fail(err)
	}
	groupInflightAfterAck := int64(0)
	if inflightRaw != nil {
		fmt.Sscanf(*inflightRaw, "%d", &groupInflightAfterAck)
	}

	second := reserveJob(client, queue, baseNowMs+151)

	out := resultView{
		SDK:                   "go",
		Queue:                 queue,
		FirstJobID:            first.JobID,
		SecondJobID:           second.JobID,
		GroupReadyAfterAck:    groupReadyAfterAck,
		GroupInflightAfterAck: groupInflightAfterAck,
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
