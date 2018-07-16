# Workerful [![GoDoc](https://godoc.org/github.com/oblq/workerful?status.svg)](https://godoc.org/github.com/oblq/workerful)

**Workerful** is a simple yet powerful worker pool.

Compatible with [sprbox](https://github.com/oblq/sprbox).

## Getting Started

#### Install

```sh
$ go get github.com/oblq/workerful
```

#### Init

It is possible to pass the config as a yaml file path, the workerful.Config struct or none of the above (the default values will be loaded).\
The config struct takes two parameters:

| Parameter   | Description  | Default
| :---        |     :---     |     :---
| queue_size    | Determine the size of the queue. Use 0 to have an unbuffered jobQueue channel. | `0` (unbuffered)
| workers | The number of parallel jobs to be executed. Using `0` will load the default value. | `runtime.NumCPU()`

Get a new instance (with unbuffered jobQueue):
```
    wp := workerful.New("", &workerful.Config{QueueSize: 0, Workers: 0})
```

...or grab the singleton (preloaded with the default values):

```
    workerful.Shared()
```

#### Push a job

You can push a simple `func() error` or a more complex custom Job (see `workerful.Job` interface).

Push a simple func():
```
    workerful.Shared().PushFunc(func() error { 
    	println("func executed...")
    	return nil
    })

    // Non blocking.
    workerful.Shared().PushFuncAsync(func() error { 
    	println("func executed...")
        return nil
    })
```

Push a custom job if you need to pass parameters or for more complicated stuff:
```
    // CustomJob implement the workerful.Job interface (F())
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
        
    // Default values.	
    wp := workerful.New("", nil)
    responses := make(chan int)

    wp.PushJob(CustomJob{responses, 1})
        
    // Non blocking.
    wp.PushJobAsync(CustomJob{responses, 2})
    
    count := 0
    for i := range responses {
        count += i
        if count == 2 {
            close(responses)
        }
    }

    wp.Stop()
```

#### Get status

```
    doneJobs, failedJobs, inQueueJobs := wp.Status()
```

#### Graceful shutdown (and restart)

```
    wp := workerful.New("./workerful.yml", nil)
    wp.Stop()
    wp.Start()
```

## Author

- [Marco Muratori](mailto:marcomrtr@gmail.com) 

## License

Workerful is available under the MIT license. See the [LICENSE](./LICENSE) file for more information.
