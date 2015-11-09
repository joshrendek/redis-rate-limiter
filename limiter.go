package limit

import (
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/redis.v3"
	"sync"
	"time"
)

// Options is used to contain config options for the limiter
// Address is the address of the redis server
// MaxRate is the concurrency limit
// LockWaitDuration is used as a duration for busy waiting on the lock key
// WorkerTimeout is used as the TTL for the worker key, this should be set
// longer than you expect your workers to take.
type Options struct {
	Address          string
	MaxRate          int64
	LockWaitDuration time.Duration
	WorkerTimeout    time.Duration
	LockName         string
}

// RateLimit is the limiter object exposed to control concurrency
type RateLimit struct {
	Redis *redis.Client
	Opts  Options
}

// Add is for telling wait group something is starting
// A lock using SetNX is created in redis based on the l.Opts.LockName
// The lock busy waits until its free and then acquires it
// LockWaitDuration should be set to something reasonable so the CPU
// isn't wasting cycles.
// Add handles generating the UUID for the lock set and also
// the lock keys that are given a TTL
// The return value should be stored so you can pass the
// uid to Done()
func (l *RateLimit) Add(i int) string {
	lockCheck := l.Redis.SetNX(l.Opts.LockName+":lock", "t", 0).Val()
	l.Redis.Expire(l.Opts.LockName+":lock", 10*time.Second)
	for {
		if lockCheck {
			break
		}
		time.Sleep(l.Opts.LockWaitDuration)
		lockCheck = l.Redis.SetNX(l.Opts.LockName+":lock", "t", 0).Val()
	}
	l.checkRate()
	uid := uuid.NewV4()
	l.Redis.Set(l.Opts.LockName+":"+uid.String(), uid.String(), l.Opts.WorkerTimeout)
	l.Redis.SAdd(l.Opts.LockName, uid.String())
	l.Redis.Del(l.Opts.LockName + ":lock")
	l.cleanLocks()
	return uid.String()
}

func (l *RateLimit) cleanLocks() {
	workers := l.Redis.SMembers(l.Opts.LockName).Val()
	for _, w := range workers {
		if l.Redis.Exists(l.Opts.LockName + ":" + w).Val() {
			continue
		} else {
			// It expired! Lets make sure it gets removed from the lock set
			fmt.Println("Removing: ", w)
			l.Redis.SRem(l.Opts.LockName, w)
		}
	}
}

// Done Instructs the limiter to remove the lock key
// and uuid from the lock set
func (l *RateLimit) Done(uid string) {
	l.Redis.Del(l.Opts.LockName + ":" + uid)
	l.Redis.SRem(l.Opts.LockName, uid)
}

func (l *RateLimit) checkRate() {
	for {
		wgVal := l.Redis.SCard(l.Opts.LockName).Val()
		if wgVal < l.Opts.MaxRate {
			break
		}
	}
}

var (
	wg    sync.WaitGroup
	tasks = make(chan bool, 40)
)

// NewRateLimit is for setting up a new rate limiter with options
func NewRateLimit(opts Options) RateLimit {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Address,
		PoolSize: 40,
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	return RateLimit{Redis: client, Opts: opts}
}
