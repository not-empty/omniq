package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	omniq "github.com/not-empty/omniq-go"
	"github.com/redis/go-redis/v9"
)

type resultView struct {
	SDK                  string   `json:"sdk"`
	QueuesFound          []string `json:"queues_found"`
	StatsManyAuto        []string `json:"stats_many_auto"`
	PausedOnlyDiscovered bool     `json:"paused_only_discovered"`
	InvalidDiscovered    bool     `json:"invalid_discovered"`
}

func main() {
	prefix := getenv("PREFIX", "validation-s30-go")
	queueA := prefix + "-alpha"
	queueB := prefix + ".beta_2"
	pausedOnly := prefix + "-paused-only"
	invalidColonKey := prefix + "-bad:name:stats"
	invalidSpaceKey := "{" + prefix + " bad}:stats"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}
	monitor, err := omniq.NewMonitor(client)
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	seed := newRawRedis()
	defer func() { _ = seed.Close() }()

	if err := seed.HSet(ctx, "{"+queueA+"}:stats", "waiting", "0").Err(); err != nil {
		fail(err)
	}
	if err := seed.HSet(ctx, "{"+queueB+"}:stats", "waiting", "1").Err(); err != nil {
		fail(err)
	}
	if err := seed.Set(ctx, "{"+pausedOnly+"}:paused", "1", 0).Err(); err != nil {
		fail(err)
	}
	if err := seed.HSet(ctx, invalidColonKey, "waiting", "9").Err(); err != nil {
		fail(err)
	}
	if err := seed.HSet(ctx, invalidSpaceKey, "waiting", "9").Err(); err != nil {
		fail(err)
	}

	queuesFound := []string{}
	for _, q := range monitor.ScanQueues() {
		if strings.HasPrefix(q, prefix) {
			queuesFound = append(queuesFound, q)
		}
	}
	sort.Strings(queuesFound)

	statsManyAuto := []string{}
	for _, row := range monitor.StatsMany(nil) {
		if strings.HasPrefix(row.Queue, prefix) {
			statsManyAuto = append(statsManyAuto, row.Queue)
		}
	}
	sort.Strings(statsManyAuto)
	expected := []string{queueA, queueB}
	sort.Strings(expected)

	if !equalStrings(queuesFound, expected) {
		fail(fmt.Errorf("unexpected discovered queues: %#v", queuesFound))
	}
	if !equalStrings(statsManyAuto, expected) {
		fail(fmt.Errorf("unexpected stats_many() discovery: %#v", statsManyAuto))
	}
	if contains(queuesFound, pausedOnly) {
		fail(fmt.Errorf("paused-only queue should not be discovered"))
	}
	if anyContains(queuesFound, "bad") {
		fail(fmt.Errorf("invalid sparse keys leaked into queue discovery"))
	}

	out := resultView{
		SDK:                  "go",
		QueuesFound:          queuesFound,
		StatsManyAuto:        statsManyAuto,
		PausedOnlyDiscovered: contains(queuesFound, pausedOnly),
		InvalidDiscovered:    anyContains(queuesFound, "bad"),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(b))
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func anyContains(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newRawRedis() redis.UniversalClient {
	if getenv("REDIS_MODE", "standalone") == "cluster" {
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{getenv("REDIS_HOST", "omniq-redis") + ":6379"},
		})
	}

	return redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{getenv("REDIS_HOST", "omniq-redis") + ":6379"},
		DB:    0,
	})
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
