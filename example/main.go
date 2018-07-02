package main

import (
	"time"

	"github.com/oblq/workerful"
)

func main() {

	responses := make(chan int, 0)

	wp := workerful.NewWorkerful("", &workerful.Config{QueueSize: 0, Workers: 0})

	i := 0
	for i < int(10) {
		wp.PushJob(CustomJob{responses, i})
		i++
	}

	j := 0
	for j < int(10) {
		jj := 10 + j
		wp.PushFunc(func() error {
			println("job", jj, "executed...")
			responses <- 1
			return nil
		})
		j++
	}

	count := 0
	for i := range responses {
		count += i
		if count == 20 {
			close(responses)
		}
	}

	wp.Stop()
}

// CustomJob implement the workerful.Job interface (DoTheJob())
type CustomJob struct {
	Responses chan int
	ID        int
}

func (cj CustomJob) F() error {
	time.Sleep(time.Second)
	println("job", cj.ID, "executed...")
	cj.Responses <- 1
	return nil
}
