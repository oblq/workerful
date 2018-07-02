# Workerful [![GoDoc](https://godoc.org/github.com/oblq/goms?status.svg)](https://godoc.org/github.com/oblq/workerful)

**Workerful** is a simple yet powerful worker pool.

### Getting Started

### Installation

```sh
$ go get github.com/oblq/workerful
```

### Init

Two parameters are required, it is possible to pass the config as a yaml file path or the workerful.Config struct.

| Parameter   | Description  | Default
| :---        |     :---     |     :---
| queue_size    | Determine the size of the queue. Use 0 to have an unbuffered jobQueue channel. | 
| workers | The number of parallel jobs to be executed. Using `0` will load the default value. | `runtime.NumCPU()`

Get a new instance:
```go
    wp := workerful.NewWorkerful("", &workerful.Config{QueueSize: 0, Workers: 0})
```

...or init the singleton:

```go
    wp := workerful.InitShared("", &workerful.Config{QueueSize: 0, Workers: 0})
    // ...then grab the preloaded shared instance from anywhere:
    workerful.Shared()
```

### Push a job

You can push a simple `func() error` or a more complex custom Job (see `workerful.Job` interface).

Push a simple func():
```go
    wp.PushFunc(func() error { 
    	println("func executed...")
    	return nil
    })

    // Non blocking, useful with buffered channels.
    wp.PushFuncAsync(func() error { 
    	println("func executed...")
        return nil
    })
```

Push a custom job if you need to do pass parameters:
```go
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
        	
    responses := make(chan int)

    wp.PushJob(CustomJob{responses, 1})
        
    // Non blocking, useful with buffered channels.
    wp.PushJobAsync(CustomJob{responses, 1})
```

### Get status

```go
    doneJobs, failedJobs, inQueueJobs := wp.Status()
```

### Graceful shutdown (and restart)

```go
    wp.Stop()
    wp.Restart()
```

## Author

- [Marco Muratori](mailto:marcomrtr@gmail.com) 

## License

Workerful is available under the MIT license. See the [LICENSE](./LICENSE) file for more information.
