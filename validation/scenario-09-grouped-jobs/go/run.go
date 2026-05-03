package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type reserveRow struct {
	JobID string `json:"job_id"`
	GID   string `json:"gid"`
}

type resultView struct {
	SDK                string       `json:"sdk"`
	Queue              string       `json:"queue"`
	GroupLimitAlpha    string       `json:"group_limit_alpha"`
	ReserveOrder       []reserveRow `json:"reserve_order"`
	FourthReserveStatus string      `json:"fourth_reserve_status"`
	FifthReserveJobID  string       `json:"fifth_reserve_job_id"`
	FifthReserveGID    string       `json:"fifth_reserve_gid"`
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
	queue := getenv("QUEUE", "validation-s09-go")
	baseNowMs := int64(1775270000000)
	alphaFirst := queue + "-alpha-job-001"
	alphaSecond := queue + "-alpha-job-002"
	betaFirst := queue + "-beta-job-001"
	ungrouped := queue + "-ungrouped-job-001"

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: getenv("REDIS_HOST", "omniq-redis"),
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queue,
		JobID:         alphaFirst,
		Payload:       map[string]any{"kind": "grouped-jobs", "slot": "alpha-1", "sdk": "go"},
		GID:           "alpha",
		GroupLimit:    1,
		NowMsOverride: baseNowMs + 1,
	})
	if err != nil {
		fail(err)
	}
	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queue,
		JobID:         alphaSecond,
		Payload:       map[string]any{"kind": "grouped-jobs", "slot": "alpha-2", "sdk": "go"},
		GID:           "alpha",
		GroupLimit:    5,
		NowMsOverride: baseNowMs + 2,
	})
	if err != nil {
		fail(err)
	}
	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queue,
		JobID:         betaFirst,
		Payload:       map[string]any{"kind": "grouped-jobs", "slot": "beta-1", "sdk": "go"},
		GID:           "beta",
		GroupLimit:    1,
		NowMsOverride: baseNowMs + 3,
	})
	if err != nil {
		fail(err)
	}
	_, err = client.Publish(omniq.PublishOpts{
		Queue:         queue,
		JobID:         ungrouped,
		Payload:       map[string]any{"kind": "grouped-jobs", "slot": "ungrouped-1", "sdk": "go"},
		NowMsOverride: baseNowMs + 4,
	})
	if err != nil {
		fail(err)
	}

	first := reserveJob(client, queue, baseNowMs+100)
	second := reserveJob(client, queue, baseNowMs+101)
	third := reserveJob(client, queue, baseNowMs+102)

	fourthRes, err := client.Reserve(queue, baseNowMs+103)
	if err != nil {
		fail(err)
	}
	fourthStatus := ""
	if fourthRes == nil {
		fourthStatus = "EMPTY"
	} else if paused, ok := fourthRes.(omniq.ReservePaused); ok {
		fourthStatus = paused.Status
	} else {
		fail(fmt.Errorf("unexpected fourth reserve response: %#v", fourthRes))
	}

	err = client.AckSuccess(queue, first.JobID, first.LeaseToken, baseNowMs+200)
	if err != nil {
		fail(err)
	}
	fifth := reserveJob(client, queue, baseNowMs+201)

	limitRaw, err := client.Ops().R.Get("{" + queue + "}:g:alpha:limit")
	if err != nil {
		fail(err)
	}
	groupLimitAlpha := ""
	if limitRaw != nil {
		groupLimitAlpha = *limitRaw
	}

	out := resultView{
		SDK:             "go",
		Queue:           queue,
		GroupLimitAlpha: groupLimitAlpha,
		ReserveOrder: []reserveRow{
			{JobID: first.JobID, GID: first.GID},
			{JobID: second.JobID, GID: second.GID},
			{JobID: third.JobID, GID: third.GID},
		},
		FourthReserveStatus: fourthStatus,
		FifthReserveJobID: fifth.JobID,
		FifthReserveGID:   fifth.GID,
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
