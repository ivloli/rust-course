//! 第 11 章：TCP 与 HTTP 网络编程
//! 运行：cargo run -p ch11_networking
//! 可选（需要外网）：cargo run -p ch11_networking -- --http

use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::{TcpListener, TcpStream};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. 启动一个监听随机端口的 echo 服务器
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let addr = listener.local_addr()?;
    println!("echo 服务器监听 {addr}");

    tokio::spawn(async move {
        loop {
            match listener.accept().await {
                Ok((socket, peer)) => {
                    println!("新连接来自 {peer}");
                    tokio::spawn(async move {
                        if let Err(error) = handle_connection(socket).await {
                            eprintln!("连接 {peer} 出错: {error}");
                        }
                    });
                }
                Err(error) => eprintln!("accept 失败: {error}"),
            }
        }
    });

    // 2. 并发发起 3 个客户端连接
    let mut clients = Vec::new();
    for client_id in 0..3 {
        clients.push(tokio::spawn(async move {
            echo_client(addr, client_id).await.unwrap();
        }));
    }
    for client in clients {
        client.await.unwrap();
    }

    // 3. 可选：HTTP 客户端（需要网络）
    if std::env::args().any(|arg| arg == "--http") {
        let response = reqwest::get("https://api.github.com/zen").await?;
        println!("HTTP 状态: {}", response.status());
        println!("响应: {}", response.text().await?.trim());
    } else {
        println!("提示：加 --http 参数可以演示 reqwest 请求外部 HTTP 服务");
    }

    Ok(())
}

async fn handle_connection(socket: TcpStream) -> std::io::Result<()> {
    let (reader, mut writer) = socket.into_split();
    let mut reader = BufReader::new(reader);
    let mut line = String::new();

    while reader.read_line(&mut line).await? > 0 {
        print!("服务器收到: {line}");
        writer.write_all(line.as_bytes()).await?;
        writer.flush().await?;
        line.clear();
    }
    println!("客户端断开");
    Ok(())
}

async fn echo_client(addr: std::net::SocketAddr, client_id: u32) -> std::io::Result<()> {
    let stream = TcpStream::connect(addr).await?;
    let (reader, mut writer) = stream.into_split();
    let mut reader = BufReader::new(reader);

    for round in 0..2 {
        let message = format!("client {client_id} message {round}\n");
        writer.write_all(message.as_bytes()).await?;

        let mut echo = String::new();
        reader.read_line(&mut echo).await?;
        print!("客户端 {client_id} 收到回声: {echo}");
    }
    Ok(())
}
