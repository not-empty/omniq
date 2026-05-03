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
	SDK         string   `json:"sdk"`
	QueuesFound []string `json:"queues_found"`
	QueueAState string   `json:"queue_a_state"`
	QueueBState string   `json:"queue_b_state"`
	HeartbeatB  int64    `json:"heartbeat_b"`
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
	queuePrefix := getenv("QUEUE", "validation-s31-go")
	queueA := queuePrefix + "-a"
	queueB := queuePrefix + "-b"
	baseNowMs := int64(1775450000000)

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

	scriptFlush(seed, ctx)
	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queueA,
		JobID:         queueA + "-job-001",
		Payload:       map[string]any{"kind": "multi-queue-noscript", "queue": "a"},
		NowMsOverride: baseNowMs + 1,
	})
	if err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queueB,
		JobID:         queueB + "-job-001",
		Payload:       map[string]any{"kind": "multi-queue-noscript", "queue": "b"},
		NowMsOverride: baseNowMs + 2,
	})
	if err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	reservedA := reserveJob(client, queueA, baseNowMs+100)

	scriptFlush(seed, ctx)
	if err := client.AckSuccess(queueA, reservedA.JobID, reservedA.LeaseToken, baseNowMs+110); err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	reservedB := reserveJob(client, queueB, baseNowMs+120)

	scriptFlush(seed, ctx)
	heartbeatB, err := client.Heartbeat(queueB, reservedB.JobID, reservedB.LeaseToken, baseNowMs+130)
	if err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	if err := client.AckSuccess(queueB, reservedB.JobID, reservedB.LeaseToken, baseNowMs+140); err != nil {
		fail(err)
	}

	queuesFound := []string{}
	for _, q := range monitor.ScanQueues() {
		if q == queueA || q == queueB {
			queuesFound = append(queuesFound, q)
		}
	}

	queueAState, err := client.Ops().R.HGet("{"+queueA+"}:job:"+queueA+"-job-001", "state")
	if err != nil {
		fail(err)
	}
	queueBState, err := client.Ops().R.HGet("{"+queueB+"}:job:"+queueB+"-job-001", "state")
	if err != nil {
		fail(err)
	}
	if len(queuesFound) != 2 {
		fail(fmt.Errorf("unexpected discovered queues: %#v", queuesFound))
	}
	if deref(queueAState) != "completed" || deref(queueBState) != "completed" {
		fail(fmt.Errorf("multi-queue NOSCRIPT flow did not complete both jobs"))
	}
	if heartbeatB <= 0 {
		fail(fmt.Errorf("heartbeat did not extend queue B lease"))
	}

	out := resultView{
		SDK:         "go",
		QueuesFound: queuesFound,
		QueueAState: deref(queueAState),
		QueueBState: deref(queueBState),
		HeartbeatB:  heartbeatB,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(b))
}

func scriptFlush(seed redis.UniversalClient, ctx context.Context) {
	if cluster, ok := seed.(*redis.ClusterClient); ok {
		err := cluster.ForEachMaster(ctx, func(ctx context.Context, c *redis.Client) error {
			return c.ScriptFlush(ctx).Err()
		})
		if err != nil {
			fail(err)
		}
		return
	}

	if err := seed.ScriptFlush(ctx).Err(); err != nil {
		fail(err)
	}
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
