package main

import "context"

func main() {
	client, err := NewDockerhealClient()
	if err != nil {
		panic(err)
	}
	err = client.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}
