//! 第 10 章：异步编程与 Tokio
//! 运行：cargo run -p ch10_async_tokio

use std::sync::Arc;
use std::time::{Duration, Instant};

use tokio::sync::{Mutex, mpsc};

#[tokio::main]
async fn main() {
    // 1. async fn 返回 Future，不 await 就不会执行
    let future = say_hello();
    println!("Future 已创建，但尚未执行");
    future.await;

    // 2. join!：并发等待多个 Future
    let started = Instant::now();
    let (a, b) = tokio::join!(work("A", 200), work("B", 300));
    println!("join! 完成: {a} / {b}，耗时 {:?}", started.elapsed());

    // 3. 顺序执行对比：总耗时约为两者之和
    let started = Instant::now();
    work("C", 100).await;
    work("D", 100).await;
    println!("顺序执行耗时 {:?}", started.elapsed());

    // 4. spawn：把任务交给 Tokio 调度器，要求 'static + Send
    let handle = tokio::spawn(async move { 6 * 7 });
    println!("spawn 结果 = {}", handle.await.unwrap());

    // 5. timeout：超时会取消内部 Future
    match tokio::time::timeout(Duration::from_millis(100), work("慢任务", 500)).await {
        Ok(value) => println!("慢任务完成: {value}"),
        Err(_) => println!("慢任务 100ms 超时，已取消"),
    }

    // 6. mpsc：异步 channel
    let (tx, mut rx) = mpsc::channel::<u32>(8);
    tokio::spawn(async move {
        for i in 1..=5 {
            tx.send(i).await.unwrap();
            tokio::time::sleep(Duration::from_millis(20)).await;
        }
    });
    while let Some(value) = rx.recv().await {
        print!("{value} ");
    }
    println!();

    // 7. tokio::sync::Mutex：可以跨 await 持有
    let shared = Arc::new(Mutex::new(0));
    let mut handles = Vec::new();
    for _ in 0..5 {
        let shared = Arc::clone(&shared);
        handles.push(tokio::spawn(async move {
            let mut guard = shared.lock().await;
            tokio::time::sleep(Duration::from_millis(10)).await;
            *guard += 1;
        }));
    }
    for handle in handles {
        handle.await.unwrap();
    }
    println!("shared = {}", *shared.lock().await);
}

async fn say_hello() {
    println!("hello from async fn");
}

async fn work(name: &str, millis: u64) -> String {
    tokio::time::sleep(Duration::from_millis(millis)).await;
    format!("{name} 完成")
}
