# Workerful [![GoDoc](https://godoc.org/github.com/oblq/goms?status.svg)](https://godoc.org/github.com/oblq/workerful)

**Workerful** is a simple yet powerful worker pool.

### Getting Started

### Installation

```sh
$ go get github.com/oblq/goms/workerful
```

### Init

Two parameters are required, it is possible to pass the config as a yaml file path or the workerful.Config struct.

| Parameter   | Description  | Default
| :---        |     :---     |     :---
| queue_size    | Determine the size of the queue. Using `0` will load the default value. | `10000`
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

You can push a simple `func()` or a more complex custom Job (see `workerful.Job` interface).
Both `PushFunc(job func())` and `PushJob(job Job)` funcs add jobs without blocking (in case you push more jobs than the queue_size). 

Push a simple func():
```go
    wp.PushFunc(func() { 
    	println("func executed...")
    })
```

Push a custom job:
```go
    // CustomJob implement the workerful.Job interface (DoTheJob())
    type CustomJob struct {
        Responses chan int
        ID        int
    }
    
    func (cj CustomJob) DoTheJob() {
        time.Sleep(time.Second)
        println("job", cj.ID, "executed...")
        cj.Responses <- 1
    }
        	
    responses := make(chan int, 0)

    wp.PushJob(CustomJob{responses, 1})
```

### Graceful shutdown

```go
    wp.Close()
```

## Author

- [Marco Muratori](mailto:marcomrtr@gmail.com) 

## License

Workerful is available under the MIT license. See the [LICENSE](./LICENSE) file for more information.
