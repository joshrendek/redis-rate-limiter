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
	Redis        *redis.Client
	maxRate      int64
	RateDuration time.Duration
}

func (l *RateLimit) Add(i int) {
	lock_check := l.Redis.SetNX("wg_lock", "t", 0).Val()
	for {
		if lock_check {
			break
		}
		time.Sleep(l.RateDuration)
		lock_check = l.Redis.SetNX("wg_lock", "t", 0).Val()
	}
	l.CheckPauseIncr()
	l.Redis.Incr("wg")
	l.Redis.Del("wg_lock")
	//l.Redis.Incr("wg")
}

func (l *RateLimit) Done() {
	l.Redis.Decr("wg")
}

func (l *RateLimit) CheckPauseIncr() {
	for {
		wgVal, _ := l.Redis.Get("wg").Int64()
		if wgVal < l.maxRate {
			break
		}
		//time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
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

func NewRateLimit(addr string, rate int64, duration time.Duration) RateLimit {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		PoolSize: 40,
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	return RateLimit{Redis: client, maxRate: rate, RateDuration: duration}
}

func startWorkers() {
	limiter := NewRateLimit("localhost:6379", maxRate, 100*time.Millisecond)
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
