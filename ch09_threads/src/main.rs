//! 第 09 章：线程与共享状态
//! 运行：cargo run -p ch09_threads

use std::sync::{Arc, Mutex, mpsc};
use std::thread;
use std::time::Duration;

fn main() {
    // 1. spawn + join：join 会等待线程结束，并拿到返回值
    let handle = thread::spawn(|| {
        for i in 1..=3 {
            println!("子线程: {i}");
            thread::sleep(Duration::from_millis(10));
        }
        42
    });
    let result = handle.join().expect("子线程 panic 了");
    println!("子线程返回值 = {result}");

    // 2. move 闭包：把数据的所有权交给新线程
    let data: Vec<i32> = (1..=4).collect();
    let sum_handle = thread::spawn(move || data.iter().sum::<i32>());
    println!("data 求和 = {}", sum_handle.join().unwrap());
    // println!("{data:?}"); // data 已被移动到线程里

    // 3. Arc<Mutex<T>>：多线程共享并修改同一份数据
    let counter = Arc::new(Mutex::new(0));
    let mut handles = Vec::new();
    for _ in 0..10 {
        let counter = Arc::clone(&counter);
        handles.push(thread::spawn(move || {
            for _ in 0..1_000 {
                let mut value = counter.lock().unwrap();
                *value += 1;
            }
        }));
    }
    for handle in handles {
        handle.join().unwrap();
    }
    println!("counter = {}", *counter.lock().unwrap());

    // 4. mpsc channel：多生产者、单消费者
    let (tx, rx) = mpsc::channel();
    for worker in 0..3 {
        let tx = tx.clone();
        thread::spawn(move || {
            thread::sleep(Duration::from_millis(worker * 10));
            tx.send(format!("worker {worker} 完成")).unwrap();
        });
    }
    drop(tx); // 关闭原始发送端，所有 worker 结束后 rx 循环才会退出
    for message in rx {
        println!("收到: {message}");
    }

    // 5. scoped threads：可以借用栈上的局部变量
    let mut shared = vec![];
    thread::scope(|scope| {
        scope.spawn(|| {
            println!("scoped 线程读取 shared.len() = {}", shared.len());
        });
        scope.spawn(|| {
            println!("第二个 scoped 线程");
        });
    });
    shared.push(1);
    println!("scope 结束后可以修改 shared = {shared:?}");
}
