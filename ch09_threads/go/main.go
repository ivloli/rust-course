// 第 09 章的 Go 平行版本。
// 每个例子都对应 ch09_threads/src/main.rs 的 Rust 写法，
// 注释从 Go 视角解释两者的差异。
package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	// 1. thread::spawn + join ↔ goroutine + channel。
	// Rust 的 join() 既等待线程结束，又能拿到返回值；Go 的 goroutine 没有返回值，
	// 通常用一个 channel 把结果传回主 goroutine（相当于 join 的返回值）。
	done := make(chan int)
	go func() {
		for i := 1; i <= 3; i++ {
			fmt.Printf("子线程: %d\n", i)
			time.Sleep(10 * time.Millisecond) // 与 Rust 例子里的 sleep 对齐
		}
		done <- 42
	}()
	result := <-done // 阻塞等待，等价于 handle.join().unwrap()
	fmt.Printf("子线程返回值 = %d\n", result)

	// 2. move 闭包 ↔ Go 直接按引用捕获变量。
	// Rust 用 move 把 data 的所有权交给新线程，主线程之后不能再访问；
	// Go 的 slice 是引用语义，没有所有权转移，只能靠约定不去并发修改。
	// 求和结果同样用 channel 传回，模拟 join 的返回值。
	data := []int{1, 2, 3, 4}
	sumCh := make(chan int)
	go func() {
		sum := 0
		for _, value := range data {
			sum += value
		}
		sumCh <- sum
	}()
	fmt.Printf("data 求和 = %d\n", <-sumCh)
	// Rust 里 data 已被 move，下面这行无法编译；Go 仍然可以访问。
	// fmt.Println(data)

	// 3. Arc<Mutex<T>> ↔ 共享指针 + sync.Mutex。
	// Rust 需要 Arc 提供跨线程共享所有权；Go 的 goroutine 直接捕获 counter 的地址，
	// 由 sync.Mutex 保证互斥。区别是：数据竞争只在运行时由 -race 检测。
	counter := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1_000 {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}
	wg.Wait() // 对应 Rust 里逐个 handle.join()
	fmt.Printf("counter = %d\n", counter)

	// 4. mpsc channel ↔ Go 内建 channel。
	// 两者都是多生产者、单消费者。Go 没有单独的发送端/接收端类型，
	// 关闭 channel 用 close，而且只能由发送方关闭。
	// Rust 的 drop(tx) 在这里对应：等所有 worker 结束后 close(messages)，
	// 这样 for range messages 才会退出。
	messages := make(chan string)
	var workers sync.WaitGroup
	for worker := 0; worker < 3; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			time.Sleep(time.Duration(worker) * 10 * time.Millisecond)
			messages <- fmt.Sprintf("worker %d 完成", worker)
		}()
	}
	go func() {
		workers.Wait()
		close(messages)
	}()
	for message := range messages {
		fmt.Printf("收到: %s\n", message)
	}

	// 5. thread::scope ↔ goroutine + WaitGroup。
	// Go 没有「借用检查的 scoped thread」：goroutine 永远可以捕获外层变量，
	// 编译器不检查它是否比变量活得更久；正确性靠 WaitGroup 等同步手段保证。
	// 两个 goroutine 的输出顺序不确定，Rust 的 scope.spawn 同理。
	shared := []int{}
	var scoped sync.WaitGroup
	scoped.Add(2)
	go func() {
		defer scoped.Done()
		fmt.Printf("scoped 线程读取 shared.len() = %d\n", len(shared))
	}()
	go func() {
		defer scoped.Done()
		fmt.Println("第二个 scoped 线程")
	}()
	scoped.Wait() // 相当于 thread::scope 结束时的隐式 join
	shared = append(shared, 1)
	fmt.Printf("scope 结束后可以修改 shared = %s\n", formatInts(shared))
}

// formatInts 让输出和 Rust 的 {:?} 对齐：[1, 2, 3]。
func formatInts(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
