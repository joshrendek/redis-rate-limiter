package limit

import (
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/redis.v3"
	"sync"
	"time"
)

const lockName = "wg"

// A RateLimit is used to contain config options for the limiter
// maxRate is the concurrency limit
// RateDuration is used as a duration for busy waiting on the lock key
// workerTimeout is used as the TTL for the worker key, this should be set
// longer than you expect your workers to take.
type RateLimit struct {
	Redis         *redis.Client
	maxRate       int64
	RateDuration  time.Duration
	workerTimeout time.Duration
}

// Add()
// Add lockName:UUID to set
// Add lockName:UUID key with expiration
// Get members in set, check for key existance, remove from set if key doesn't exist
// Return uuid used

// Done(uuid)
// Remove lockName:UUID

func (l *RateLimit) acquireLock() {

}

// Add is for telling wait group something is starting
// A lock using SetNX is created in redis based on the lockName
// The lock busy waits until its free and then acquires it
// RateDuration should be set to something reasonable so the CPU
// isn't wasting cycles.
// This handles generating the UUID for the lock set and also
// the lock keys that are given a TTL
func (l *RateLimit) Add(i int) string {
	lockCheck := l.Redis.SetNX(lockName+":lock", "t", 0).Val()
	l.Redis.Expire(lockName+":lock", 10*time.Second)
	for {
		if lockCheck {
			break
		}
		time.Sleep(l.RateDuration)
		lockCheck = l.Redis.SetNX(lockName+":lock", "t", 0).Val()
	}
	l.checkRate()
	uid := uuid.NewV4()
	l.Redis.Set(lockName+":"+uid.String(), uid.String(), l.workerTimeout)
	l.Redis.SAdd(lockName, uid.String())
	l.Redis.Del(lockName + ":lock")
	l.cleanLocks()
	return uid.String()
}

func (l *RateLimit) cleanLocks() {
	workers := l.Redis.SMembers(lockName).Val()
	for _, w := range workers {
		if l.Redis.Exists(lockName + ":" + w).Val() {
			continue
		} else {
			// It expired! Lets make sure it gets removed from the lock set
			fmt.Println("Removing: ", w)
			l.Redis.SRem(lockName, w)
		}
	}
}

// Done Instructs the limiter to remove the lock key
// and uuid from the lock set
func (l *RateLimit) Done(uid string) {
	l.Redis.Del(lockName + ":" + uid)
	l.Redis.SRem(lockName, uid)
}

func (l *RateLimit) checkRate() {
	for {
		wgVal := l.Redis.SCard(lockName).Val()
		if wgVal < l.maxRate {
			break
		}
	}
}

var (
	wg    sync.WaitGroup
	tasks = make(chan bool, 40)
)

// NewRateLimit is for setting up a new rate limiter with options
func NewRateLimit(addr string, rate int64, duration time.Duration, workerTimeout time.Duration) RateLimit {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		PoolSize: 40,
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	return RateLimit{Redis: client, maxRate: rate, RateDuration: duration, workerTimeout: workerTimeout}
}
