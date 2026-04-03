package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK             string              `json:"sdk"`
	Queue           string              `json:"queue"`
	WaitPage        []omniq.LaneJob     `json:"wait_page"`
	WaitPageReverse []omniq.LaneJob     `json:"wait_page_reverse"`
	FindWait        []omniq.LaneJob     `json:"find_wait"`
	GetExisting     *omniq.JobInfo      `json:"get_existing"`
	GetMissing      *omniq.JobInfo      `json:"get_missing"`
	Overview        omniq.QueueOverview `json:"overview"`
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
	queue := getenv("QUEUE", "validation-s14-go")
	baseNowMs := int64(1775310000000)

	waitKeep := queue + "-wait-keep-001"
	waitMissing := queue + "-wait-missing-001"
	activeJob := queue + "-active-001"
	delayedJob := queue + "-delayed-001"
	completedJob := queue + "-completed-001"
	failedJob := queue + "-failed-001"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: completedJob, Payload: map[string]any{"kind": "monitor-lanes", "slot": "completed"}, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: activeJob, Payload: map[string]any{"kind": "monitor-lanes", "slot": "active"}, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: failedJob, Payload: map[string]any{"kind": "monitor-lanes", "slot": "failed"}, MaxAttempts: 1, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: delayedJob, Payload: map[string]any{"kind": "monitor-lanes", "slot": "delayed"}, DueMs: baseNowMs + 100000, NowMsOverride: baseNowMs + 4})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: waitKeep, Payload: map[string]any{"kind": "monitor-lanes", "slot": "wait-keep"}, NowMsOverride: baseNowMs + 5})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: waitMissing, Payload: map[string]any{"kind": "monitor-lanes", "slot": "wait-missing"}, NowMsOverride: baseNowMs + 6})

	completedRes := reserveJob(client, queue, baseNowMs+100)
	activeRes := reserveJob(client, queue, baseNowMs+101)
	failedRes := reserveJob(client, queue, baseNowMs+102)

	if err := client.AckSuccess(queue, completedRes.JobID, completedRes.LeaseToken, baseNowMs+150); err != nil {
		fail(err)
	}
	failMsg := "terminal failure"
	if _, err := client.AckFail(queue, failedRes.JobID, failedRes.LeaseToken, &failMsg, baseNowMs+151); err != nil {
		fail(err)
	}

	_, err = client.Ops().R.Eval("return redis.call('DEL', KEYS[1])", 1, "{"+queue+"}:job:"+waitMissing)
	if err != nil {
		fail(err)
	}
	_ = activeRes

	waitPage := monitor.LanePage(queue, omniq.LaneWait, 0, 10, false)
	waitPageReverse := monitor.LanePage(queue, omniq.LaneWait, 0, 10, true)
	findWait := monitor.FindJobs(queue, omniq.LaneWait, []string{waitKeep, waitMissing})
	getExisting := monitor.GetJob(queue, activeJob)
	getMissing := monitor.GetJob(queue, waitMissing)
	overview := monitor.Overview(queue, 10)

	out := resultView{
		SDK:             "go",
		Queue:           queue,
		WaitPage:        waitPage,
		WaitPageReverse: waitPageReverse,
		FindWait:        findWait,
		GetExisting:     getExisting,
		GetMissing:      getMissing,
		Overview:        overview,
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
