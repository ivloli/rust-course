// 第 11 章的 Go 平行版本（只用标准库）。
// 对应 ch11_networking/src/main.rs：用 net.Listen/Accept + 每连接一个 goroutine
// 替代 Tokio 的 TcpListener + tokio::spawn。
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const clientCount = 3

func main() {
	// 1. TcpListener::bind("127.0.0.1:0") ↔ net.Listen("tcp", "127.0.0.1:0")。
	// 端口 0 表示让内核分配随机端口；local_addr() ↔ listener.Addr()。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "监听失败: %v\n", err)
		os.Exit(1)
	}
	addr := listener.Addr().String()
	fmt.Printf("echo 服务器监听 %s\n", addr)

	// 2. accept 循环放进 goroutine；每个连接再开一个 goroutine。
	// 这正是 Go 的经典模型：goroutine-per-connection，代码是同步阻塞风格，
	// 但并发连接之间互不阻塞（Rust 靠 tokio::spawn 达到同样效果）。
	// handlerDone 只用于让演示在 3 个连接都处理完后整齐退出；
	// 监听 goroutine 本身可以一直阻塞在 Accept 上，main 返回时进程会退出。
	handlerDone := make(chan struct{}, clientCount)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "accept 失败: %v\n", err)
				return
			}
			peer := conn.RemoteAddr().String()
			fmt.Printf("新连接来自 %s\n", peer)
			go func() {
				handleConnection(conn, peer)
				handlerDone <- struct{}{}
			}()
		}
	}()

	// 3. 3 个并发客户端 ↔ goroutine + WaitGroup。
	// Rust 用 tokio::spawn 收集 JoinHandle 再逐个 await；
	// Go 用 sync.WaitGroup 等待全部完成。
	var clients sync.WaitGroup
	for clientID := 0; clientID < clientCount; clientID++ {
		clients.Add(1)
		go func() {
			defer clients.Done()
			if err := echoClient(addr, clientID); err != nil {
				fmt.Fprintf(os.Stderr, "客户端 %d 出错: %v\n", clientID, err)
			}
		}()
	}
	clients.Wait()
	for range clientCount {
		<-handlerDone // 等服务端也读完 EOF 并打印「客户端断开」
	}
	// 客户端与服务端都已结束，main 直接返回；监听 goroutine 随进程退出。

	// 4. reqwest::get ↔ net/http.Get（可选，需要外网）。
	if hasArg("--http") {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get("https://api.github.com/zen")
		if err != nil {
			fmt.Fprintf(os.Stderr, "HTTP 请求失败: %v\n", err)
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取响应失败: %v\n", err)
			return
		}
		// Rust 的 StatusCode Display 和 Go 的 resp.Status 都是「200 OK」形式。
		fmt.Printf("HTTP 状态: %s\n", response.Status)
		fmt.Printf("响应: %s\n", strings.TrimSpace(string(body)))
	} else {
		fmt.Println("提示：加 --http 参数可以演示 reqwest 请求外部 HTTP 服务")
	}
}

// handleConnection 对应 Rust 的 handle_connection：
// bufio.Scanner 按行读取，等价于 BufReader::read_line；每读一行就原样写回。
// TCP 是字节流，这里用换行作为消息边界（line framing）。
func handleConnection(conn net.Conn, peer string) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("服务器收到: %s\n", line)
		if _, err := fmt.Fprintf(conn, "%s\n", line); err != nil {
			fmt.Fprintf(os.Stderr, "连接 %s 出错: %v\n", peer, err)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "连接 %s 出错: %v\n", peer, err)
		return
	}
	fmt.Println("客户端断开")
}

// echoClient 对应 Rust 的 echo_client：两轮「发一行、读一行」。
// net.Dial 等价于 TcpStream::connect；bufio.Reader.ReadString 保留换行符，
// 所以打印格式和 Rust 的 read_line 一致。
func echoClient(addr string, clientID int) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for round := 0; round < 2; round++ {
		message := fmt.Sprintf("client %d message %d\n", clientID, round)
		if _, err := conn.Write([]byte(message)); err != nil {
			return err
		}
		echo, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		fmt.Printf("客户端 %d 收到回声: %s", clientID, echo)
	}
	return nil
}

func hasArg(target string) bool {
	for _, arg := range os.Args[1:] {
		if arg == target {
			return true
		}
	}
	return false
}
