//! 第 06 章：错误处理
//! 运行：cargo run -p ch06_errors

use std::fmt;
use std::num::ParseIntError;

#[derive(Debug)]
enum AppError {
    EmptyInput,
    NotANumber(ParseIntError),
    OutOfRange(i64),
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::EmptyInput => write!(f, "输入不能为空"),
            AppError::NotANumber(e) => write!(f, "不是合法整数: {e}"),
            AppError::OutOfRange(n) => write!(f, "数值 {n} 超出 1..=100 范围"),
        }
    }
}

impl std::error::Error for AppError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            AppError::NotANumber(e) => Some(e),
            _ => None,
        }
    }
}

// 有了 From，? 才能自动把 ParseIntError 转成 AppError
impl From<ParseIntError> for AppError {
    fn from(error: ParseIntError) -> Self {
        AppError::NotANumber(error)
    }
}

fn parse_score(input: &str) -> Result<u32, AppError> {
    if input.trim().is_empty() {
        return Err(AppError::EmptyInput);
    }
    let value: i64 = input.trim().parse()?; // ? 触发 From 转换
    if !(1..=100).contains(&value) {
        return Err(AppError::OutOfRange(value));
    }
    Ok(value as u32)
}

fn main() {
    // 1. Option：可能不存在的值
    let numbers = [1, 2, 3];
    println!("get(1) = {:?}", numbers.get(1));
    println!("get(9) = {:?}", numbers.get(9));

    let first_even = numbers.iter().find(|&&n| n % 2 == 0).copied();
    let scaled = first_even.map(|n| n * 10).unwrap_or(0);
    println!("first_even * 10 = {scaled}");

    // 2. Result + ? 运算符
    for input in ["42", "", "abc", "150", " 7 "] {
        match parse_score(input) {
            Ok(score) => println!("{input:?} -> {score}"),
            Err(error) => println!("{input:?} -> 错误: {error}"),
        }
    }

    // 3. unwrap 系列：什么时候可以 panic
    let parsed: i32 = "10".parse().unwrap(); // 确定不会失败时可以，但生产代码要谨慎
    let fallback: i32 = "x".parse().unwrap_or(-1);
    let computed: i32 = "8".parse::<i32>().map(|n| n * 2).unwrap_or_default();
    println!("parsed = {parsed}, fallback = {fallback}, computed = {computed}");
    // None::<i32>.unwrap(); // panic

    // 4. ? 与 Box<dyn Error>：应用层常见的简化写法
    match read_text_len("Cargo.toml") {
        Ok(len) => println!("Cargo.toml 长度 = {len} 字节"),
        Err(error) => println!("读取失败: {error}"),
    }

    // 5. main 也可以返回 Result，错误会被打印并设置退出码
    match run() {
        Ok(()) => {}
        Err(error) => eprintln!("run() 失败: {error}"),
    }
}

fn read_text_len(path: &str) -> Result<usize, Box<dyn std::error::Error>> {
    let content = std::fs::read_to_string(path)?;
    Ok(content.len())
}

fn run() -> Result<(), AppError> {
    let score = parse_score("100")?;
    println!("run() 解析结果 = {score}");
    Ok(())
}
