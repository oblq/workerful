// Package workerful provides a simple, elegant and yet a powerful
// implementation of a worker-pool by making use of sync.WaitGroup.
package workerful

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/oblq/sprbox"
	"github.com/oblq/swap"
)

// Job is a job interface, useful if you need to pass parameters or do more complicated stuff.
type Job interface {
	F() error
}

// SimpleJob is a job func useful for simple operations.
type SimpleJob func() error

// JobQueue is a job queue in which you can append Job or SimpleJob types.
type jobQueue chan interface{}

// Config defines the config for workerful.
type Config struct {
	QueueSize int `yaml:"QueueSize"`
	Workers   int `yaml:"Workers"`
}

// Workerful is the workerful instance type.
type Workerful struct {
	Config *Config

	doneCount   uint64
	failedCount uint64

	jobQueue     jobQueue
	workersGroup *sync.WaitGroup

	// stopGroup wait until the number of routines opened by PushFuncAsync() or PushJobAsync() goes to zero.
	// Necessary to avoid deadlocks by sending jobs in the jobQueue when it has been already closed.
	stopGroup *sync.WaitGroup
	// If true no more jobs can be pushed in the jobQueue.
	blockPush bool
}

// New creates and returns a new workerful instance, and starts the workers to process the queue.
//
// An important point:
// If the jobQueue channel is unbuffered, the sender blocks until the receiver has received the value.
// If the channel has a buffer, the sender blocks only until the value has been copied to the buffer;
// if the buffer is full, this means waiting until some receiver has retrieved a value.
//
// Said otherwise:
// When a channel is full, the sender waits for another goroutine to make some room by receiving.
// You can see an unbuffered channel as an always full one:
// there must be another goroutine to take what the sender sends.
//
// Basically you can't send values to an unbuffered channel
// if there is not a listener that grab those values in place.
//
// Also accept no configPath nor config, the default values will be loaded.
func New(configPath string, config *Config) *Workerful {
	wp := &Workerful{Config: &Config{0, 0}}

	if len(configPath) > 0 {
		if err := sprbox.LoadConfig(&wp.Config, configPath); err != nil {
			fmt.Printf("load config error: %v", err)
		}
	} else if config != nil {
		wp.Config = config
	}

	wp.configAndStart()
	return wp
}

// Configure is the https://github.com/oblq/swap 'Configurable' interface implementation.
func (wp *Workerful) Configure(configFiles ...string) error {
	if err := swap.Parse(wp.Config, configFiles...); err != nil {
		return err
	}
	wp.configAndStart()
	return nil
}

// SpareConfig is the https://github.com/oblq/sprbox 'configurable' interface implementation.
//
// Deprecated: use Swap package interfaces implementation instead.
// https://github.com/oblq/swap
func (wp *Workerful) SpareConfig(configFiles []string) error {
	if err := sprbox.LoadConfig(&wp.Config, configFiles...); err != nil {
		return err
	}
	wp.configAndStart()
	return nil
}

// SpareConfigBytes is the https://github.com/oblq/sprbox 'configurableInCollection' interface implementation.
//
// Deprecated: use Swap package interfaces implementation instead.
// https://github.com/oblq/swap
func (wp *Workerful) SpareConfigBytes(configBytes []byte) error {
	if err := sprbox.Unmarshal(configBytes, &wp.Config); err != nil {
		return err
	}
	wp.configAndStart()
	return nil
}

func (wp *Workerful) configAndStart() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if wp.Config.Workers == 0 {
		wp.Config.Workers = runtime.NumCPU()
	}

	// Start
	wp.jobQueue = make(jobQueue, wp.Config.QueueSize)

	// Create workers
	wp.workersGroup = &sync.WaitGroup{}
	wp.workersGroup.Add(wp.Config.Workers)
	for i := 1; i <= wp.Config.Workers; i++ {
		go wp.newWorker()
	}

	wp.stopGroup = &sync.WaitGroup{}
	wp.blockPush = false

	//println("[workerful] restarted...")

	// Wait for workers to complete
	// Wait() blocks until the WaitGroup counter is zero and the channel closed
	// not needed
	go func() {
		wp.workersGroup.Wait()
		//println("[workerful] gracefully stopped...")
	}()
}

// Stop close the jobQueue, gracefully, it is blocking.
// It is possible to Stop and Restart Workerful at any time.
// Already queued jobs will be processed.
// Jobs pushed asynchronously will be added to the queue and processed.
// It will block until all of the jobs are added to the queue and processed.
func (wp *Workerful) Stop() {
	// Disable pushing...
	wp.blockPush = true

	// stopGroup will wait until all jobs are sent to the queue
	// sending a job after the channel has been closed will cause a crash otherwise
	if wp.stopGroup != nil && wp.jobQueue != nil {
		wp.stopGroup.Wait()
		close(wp.jobQueue)
	}
}

// newWorker creates a new worker
func (wp *Workerful) newWorker() {
	// range over channels only stop after the channel has been closed
	// wg.Done() then is never called until jobQueue is closed: close(jobQueue)
	// so we have 'workers' routines executing jobs in chunks forever
	defer wp.workersGroup.Done()

	for job := range wp.jobQueue {
		switch job.(type) {
		case Job:
			if err := job.(Job).F(); err != nil {
				atomic.AddUint64(&wp.failedCount, 1)
				fmt.Printf("[workerful] error from job: %s", err.Error())
			} else {
				atomic.AddUint64(&wp.doneCount, 1)
			}
		case SimpleJob:
			if err := job.(SimpleJob)(); err != nil {
				atomic.AddUint64(&wp.failedCount, 1)
				fmt.Printf("[workerful] error from job: %s", err.Error())
			} else {
				atomic.AddUint64(&wp.doneCount, 1)
			}
		default:
			fmt.Println("[workerful] Push() func only accept `Job` and `SimpleJob` (func() error) types")
			continue
		}
	}
}

// Status return the number of processed, failed and inQueue jobs.
func (wp *Workerful) Status() (done uint64, failed uint64, inQueue int) {
	return atomic.LoadUint64(&wp.doneCount), atomic.LoadUint64(&wp.failedCount), len(wp.jobQueue)
}

// check if jobQueue is closed
func (wp *Workerful) canPush() bool {
	if wp.blockPush {
		atomic.AddUint64(&wp.failedCount, 1)
		println("[workerful] the queue is closed, can't push a new job")
		return false
	}
	return true
}

// PushJob is an helper method that add a Job to the queue.
func (wp *Workerful) PushJob(job Job) {
	if !wp.canPush() {
		return
	}
	wp.jobQueue <- job
}

// PushJobAsync is an helper method that add a Job to the queue whithout blocking.
func (wp *Workerful) PushJobAsync(job Job) {
	if !wp.canPush() {
		return
	}
	wp.stopGroup.Add(1)
	go func() {
		wp.jobQueue <- job
		wp.stopGroup.Done()
	}()
}

// PushFunc is an helper method that add a func() to the queue.
func (wp *Workerful) PushFunc(job SimpleJob) {
	if !wp.canPush() {
		return
	}
	wp.jobQueue <- job
}

// PushFuncAsync is an helper method that add a func() to the queue whithout blocking.
func (wp *Workerful) PushFuncAsync(job SimpleJob) {
	if !wp.canPush() {
		return
	}
	wp.stopGroup.Add(1)
	go func() {
		wp.jobQueue <- job
		wp.stopGroup.Done()
	}()
}
