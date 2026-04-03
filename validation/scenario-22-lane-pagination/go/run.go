package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                 string           `json:"sdk"`
	Queue               string           `json:"queue"`
	Stats               omniq.QueueStats `json:"stats"`
	WaitForwardPages    [][]string       `json:"wait_forward_pages"`
	WaitReversePages    [][]string       `json:"wait_reverse_pages"`
	DelayedForwardPages [][]string       `json:"delayed_forward_pages"`
	DelayedReversePages [][]string       `json:"delayed_reverse_pages"`
	IdxWaitRaw          []string         `json:"idx_wait_raw"`
	IdxDelayedRaw       []string         `json:"idx_delayed_raw"`
}

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) {
	_, err := client.Publish(opts)
	if err != nil {
		fail(err)
	}
}

func jobIDs(rows []omniq.LaneJob) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.JobID)
	}
	return out
}

func main() {
	queue := getenv("QUEUE", "validation-s22-go")
	baseNowMs := int64(1775390000000)

	waitJobs := []string{
		queue + "-wait-001",
		queue + "-wait-002",
		queue + "-wait-003",
		queue + "-wait-004",
		queue + "-wait-005",
	}
	delayedJobs := []string{
		queue + "-delayed-001",
		queue + "-delayed-002",
		queue + "-delayed-003",
		queue + "-delayed-004",
		queue + "-delayed-005",
	}

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	for i, jobID := range waitJobs {
		mustPublish(client, omniq.PublishOpts{
			Queue:         queue,
			JobID:         jobID,
			Payload:       map[string]any{"kind": "lane-pagination", "lane": "wait", "order": i + 1},
			NowMsOverride: baseNowMs + int64(i) + 1,
		})
	}

	for i, jobID := range delayedJobs {
		mustPublish(client, omniq.PublishOpts{
			Queue:         queue,
			JobID:         jobID,
			Payload:       map[string]any{"kind": "lane-pagination", "lane": "delayed", "order": i + 1},
			DueMs:         baseNowMs + 100000 + int64(i) + 1,
			NowMsOverride: baseNowMs + 100 + int64(i) + 1,
		})
	}

	waitForwardPages := [][]string{
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 0, 2, false)),
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 2, 2, false)),
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 4, 2, false)),
	}
	waitReversePages := [][]string{
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 0, 2, true)),
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 2, 2, true)),
		jobIDs(monitor.LanePage(queue, omniq.LaneWait, 4, 2, true)),
	}
	delayedForwardPages := [][]string{
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 0, 2, false)),
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 2, 2, false)),
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 4, 2, false)),
	}
	delayedReversePages := [][]string{
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 0, 2, true)),
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 2, 2, true)),
		jobIDs(monitor.LanePage(queue, omniq.LaneDelayed, 4, 2, true)),
	}

	idxWaitRaw, err := client.Ops().R.ZRange("{"+queue+"}:idx:wait", 0, -1)
	if err != nil {
		fail(err)
	}
	idxDelayedRaw, err := client.Ops().R.ZRange("{"+queue+"}:idx:delayed", 0, -1)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                 "go",
		Queue:               queue,
		Stats:               monitor.Stats(queue),
		WaitForwardPages:    waitForwardPages,
		WaitReversePages:    waitReversePages,
		DelayedForwardPages: delayedForwardPages,
		DelayedReversePages: delayedReversePages,
		IdxWaitRaw:          idxWaitRaw,
		IdxDelayedRaw:       idxDelayedRaw,
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
