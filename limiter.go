package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	"math/rand"
	"sync"
	"time"
)

const maxRate = 15

type RateLimit struct {
	Redis   *redis.Client
	maxRate int64
}

func NewRateLimit(addr string, rate int64) RateLimit {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		PoolSize: 40,
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	return RateLimit{Redis: client, maxRate: rate}
}

func (l *RateLimit) Add(i int) {
	multi := l.Redis.Multi()
	multi.Watch("wg")
	l.CheckPauseIncr(multi)
	multi.Incr("wg")
	multi.Unwatch("wg")
	multi.Close()
	//l.Redis.Incr("wg")
}

func (l *RateLimit) Done() {
	multi := l.Redis.Multi()
	multi.Watch("wg")
	multi.Decr("wg")
	multi.Unwatch("wg")
	multi.Close()
}

func (l *RateLimit) CheckPauseIncr(multi *redis.Multi) {
	for {
		wgVal, _ := multi.Get("wg").Int64()
		if wgVal < l.maxRate {
			break
		}
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	}
}

func (l *RateLimit) Wait() {
	for {
		wgVal, _ := l.Redis.Get("wg").Int64()
		if wgVal != 0 {
			time.Sleep(1 * time.Second)
			continue
		} else {
			break
		}
	}
}

var (
	wg    sync.WaitGroup
	tasks = make(chan bool, 40)
)

func startWorkers() {
	limiter := NewRateLimit("localhost:6379", maxRate)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			for data := range tasks {
				limiter.Add(1)
				fmt.Printf("Working .... %+v\n", data)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				// do some work on data
				limiter.Done()
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
