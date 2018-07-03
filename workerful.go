// Package workerful provides a simple, elegant and yet a powerful
// implementation of a worker-pool by making use of sync.WaitGroup.
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
package workerful

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v2"
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
	QueueSize int `yaml:"queue_size"`
	Workers   int `yaml:"workers"`
}

// Workerful is the workerful instance type.
type Workerful struct {
	Config *Config

	DoneCount   uint64
	FailedCount uint64

	jobQueue     jobQueue
	workersGroup *sync.WaitGroup

	// stopGroup wait until the number of routines opened by PushFuncAsync() or PushJobAsync() goes to zero.
	// Necessary to avoid deadlocks by sending jobs in the jobQueue when it has been already closed.
	stopGroup *sync.WaitGroup
	// If true no more jobs can be pushed in the jobQueue.
	queueClosed bool
}

var once sync.Once
var instance *Workerful

// InitShared initialize the shared singleton,
// workerful also accept no configPath nor Config, the default values will be loaded.
func InitShared(configPath string, config interface{}) error {
	once.Do(func() {
		if config, ok := config.(*Config); ok {
			instance = New("", config)
			return
		}
		if config, ok := config.(Config); ok {
			instance = New("", &config)
			return
		}
		instance = New(configPath, nil)
	})
	return nil
}

// Shared returns the workerful singleton.
//  workerful.Shared().PushAsync(func() { println("test") })
func Shared() *Workerful {
	return instance
}

// NewWorkerful creates and returns a new workerful instance, and starts the workers to process the queue.
func New(configPath string, config *Config) *Workerful {

	if len(configPath) > 0 {
		compsConfigPath := filepath.Join(configPath, "workerful.yml")
		if compsConfigFile, err := ioutil.ReadFile(compsConfigPath); err != nil {
			log.Fatalln("Wrong config path", err)
		} else if err = yaml.Unmarshal(compsConfigFile, &config); err != nil {
			log.Fatalln("Can't unmarshal config file", err)
		}
	} else if config == nil {
		config = &Config{0, 0}
	}

	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
		runtime.GOMAXPROCS(config.Workers)
	}

	wp := &Workerful{
		Config:       config,
		jobQueue:     make(jobQueue, config.QueueSize),
		workersGroup: &sync.WaitGroup{},
		queueClosed:  false,
		stopGroup:    &sync.WaitGroup{},
	}

	// Create workers
	wp.workersGroup.Add(wp.Config.Workers)
	for i := 1; i <= wp.Config.Workers; i++ {
		go wp.newWorker()
	}

	println("[workerful] started...")

	// Wait for workers to complete
	// Wait() blocks until the WaitGroup counter is zero and the channel closed
	go func() {
		wp.workersGroup.Wait()
		println("[workerful] gracefully stopped...")
	}()

	return wp
}

func (wp *Workerful) newWorker() {
	// range over channels only stop after the channel has been closed
	// wg.Done() then is never called until jobQueue is closed: close(jobQueue)
	// so we have 'workers' routines executing jobs in chunks forever
	defer wp.workersGroup.Done()

	for job := range wp.jobQueue {
		switch job.(type) {
		case Job:
			if err := job.(Job).F(); err != nil {
				atomic.AddUint64(&wp.FailedCount, 1)
				log.Printf("[workerful] error from job: %s", err.Error())
			} else {
				atomic.AddUint64(&wp.DoneCount, 1)
			}
		case SimpleJob:
			if err := job.(SimpleJob)(); err != nil {
				atomic.AddUint64(&wp.FailedCount, 1)
				log.Printf("[workerful] error from job: %s", err.Error())
			} else {
				atomic.AddUint64(&wp.DoneCount, 1)
			}
		default:
			log.Println("[workerful] Push() func only accept `Job` (see workerful.Job interface) and `func() error` types")
			continue
		}
	}
}

// Stop close the jobQueue, gracefully, it is blocking.
// Already queued jobs will be processed.
// It is possible to Stop and Restart Workerful at any time.
// If you continue to send funcs/jobs after Stop() with a buffered jobQueue
// it will block until all of the jobs are added to the queue.
func (wp *Workerful) Stop() {
	// stopGroup will waint until all jobs are sent to the queue
	// send a job after the channel has been closed will cause a crash otherwise
	wp.stopGroup.Wait()
	wp.queueClosed = true
	close(wp.jobQueue)
}

// Restart will launch the workers to process jobs.
// It is possible to Stop and Restart Workerful at any time.
func (wp *Workerful) Restart() {
	wp.jobQueue = make(jobQueue, wp.Config.QueueSize)

	// Create workers
	wp.workersGroup = &sync.WaitGroup{}
	wp.workersGroup.Add(wp.Config.Workers)
	for i := 1; i <= wp.Config.Workers; i++ {
		go wp.newWorker()
	}

	wp.stopGroup = &sync.WaitGroup{}
	wp.queueClosed = false

	println("[workerful] restarted...")

	// Wait for workers to complete
	// Wait() blocks until the WaitGroup counter is zero and the channel closed
	// we don't need it
	go func() {
		wp.workersGroup.Wait()
		println("[workerful] gracefully stopped...")
	}()
}

// Status return the number of processed, failed and inQueue jobs.
func (wp *Workerful) Status() (done uint64, failed uint64, inQueue int) {
	return atomic.LoadUint64(&wp.DoneCount), atomic.LoadUint64(&wp.FailedCount), len(wp.jobQueue)
}

func (wp *Workerful) canPush() bool {
	if wp.queueClosed {
		atomic.AddUint64(&wp.FailedCount, 1)
		println("[workerful] the queue is closed, can't push a new job")
		return false
	}
	return true
}

// PushJob is an helper method that add a Job to the queue.
// With a buffered chan queue (queue_size != 0),
// when it is full wp.JobQueue <- job block the current routine until it free a space.
func (wp *Workerful) PushJob(job Job) {
	if !wp.canPush() {
		return
	}
	wp.jobQueue <- job
}

// PushJobAsync is an helper method that add a Job to the queue whithout blocking.
// With a buffered chan queue (queue_size != 0),
// when it is full wp.JobQueue <- job does not block the current routine.
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
// With a buffered chan queue (queue_size != 0),
// when it is full wp.JobQueue <- job block the current routine until it free a space.
func (wp *Workerful) PushFunc(job SimpleJob) {
	if !wp.canPush() {
		return
	}
	wp.jobQueue <- job
}

// PushFuncAsync is an helper method that add a func() to the queue whithout blocking.
// With a buffered chan queue (queue_size != 0),
// when it is full wp.JobQueue <- job does not block the current routine.
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
