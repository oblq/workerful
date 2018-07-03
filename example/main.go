package main

import (
	"time"

	"github.com/oblq/workerful"
)

func main() {
	jobs := 20

	wp := workerful.New("", &workerful.Config{QueueSize: jobs, Workers: 0})

	responses := make(chan int)

	i := 0
	for i < jobs/2 {
		wp.PushJobAsync(CustomJob{responses, i})
		i++
	}

	j := 0
	for j < jobs/2 {
		jj := jobs/2 + j
		wp.PushFunc(func() error {
			responses <- jj
			return nil
		})
		j++
	}

	count := 0
	for jobID := range responses {
		println("job", jobID, "executed...")
		count++
		if count == jobs {
			println("finished, jobs executed:", count)
			close(responses)
		}
	}

	wp.Stop()
}

// CustomJob implement the workerful.Job interface (F())
type CustomJob struct {
	Responses chan int
	ID        int
}

// F execute the job
func (cj CustomJob) F() error {
	time.Sleep(time.Second)
	cj.Responses <- cj.ID
	return nil
}
