package main

import (
	"time"

	"sync"

	"github.com/oblq/workerful"
)

func main() {
	jobs := 20

	wp := workerful.New("", &workerful.Config{QueueSize: jobs, Workers: 0})

	wg := sync.WaitGroup{}
	wg.Add(jobs)

	i := 0
	for i < jobs/2 {
		wp.PushJobAsync(CustomJob{&wg, i})
		i++
	}

	j := 0
	for j < jobs/2 {
		jj := jobs/2 + j
		wp.PushFunc(func() error {
			time.Sleep(time.Second)
			println("job", jj, "executed...")
			wg.Done()
			return nil
		})
		j++
	}

	wg.Wait()
	println("finished, jobs executed:", jobs)
	wp.Stop()
}

// CustomJob implement the workerful.Job interface (F())
type CustomJob struct {
	WG *sync.WaitGroup
	ID int
}

// F execute the job
func (cj CustomJob) F() error {
	time.Sleep(time.Second)
	println("job", cj.ID, "executed...")
	cj.WG.Done()
	return nil
}
