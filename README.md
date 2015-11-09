## Why?


If you:

* Have several distributed workers that are dependent on a rate limited service
* Need to control how fast a group of workers and how quickly they hit an API/Service

Then this is the package for you.


## Installation

`go get github.com/joshrendek/redis-rate-limit`



## Usage

See `examples/main.go`

Importing:

```
import(
	"github.com/joshrendek/redis-rate-limit"
)
```

Using with a regular WG - we can have other parts of a job run at one
concurrency (lets say this is CPU intensive) and then sending to an API.

We want to run N workers but also limit the amount of workers that can
concurrently access the limited resource:

``` go
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

```


## License

```

The MIT License (MIT)

Copyright (c) 2015 Josh Rendek

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
