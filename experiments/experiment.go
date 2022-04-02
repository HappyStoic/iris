package main

import (
	"context"
	"fmt"
	"sync"
)

type experiment struct {
	dir       string
	instances []*instance
}

func (e *experiment) run(ctx context.Context) {
	fmt.Print("Starting experiment\n\n")

	wg := sync.WaitGroup{}
	for idx, i := range e.instances {

		idx := idx
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Printf("launching instance %d\n", idx)

			err := i.run(ctx)
			if err != nil {
				fmt.Printf("error running instance %d:\n%s\n", idx, err)
				return
			}
			fmt.Printf("instance %d exited", idx)
		}()
	}
	fmt.Printf("\nlogs in: %s/*/out\n\n", e.dir)
	wg.Wait()
}
