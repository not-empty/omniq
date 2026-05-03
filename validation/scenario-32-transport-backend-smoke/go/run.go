package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK           string   `json:"sdk"`
	Backend       string   `json:"backend"`
	Queue         string   `json:"queue"`
	ReserveStatus string   `json:"reserve_status"`
	ReservedJobID string   `json:"reserved_job_id"`
	QueuesFound   []string `json:"queues_found"`
}

func main() {
	queue := getenv("QUEUE", "validation-s32-go")
	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue:   queue,
		JobID:   queue + "-job-001",
		Payload: map[string]any{"kind": "transport-backend-smoke", "backend": getenv("REDIS_MODE", "standalone"), "sdk": "go"},
	})
	if err != nil {
		fail(err)
	}

	reservedRaw, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}
	reserved, ok := reservedRaw.(omniq.ReserveJob)
	if !ok {
		fail(fmt.Errorf("unexpected reserve response: %#v", reservedRaw))
	}

	queuesFound := []string{}
	for _, q := range monitor.ScanQueues() {
		if q == queue {
			queuesFound = append(queuesFound, q)
		}
	}
	if reserved.Status != "JOB" {
		fail(fmt.Errorf("unexpected reserve status: %s", reserved.Status))
	}
	if len(queuesFound) != 1 || queuesFound[0] != queue {
		fail(fmt.Errorf("unexpected discovered queues: %#v", queuesFound))
	}

	out := resultView{
		SDK:           "go",
		Backend:       getenv("REDIS_MODE", "standalone"),
		Queue:         queue,
		ReserveStatus: reserved.Status,
		ReservedJobID: reserved.JobID,
		QueuesFound:   queuesFound,
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
