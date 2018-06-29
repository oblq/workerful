package workerful

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"gopkg.in/yaml.v2"
)

// Config defines the config for workerful.
type Config struct {
	QueueSize int `yaml:"queue_size"`
	Workers   int `yaml:"workers"`
}

// Job is a job interface, it only has to have an Execute() func
type Job interface {
	DoTheJob()
}

// JobQueue is a job queue in which you can append jobs.
type jobQueue chan interface{}

// Workerful is the workerful instance type.
type Workerful struct {
	Config   *Config
	jobQueue jobQueue
}

var once sync.Once
var instance *Workerful

// InitShared initialize the shared singleton,
// workerful also accept no configPath nor Config, the default values will be loaded.
// It can be used as a goms PackageInitFunc to be loaded automatically.
func InitShared(configPath string, config interface{}) error {
	once.Do(func() {
		if config, ok := config.(*Config); ok {
			instance = NewWorkerful("", config)
			return
		}
		if config, ok := config.(Config); ok {
			instance = NewWorkerful("", &config)
			return
		}
		instance = NewWorkerful(configPath, nil)
	})
	return nil
}

// Shared returns the workerful singleton.
//  workerful.Shared().PushAsync(func() { println("test") })
func Shared() *Workerful {
	return instance
}

// NewWorkerful creates and returns a new workerful instance, and starts a dispatcher to process the queue.
// Using 0 for maxQueueSize and/or maxWorkers will load the default values,
// 10000 and the number of CPU cores respectively
// close(workerful.jobQueue) when finished.
func NewWorkerful(configPath string, config *Config) *Workerful {

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

	if config.QueueSize == 0 {
		config.QueueSize = 10000
	}

	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
	}

	wp := &Workerful{
		Config:   config,
		jobQueue: make(jobQueue, config.QueueSize),
	}

	// Create workers
	wg := &sync.WaitGroup{}
	wg.Add(config.Workers)
	for i := 1; i <= config.Workers; i++ {
		go newWorker(wp.jobQueue, wg)
	}

	// Wait for workers to complete
	// Wait() blocks until the WaitGroup counter is zero and the channel closed
	// we don't need it
	go func() {
		wg.Wait()
		println("jobQueue closed...")
	}()

	return wp
}

func newWorker(jq jobQueue, wg *sync.WaitGroup) {
	defer wg.Done()
	// range over channels only stop after the channel has been closed
	// wg.Done() then is never called until jobQueue is closed: close(jobQueue)
	// so we have 'maxWorkers' routines executing jobs in chunks forever
	for job := range jq {
		switch job.(type) {
		case Job:
			job.(Job).DoTheJob()
		case func():
			job.(func())()
		default:
			log.Println("[workerful] Push() func only accept `Job` (see workerful.Job interface) and `func()` types")
			continue
		}
	}
}

// PushJob is an helper method that add a Job to the queue whithout blocking.
// When the jobQueue is full wp.JobQueue <- job block the current routine until it finish the queued jobs
func (wp Workerful) PushJob(job Job) {
	go func() {
		wp.jobQueue <- job
	}()
}

// PushFunc is an helper method that add a func() to the queue whithout blocking.
// When the jobQueue is full wp.JobQueue <- job block the current routine until it finish the queued jobs
func (wp Workerful) PushFunc(job func()) {
	go func() {
		wp.jobQueue <- job
	}()
}

// Close close the jobQueue
func (wp Workerful) Close() {
	close(wp.jobQueue)
}
