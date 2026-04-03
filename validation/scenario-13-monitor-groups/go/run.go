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
	GroupsReadyPage []string            `json:"groups_ready_page"`
	GroupsReadyAll  []omniq.GroupReady  `json:"groups_ready_all"`
	GroupStatus     []omniq.GroupStatus `json:"group_status"`
}

func main() {
	queue := getenv("QUEUE", "validation-s13-go")
	baseNowMs := int64(1775300000000)

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-001", Payload: map[string]any{"kind": "monitor-groups", "slot": "alpha-1"}, GID: "alpha", GroupLimit: 2, NowMsOverride: baseNowMs + 1})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-alpha-job-002", Payload: map[string]any{"kind": "monitor-groups", "slot": "alpha-2"}, GID: "alpha", GroupLimit: 2, NowMsOverride: baseNowMs + 2})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-beta-job-001", Payload: map[string]any{"kind": "monitor-groups", "slot": "beta-1"}, GID: "beta", GroupLimit: 1, NowMsOverride: baseNowMs + 3})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-gamma-job-001", Payload: map[string]any{"kind": "monitor-groups", "slot": "gamma-1"}, GID: "gamma", GroupLimit: 1, NowMsOverride: baseNowMs + 4})
	mustPublish(client, omniq.PublishOpts{Queue: queue, JobID: queue + "-delta-job-001", Payload: map[string]any{"kind": "monitor-groups", "slot": "delta-1"}, GID: "delta", GroupLimit: 1, NowMsOverride: baseNowMs + 5})

	res, err := client.Reserve(queue, baseNowMs+100)
	if err != nil {
		fail(err)
	}
	if _, ok := res.(omniq.ReserveJob); !ok {
		fail(fmt.Errorf("unexpected reserve response: %#v", res))
	}

	gids := []string{"alpha", "beta", "gamma", "delta"}
	groupsReadyPage := monitor.GroupsReady(queue, 0, 2)
	groupsReadyAll := monitor.GroupsReadyWithScores(queue, 0, 10)
	groupStatus := monitor.GroupStatus(queue, gids, 1)

	out := resultView{
		SDK:             "go",
		Queue:           queue,
		GroupsReadyPage: groupsReadyPage,
		GroupsReadyAll:  groupsReadyAll,
		GroupStatus:     groupStatus,
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
