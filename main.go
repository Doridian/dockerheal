package main

import (
	"context"
	"time"
)

func main() {
	client, err := NewDockerhealClient()
	if err != nil {
		panic(err)
	}

	time.Sleep(60 * time.Second)

	err = client.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}
