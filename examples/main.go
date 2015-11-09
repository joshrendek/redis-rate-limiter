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
	opts := limit.Options{
		Address:          "localhost:6379",
		LockName:         "wg",
		MaxRate:          15,
		LockWaitDuration: 100 * time.Millisecond,
		WorkerTimeout:    5 * time.Second,
	}
	limiter := limit.NewRateLimit(opts)
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

	for i := 0; i < 5000; i++ {
		tasks <- true
		time.Sleep(100 * time.Millisecond)
	}

	close(tasks)
	wg.Wait()
}
