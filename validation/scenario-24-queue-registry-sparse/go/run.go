package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
	"github.com/redis/go-redis/v9"
)

type resultView struct {
	SDK         string             `json:"sdk"`
	QueuesFound []string           `json:"queues_found"`
	StatsEmpty  omniq.QueueStats   `json:"stats_empty"`
	StatsPartial omniq.QueueStats  `json:"stats_partial"`
	StatsPaused omniq.QueueStats   `json:"stats_paused"`
	StatsMany   []omniq.QueueStats `json:"stats_many"`
}

func main() {
	prefix := getenv("PREFIX", "validation-s24-go")
	queueEmpty := prefix + "-empty"
	queuePartial := prefix + "-partial"
	queuePaused := prefix + "-paused"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: "omniq-redis", Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	seed := redis.NewClient(&redis.Options{Addr: "omniq-redis:6379"})
	defer func() { _ = seed.Close() }()

	if err := seed.SAdd(ctx, "omniq:queues", "{"+queueEmpty+"}", "{"+queuePartial+"}", "{"+queuePaused+"}").Err(); err != nil {
		fail(err)
	}
	if err := seed.HSet(ctx, "{"+queuePartial+"}:stats",
		"waiting", "2",
		"group_waiting", "1",
		"active", "3",
		"last_activity_ms", "1775410000001",
	).Err(); err != nil {
		fail(err)
	}
	if err := seed.Set(ctx, "{"+queuePaused+"}:paused", "1", 0).Err(); err != nil {
		fail(err)
	}

	queuesFound := []string{}
	for _, q := range monitor.ListQueues() {
		if q == queueEmpty || q == queuePartial || q == queuePaused {
			queuesFound = append(queuesFound, q)
		}
	}

	out := resultView{
		SDK:          "go",
		QueuesFound:  queuesFound,
		StatsEmpty:   monitor.Stats(queueEmpty),
		StatsPartial: monitor.Stats(queuePartial),
		StatsPaused:  monitor.Stats(queuePaused),
		StatsMany:    monitor.StatsMany([]string{queueEmpty, queuePartial, queuePaused}),
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
