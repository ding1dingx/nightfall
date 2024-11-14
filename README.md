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

### example_1

#### nightfall

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

##### cpu

![nightfall_cpu_1.png](example/nightfall_cpu_1.png)

##### mem

![nightfall_mem_1.png](example/nightfall_mem_1.png)

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

##### cpu

![ants_cpu_1.png](example/ants_cpu_1.png)

##### mem

![ants_mem_1.png](example/ants_mem_1.png)

### example_2

#### nightfall

```go
func main() {
    ctx := context.Background()
    
    pool := woker.NewPool(5000)
    for i := 0; i < 100; i++ {
        i := i
        pool.Go(ctx, func(ctx context.Context) {
            for j := 0; j < 1000000; j++ {
                j := j
                pool.Go(ctx, func(ctx context.Context) {
                    time.Sleep(time.Second)
                    fmt.Println("Index:", i, "-", j)
                })
            }
        })
    }
    
    <-ctx.Done()
}
```

##### cpu

![nightfall_cpu_2.png](example/nightfall_cpu_2.png)

##### mem

![nightfall_mem_2.png](example/nightfall_mem_2.png)

#### ants

```go
func main() {
    ctx := context.Background()
    
    pool, _ := ants.NewPool(5000)
    for i := 0; i < 100; i++ {
        i := i
        pool.Submit(func() {
            for j := 0; j < 1000000; j++ {
                j := j
                pool.Submit(func() {
                    time.Sleep(time.Second)
                    fmt.Println("Index:", i, "-", j)
                })
            }
        })
    }
    
    <-ctx.Done()
}
```

##### cpu

![ants_cpu_2.png](example/ants_cpu_2.png)

##### mem

![ants_mem_2.png](example/ants_mem_2.png)
