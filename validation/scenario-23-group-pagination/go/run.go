package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                    string              `json:"sdk"`
	Queue                  string              `json:"queue"`
	GroupsReadyPage1       []string            `json:"groups_ready_page_1"`
	GroupsReadyPage2       []string            `json:"groups_ready_page_2"`
	GroupsReadyScoredPage1 []omniq.GroupReady  `json:"groups_ready_scored_page_1"`
	GroupsReadyScoredPage2 []omniq.GroupReady  `json:"groups_ready_scored_page_2"`
	GroupStatus            []omniq.GroupStatus `json:"group_status"`
	GroupsReadyRaw         []string            `json:"groups_ready_raw"`
}

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) {
	_, err := client.Publish(opts)
	if err != nil {
		fail(err)
	}
}

func main() {
	queue := getenv("QUEUE", "validation-s23-go")
	baseNowMs := int64(1775400000000)
	gids := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta"}

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	for i, gid := range gids {
		mustPublish(client, omniq.PublishOpts{
			Queue:         queue,
			JobID:         queue + "-" + gid + "-job-001",
			Payload:       map[string]any{"kind": "group-pagination", "gid": gid, "slot": 1},
			GID:           gid,
			GroupLimit:    1,
			NowMsOverride: baseNowMs + int64(i) + 1,
		})
	}

	page1 := monitor.GroupsReady(queue, 0, 3)
	page2 := monitor.GroupsReady(queue, 3, 3)
	scoredPage1 := monitor.GroupsReadyWithScores(queue, 0, 3)
	scoredPage2 := monitor.GroupsReadyWithScores(queue, 3, 3)
	status := monitor.GroupStatus(queue, []string{"alpha", "delta", "eta"}, 1)

	raw, err := client.Ops().R.ZRange("{"+queue+"}:groups:ready", 0, -1)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                    "go",
		Queue:                  queue,
		GroupsReadyPage1:       page1,
		GroupsReadyPage2:       page2,
		GroupsReadyScoredPage1: scoredPage1,
		GroupsReadyScoredPage2: scoredPage2,
		GroupStatus:            status,
		GroupsReadyRaw:         raw,
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
