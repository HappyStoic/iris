package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func now() string {
	return time.Now().Format("15:04-Jan02")
}

// diferent setup for experiments:
//		connections cannot have time based protection against trimming. I need the network to converge into stable connections asap
//		different bucket size for DHT?

func main() {
	dir := fmt.Sprintf("/home/yourmother/moje/projects/p2pnetwork/experiments/out/%s", now())
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("output dir: %s\n", dir)

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\nreceived interrupt, stopping...")
		cancel()
	}()

	n := 30
	instances := make([]*instance, 0, n)

	var prev *instance
	for i := 0; i < n; i++ {
		fmt.Printf("creating instance %d\n", i)
		cur, err := NewInstance(fmt.Sprintf("%s/ins%d", dir, i))
		if err != nil {
			log.Fatal(err)
		}

		instances = append(instances, cur)
		if prev != nil {
			cur.addInitConn(prev)
		}
		prev = cur
	}

	e := experiment{
		dir:       dir,
		instances: instances,
	}
	e.run(ctx)
}
