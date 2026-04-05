package main

import (
	"encoding/json"
	"fmt"
	"os"

	omniq "github.com/not-empty/omniq-go"
)

type resultView struct {
	SDK             string `json:"sdk"`
	Key             string `json:"key"`
	AckSequence     []int  `json:"ack_sequence"`
	CountExistsAfter int   `json:"count_exists_after"`
	DoneExistsAfter int    `json:"done_exists_after"`
}

func main() {
	key := getenv("KEY", "validation-s11-go")

	client, err := omniq.NewClient(omniq.ClientOpts{
		Host: "omniq-redis",
		Port: 6379,
	})
	if err != nil {
		fail(err)
	}

	if err := client.ChildsInit(key, 3); err != nil {
		fail(err)
	}

	a1, err := client.ChildAck(key, "a")
	if err != nil {
		fail(err)
	}
	a2, err := client.ChildAck(key, "a")
	if err != nil {
		fail(err)
	}
	b, err := client.ChildAck(key, "b")
	if err != nil {
		fail(err)
	}
	c, err := client.ChildAck(key, "c")
	if err != nil {
		fail(err)
	}

	base := "{cc:" + key + "}"

	countExists, err := client.Ops().R.Exists(base + ":count")
	if err != nil {
		fail(err)
	}
	doneExists, err := client.Ops().R.Exists(base + ":done")
	if err != nil {
		fail(err)
	}

	out := resultView{
		SDK:             "go",
		Key:             key,
		AckSequence:     []int{a1, a2, b, c},
		CountExistsAfter: int(countExists),
		DoneExistsAfter:  int(doneExists),
	}

	bb, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(bb))
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
