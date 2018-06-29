package workerful

import (
	"testing"
	"time"
)

// Job implement the job interface
type CustomJob struct {
	Responses chan int
	ID        int
}

func (mj CustomJob) DoTheJob() {
	time.Sleep(time.Second)
	//println("job", mj.ID, "executed...")
	mj.Responses <- 1
}

func TestJobs(t *testing.T) {
	jobsNum := 48
	workers := 16
	wp := NewWorkerful("", &Config{jobsNum, workers})

	responses := make(chan int, jobsNum)
	start := time.Now()

	i := 0
	for i < int(jobsNum) {
		wp.PushJob(CustomJob{responses, i})
		i++
	}

	count := 0
	for i := range responses {
		count += i
		if count == jobsNum {
			close(responses)
		}
	}

	wp.Close()

	timeElapsed := time.Since(start).Seconds()
	timeNeeded := float64(jobsNum) / float64(workers)

	if timeElapsed < timeNeeded {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed, timeNeeded)
	}

	if timeElapsed > (timeNeeded * 1.1) {
		t.Errorf("Time elapsed (%f) too long (time needed * 1.1: %f), something is wrong", timeElapsed, timeNeeded*1.1)
	}
}

func TestFuncs(t *testing.T) {
	jobsNum := 48
	workers := 16
	wp := NewWorkerful("", &Config{jobsNum, workers})

	responses := make(chan int, jobsNum)
	start := time.Now()

	i := 0
	for i < int(jobsNum) {
		//j := i
		wp.PushFunc(func() {
			time.Sleep(time.Second)
			//println("job", j, "executed...")
			responses <- 1
		})
		i++
	}

	count := 0
	for i := range responses {
		count += i
		if count == jobsNum {
			close(responses)
		}
	}

	wp.Close()

	timeElapsed := time.Since(start).Seconds()
	timeNeeded := float64(jobsNum) / float64(workers)

	if timeElapsed < timeNeeded {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed, timeNeeded)
	}

	if timeElapsed > (timeNeeded * 1.1) {
		t.Errorf("Time elapsed (%f) too long (time needed * 1.1: %f), something is wrong", timeElapsed, timeNeeded*1.1)
	}
}
