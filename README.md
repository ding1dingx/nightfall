# nightfall

Go协程并发控制，降低CPU和内存负载

## 特点

1. 实现简单
2. 性能优秀
3. 任务支持context
4. 任务队列支持缓冲
5. 非阻塞模式下，任务缓冲到全局链表

## 与ants比较

```shell
goos: darwin
goarch: amd64
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
```

### nightfall

```go
func main() {
    ctx := context.Background()
    
    pool := woker.NewPool(5000)
    for i := 0; i < 100000000; i++ {
        i := i
        pool.Go(ctx, func(ctx context.Context) {
            time.Sleep(time.Second)
            fmt.Println("Index:", i)
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
            fmt.Println("Index:", i)
        })
    }
    
    <-ctx.Done()
}
```

- CPU

![ants_cpu.png](example/ants_cpu.png)

- 内存

![ants_mem.png](example/ants_mem.png)
