//! 第 05 章：集合、迭代器与闭包
//! 运行：cargo run -p ch05_collections

use std::collections::HashMap;

fn main() {
    // 1. Vec：动态数组
    let mut numbers = vec![3, 1, 2];
    numbers.sort();
    numbers.push(5);
    println!("numbers = {numbers:?}");

    println!("get(10) = {:?}", numbers.get(10)); // 安全访问，返回 Option
    // numbers[10]; // 越界会 panic

    let doubled: Vec<i32> = numbers.iter().map(|n| n * 2).collect();
    println!("doubled = {doubled:?}");

    // 2. String 与 UTF-8
    let mut text = String::from("hello");
    text.push_str(" rust");
    text.push('!');
    println!("{text} | 大写: {}", text.to_uppercase());
    println!("字符数: {}", "中文abc".chars().count()); // 5
    println!("字节数: {}", "中文abc".len()); // 9

    // 3. HashMap：统计词频
    let sentence = "the quick brown fox jumps over the lazy dog the fox";
    let mut counts: HashMap<&str, u32> = HashMap::new();
    for word in sentence.split_whitespace() {
        *counts.entry(word).or_insert(0) += 1;
    }

    let mut ranked: Vec<(&str, u32)> = counts.into_iter().collect();
    ranked.sort_by(|a, b| b.1.cmp(&a.1).then(a.0.cmp(b.0)));
    for (word, count) in ranked.iter().take(3) {
        println!("{word}: {count}");
    }

    // 4. 迭代器：惰性求值，调用 collect / sum 等消费者时才真正执行
    let evens_squared: Vec<i32> = (1..=10).filter(|n| n % 2 == 0).map(|n| n * n).collect();
    println!("evens_squared = {evens_squared:?}");

    let sum: i32 = (1..=100).sum();
    let product: u64 = (1..=10).product();
    println!("1..=100 求和 = {sum}, 1..=10 求积 = {product}");

    let max = vec![3, 9, 2].into_iter().max();
    println!("max = {:?}", max.unwrap());

    let names = ["alice", "bob"];
    let scores = [90, 82];
    let joined = names
        .iter()
        .map(|name| name.to_uppercase())
        .collect::<Vec<_>>()
        .join(", ");
    println!("joined = {joined}");

    for (index, (name, score)) in names.iter().zip(scores.iter()).enumerate() {
        println!("#{index}: {name} -> {score}");
    }

    // fold：把一串值累积成一个结果（这里拼成 CSV）
    let csv = scores.iter().fold(String::new(), |mut acc, score| {
        if !acc.is_empty() {
            acc.push(',');
        }
        acc.push_str(&score.to_string());
        acc
    });
    println!("csv = {csv}");

    // 5. 闭包：可以捕获环境变量
    let threshold = 5;
    let above: Vec<i32> = (1..=10).filter(|n| *n > threshold).collect();
    println!("大于 {threshold} 的数: {above:?}");

    let mut calls = 0;
    (1..=3).for_each(|_| calls += 1); // FnMut 闭包
    println!("calls = {calls}");
}
