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

func (mj CustomJob) F() error {
	time.Sleep(time.Second)
	//println("job", mj.ID, "executed...")
	mj.Responses <- 1
	return nil
}

func TestBufferedWUnblockingPush(t *testing.T) {
	jobsNum := 48
	queueSize := 16
	workers := 16
	wp := New("", &Config{queueSize, workers})

	responses := make(chan int, jobsNum)
	start := time.Now()

	i := 0
	for i < int(jobsNum) {
		wp.PushJobAsync(CustomJob{responses, i})
		i++
	}

	count := 0
	for i := range responses {
		count += i
		if count == jobsNum {
			close(responses)
		}
	}

	wp.Stop()

	timeElapsed := time.Since(start)
	timeNeeded := time.Duration(float64(jobsNum)/float64(workers)) * time.Second

	if timeElapsed < timeNeeded {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed.Seconds(), timeNeeded.Seconds())
	}

	if timeElapsed > (timeNeeded + time.Second) {
		t.Errorf("Time elapsed (%f) too long than (time needed + 1 sec): %f, something is wrong", timeElapsed.Seconds(), (timeNeeded + time.Second).Seconds())
	}
}

func TestBufferedWBlockingPush(t *testing.T) {
	jobsNum := 48
	queueSize := 16
	workers := 16
	wp := New("", &Config{queueSize, workers})

	responses := make(chan int, jobsNum)
	start := time.Now()

	i := 0
	for i < int(jobsNum) {
		wp.PushFunc(func() error {
			time.Sleep(time.Second)
			responses <- 1
			return nil
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

	wp.Stop()

	timeElapsed := time.Since(start)
	timeNeeded := time.Duration(float64(jobsNum)/float64(workers)) * time.Second

	if timeElapsed < timeNeeded {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed.Seconds(), timeNeeded.Seconds())
	}

	if timeElapsed > (timeNeeded + time.Second) {
		t.Errorf("Time elapsed (%f) too long than (time needed + 1 sec): %f, something is wrong", timeElapsed.Seconds(), (timeNeeded + time.Second).Seconds())
	}
}

func TestUnbufferedWBlockingPush(t *testing.T) {
	jobsNum := 48
	workers := 16
	wp := New("", &Config{0, workers})

	responses := make(chan int, jobsNum)
	start := time.Now()

	i := 0
	for i < int(jobsNum) {
		wp.PushFunc(func() error {
			time.Sleep(time.Second)
			responses <- 1
			return nil
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

	wp.Stop()

	timeElapsed := time.Since(start)
	timeNeeded := time.Duration(float64(jobsNum)/float64(workers)) * time.Second

	if timeElapsed < timeNeeded {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed.Seconds(), timeNeeded.Seconds())
	}

	if timeElapsed > (timeNeeded + time.Second) {
		t.Errorf("Time elapsed (%f) too long than (time needed): %f, something is wrong", timeElapsed.Seconds(), (timeNeeded + time.Second).Seconds())
	}
}

func TestStopStart(t *testing.T) {
	jobsNum := 192
	workers := 48
	wp := New("", &Config{jobsNum, workers})

	var pause time.Duration = 5

	start := time.Now()

	responses := make(chan int, jobsNum)

	i := 0
	for i < int(jobsNum/2) {
		wp.PushFuncAsync(func() error {
			time.Sleep(time.Second)
			responses <- 1
			return nil
		})
		i++
	}

	wp.Stop()
	time.Sleep(pause * time.Second)
	wp.Start()

	i = 0
	for i < int(jobsNum/2) {
		wp.PushFuncAsync(func() error {
			time.Sleep(time.Second)
			responses <- 1
			return nil
		})
		i++
	}

	wp.Stop()

	count := 0
	for i := range responses {
		count += i
		if count == jobsNum {
			close(responses)
		}
	}

	timeElapsed := time.Since(start)
	timeNeeded := (time.Duration(float64(jobsNum)/float64(workers)) + pause) * time.Second

	if timeElapsed < timeNeeded-2*time.Second {
		t.Errorf("Time elapsed (%f) too short (time needed: %f), something is wrong", timeElapsed.Seconds(), timeNeeded.Seconds())
	}

	if timeElapsed > (timeNeeded - 1*time.Second) {
		t.Errorf("Time elapsed (%f) too long than (time needed): %f, something is wrong", timeElapsed.Seconds(), timeNeeded.Seconds())
	}

}
