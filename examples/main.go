package main

import (
	"fmt"
	"github.com/joshrendek/redis-rate-limit"
	"math/rand"
	"sync"
	"time"
)

var (
	wg    sync.WaitGroup
	tasks = make(chan bool, 40)
)

func startWorkers() {
	limiter := limit.NewRateLimit("localhost:6379", 15, 100*time.Millisecond, 5*time.Second)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			for data := range tasks {
				uid := limiter.Add(1)
				fmt.Printf("Working .... %+v\n", data)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				// do some work on data
				limiter.Done(uid)
			}
			wg.Done()
		}()
	}
}

func main() {

	go startWorkers()

	//for i := 0; i < 50; i++ {
	for {
		tasks <- true
		time.Sleep(100 * time.Millisecond)
	}

	//// Push to it like this:
	//tasks <- someData

	// Finish like this
	close(tasks)
	wg.Wait()
}
