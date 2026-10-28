// 第 10 章的 Go 平行版本（只用标准库）。
// 对应 ch10_async_tokio/src/main.rs：用 goroutine + channel/select + context
// 替代 async/await + Tokio。
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func main() {
	// 1. async fn 返回惰性 Future，不 await 就不执行；Go 没有 Future。
	// 函数值只是「还没调用」；真正的 go sayHello() 会立即开始执行。
	// 这里用函数值模拟「已创建、未执行」，调用时才打印，保持与 Rust 相同的行序。
	future := sayHello
	fmt.Println("Future 已创建，但尚未执行")
	future()

	// 2. tokio::join! ↔ 两个 goroutine + WaitGroup。
	// join! 在同一个任务里并发轮询两个 Future；Go 直接开两个 goroutine，
	// 各自把结果写进 slice 的不同下标，最后 WaitGroup 等待全部完成。
	started := time.Now()
	results := make([]string, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0] = work(context.Background(), "A", 200)
	}()
	go func() {
		defer wg.Done()
		results[1] = work(context.Background(), "B", 300)
	}()
	wg.Wait()
	fmt.Printf("join! 完成: %s / %s，耗时 %v\n",
		results[0], results[1], time.Since(started).Round(time.Millisecond))

	// 3. 顺序 await ↔ 顺序调用（阻塞当前 goroutine）。总耗时约为两者之和。
	started = time.Now()
	work(context.Background(), "C", 100)
	work(context.Background(), "D", 100)
	fmt.Printf("顺序执行耗时 %v\n", time.Since(started).Round(time.Millisecond))

	// 4. tokio::spawn ↔ go 关键字。
	// Go 没有 JoinHandle，用 channel 取回结果；任务从一开始就独立运行。
	spawned := make(chan int, 1)
	go func() { spawned <- 6 * 7 }()
	fmt.Printf("spawn 结果 = %d\n", <-spawned)

	// 5. tokio::time::timeout ↔ context.WithTimeout。
	// Rust 超时会 drop 内部 Future，取消点自动执行清理；
	// Go 的取消是协作式的：workCtx 必须自己 select ctx.Done()，否则会继续跑完。
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if value, err := workCtx(ctx, "慢任务", 500); err != nil {
		fmt.Println("慢任务 100ms 超时，已取消")
	} else {
		fmt.Printf("慢任务完成: %s\n", value)
	}

	// 6. tokio::sync::mpsc ↔ 带缓冲的 Go channel。
	// Go 的 channel 内建背压：缓冲满时发送方阻塞，和 send().await 一样。
	// 关闭 channel 用 close，for range 会在消费完剩余值后退出。
	values := make(chan int, 8)
	go func() {
		defer close(values)
		for i := 1; i <= 5; i++ {
			values <- i
			time.Sleep(20 * time.Millisecond)
		}
	}()
	for value := range values {
		fmt.Printf("%d ", value)
	}
	fmt.Println()

	// 7. tokio::sync::Mutex ↔ sync.Mutex。
	// Go 没有 async mutex：goroutine 阻塞的代价很低，运行时会把同一个 M 上的
	// 其他 G 调度走，所以不需要「可跨 await 的锁」，也没有取消等待者的语义。
	// 和 Rust 例子一样，故意在持锁期间 sleep，演示临界区过大的代价。
	shared := 0
	var mu sync.Mutex
	var handles sync.WaitGroup
	for range 5 {
		handles.Add(1)
		go func() {
			defer handles.Done()
			mu.Lock()
			defer mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			shared++
		}()
	}
	handles.Wait() // 对应 Rust 里逐个 handle.await
	fmt.Printf("shared = %d\n", shared)
}

func sayHello() {
	fmt.Println("hello from async fn")
}

// work 对应 Rust 的 async fn work(name, millis) -> String。
func work(ctx context.Context, name string, millis int) string {
	value, _ := workCtx(ctx, name, millis)
	return value
}

// workCtx 用 select 同时等待定时器和 context 取消，对应 tokio::time::sleep +
// tokio::time::timeout 的组合。取消是协作式的，只有 select 到 ctx.Done() 才返回。
func workCtx(ctx context.Context, name string, millis int) (string, error) {
	timer := time.NewTimer(time.Duration(millis) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return name + " 完成", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
