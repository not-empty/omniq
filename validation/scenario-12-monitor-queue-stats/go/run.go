package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK        string             `json:"sdk"`
	QueuesFound []string          `json:"queues_found"`
	StatsA     omniq.QueueStats   `json:"stats_a"`
	StatsB     omniq.QueueStats   `json:"stats_b"`
	StatsMany  []omniq.QueueStats `json:"stats_many"`
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
	prefix := getenv("PREFIX", "validation-s12-go")
	queueA := prefix + "-paused"
	queueB := prefix + "-mixed"
	baseNowMs := int64(1775290000000)

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queueA, JobID: queueA + "-job-001", Payload: map[string]any{"kind": "monitor", "queue": "a"}, NowMsOverride: baseNowMs + 1})
	if _, err := client.Pause(queueA); err != nil {
		fail(err)
	}

	completedJob := queueB + "-completed-job-001"
	activeJob := queueB + "-active-job-001"
	delayedJob := queueB + "-delayed-job-001"

	mustPublish(client, omniq.PublishOpts{Queue: queueB, JobID: completedJob, Payload: map[string]any{"kind": "monitor", "slot": "completed"}, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queueB, JobID: activeJob, Payload: map[string]any{"kind": "monitor", "slot": "active"}, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queueB, JobID: delayedJob, Payload: map[string]any{"kind": "monitor", "slot": "delayed"}, DueMs: baseNowMs + 100000, NowMsOverride: baseNowMs + 4})

	completedRes := reserveJob(client, queueB, baseNowMs+100)
	activeRes := reserveJob(client, queueB, baseNowMs+101)
	if err := client.AckSuccess(queueB, completedRes.JobID, completedRes.LeaseToken, baseNowMs+150); err != nil {
		fail(err)
	}
	_ = activeRes

	listQueues := monitor.ListQueues()
	queuesFound := make([]string, 0, 2)
	for _, q := range listQueues {
		if q == queueA || q == queueB {
			queuesFound = append(queuesFound, q)
		}
	}
	slices.Sort(queuesFound)

	statsA := monitor.Stats(queueA)
	statsB := monitor.Stats(queueB)
	statsMany := monitor.StatsMany([]string{queueA, queueB})

	out := resultView{
		SDK:         "go",
		QueuesFound: queuesFound,
		StatsA:      statsA,
		StatsB:      statsB,
		StatsMany:   statsMany,
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
