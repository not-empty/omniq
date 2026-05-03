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
	SDK                string `json:"sdk"`
	Queue              string `json:"queue"`
	PublishedJobID     string `json:"published_job_id"`
	ReservedJobID      string `json:"reserved_job_id"`
	HeartbeatLockUntil int64  `json:"heartbeat_lock_until_ms"`
	CompletedState     string `json:"completed_state"`
	DelayedJobID       string `json:"delayed_job_id"`
	PromotedCount      int    `json:"promoted_count"`
	PromotedState      string `json:"promoted_state"`
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

func mustPublish(client *omniq.Client, opts omniq.PublishOpts) string {
	id, err := client.Publish(opts)
	if err != nil {
		fail(err)
	}
	return id
}

func main() {
	queue := getenv("QUEUE", "validation-s25-go")
	baseNowMs := int64(1775420000000)

	publishJob := queue + "-job-001"
	delayedJob := queue + "-delayed-001"

	client, err := omniq.NewClient(omniq.ClientOpts{Host: getenv("REDIS_HOST", "omniq-redis"), Port: 6379})
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	seed := newRawRedis()
	defer func() { _ = seed.Close() }()

	scriptFlush(seed, ctx)
	publishedJobID := mustPublish(client, omniq.PublishOpts{
		Queue:         queue,
		JobID:         publishJob,
		Payload:       map[string]any{"kind": "noscript-recovery", "slot": "publish"},
		NowMsOverride: baseNowMs + 1,
	})

	scriptFlush(seed, ctx)
	reserved := reserveJob(client, queue, baseNowMs+100)

	scriptFlush(seed, ctx)
	heartbeatLockUntil, err := client.Heartbeat(queue, reserved.JobID, reserved.LeaseToken, baseNowMs+110)
	if err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	if err := client.AckSuccess(queue, reserved.JobID, reserved.LeaseToken, baseNowMs+120); err != nil {
		fail(err)
	}

	scriptFlush(seed, ctx)
	delayedJobID := mustPublish(client, omniq.PublishOpts{
		Queue:         queue,
		JobID:         delayedJob,
		Payload:       map[string]any{"kind": "noscript-recovery", "slot": "delayed"},
		DueMs:         baseNowMs + 500,
		NowMsOverride: baseNowMs + 2,
	})

	scriptFlush(seed, ctx)
	promotedCount, err := client.PromoteDelayed(queue, 10, baseNowMs+600)
	if err != nil {
		fail(err)
	}

	completedStateRaw, err := client.Ops().R.HGet("{"+queue+"}:job:"+publishJob, "state")
	if err != nil {
		fail(err)
	}
	promotedStateRaw, err := client.Ops().R.HGet("{"+queue+"}:job:"+delayedJob, "state")
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                "go",
		Queue:              queue,
		PublishedJobID:     publishedJobID,
		ReservedJobID:      reserved.JobID,
		HeartbeatLockUntil: heartbeatLockUntil,
		CompletedState:     deref(completedStateRaw),
		DelayedJobID:       delayedJobID,
		PromotedCount:      promotedCount,
		PromotedState:      deref(promotedStateRaw),
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
