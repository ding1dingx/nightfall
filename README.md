# nightfall

Go协程并发控制，降低CPU和内存负载

### env

```shell
goos: darwin
goarch: amd64
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
```

### goworker

```go
func main() {
    ctx := context.Background()
    
    pool := gopool.New(5000)
    for i := 0; i < 100000000; i++ {
        i := i
        pool.Go(ctx, func(ctx context.Context) {
            time.Sleep(time.Second)
            fmt.Println("Hello World:", i)
        })
    }
    
    <-ctx.Done()
}
```

- CPU

![goworker_cpu.png](example/goworker_cpu.png)

- 内存

![goworker_mem.png](example/goworker_mem.png)

### ants

```go
func main() {
    ctx := context.Background()
    
    pool, _ := ants.NewPool(5000)
    for i := 0; i < 100000000; i++ {
        i := i
        pool.Submit(func() {
            time.Sleep(time.Second)
            fmt.Println("Hello World:", i)
        })
    }
    
    <-ctx.Done()
}
```

- CPU

![ants_cpu.png](example/ants_cpu.png)

- 内存

![ants_mem.png](example/ants_mem.png)
