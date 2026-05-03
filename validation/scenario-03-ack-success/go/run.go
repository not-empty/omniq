package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK                              string `json:"sdk"`
	Queue                            string `json:"queue"`
	JobID                            string `json:"job_id"`
	AckSuccessOK                     bool   `json:"ack_success_ok"`
	InvalidTokenError                string `json:"invalid_token_error"`
	InvalidTokenContainsTokenMismatch bool  `json:"invalid_token_contains_token_mismatch"`
}

func main() {
	queue := getenv("QUEUE", "validation-s03-go")
	jobID := getenv("JOB_ID", queue+"-job-001")

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: getenv("REDIS_HOST", "omniq-redis"),
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	_, err = client.Publish(omniq.PublishOpts{
		Queue: queue,
		JobID: jobID,
		Payload: map[string]any{
			"kind":   "ack-success",
			"source": "validation",
			"sdk":    "go",
			"value":  3,
		},
		Timeout:     30_000,
		MaxAttempts: 3,
		Backoff:     5_000,
	})
	if err != nil {
		fail(err)
	}

	res, err := client.Reserve(queue, 0)
	if err != nil {
		fail(err)
	}

	job, ok := res.(omniq.ReserveJob)
	if !ok {
		fail(fmt.Errorf("unexpected reserve result: %#v", res))
	}

	badErr := ""
	err = client.AckSuccess(queue, job.JobID, "bad-token", 0)
	if err != nil {
		badErr = err.Error()
	}

	err = client.AckSuccess(queue, job.JobID, job.LeaseToken, 0)
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:                               "go",
		Queue:                             queue,
		JobID:                             job.JobID,
		AckSuccessOK:                      true,
		InvalidTokenError:                 badErr,
		InvalidTokenContainsTokenMismatch: strings.Contains(badErr, "TOKEN_MISMATCH"),
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
