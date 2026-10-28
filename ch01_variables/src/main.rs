//! 第 01 章：变量、类型与可变性
//! 运行：cargo run -p ch01_variables

fn main() {
    // 1. 变量默认不可变。取消下面一行的注释会得到 error[E0384]
    let x = 5;
    println!("x = {x}");
    // x = 6;

    // 2. 需要修改时加 mut
    let mut y = 10;
    y += 1;
    println!("y = {y}");

    // 3. 隐藏（shadowing）：可以改变类型，旧变量被遮蔽
    let x = x + 1;
    let x = format!("现在 x 变成了字符串: {x}");
    println!("{x}");

    // 4. 类型标注与解析
    let guess: u32 = "42".parse().expect("输入不是数字");
    println!("guess = {guess}");

    // 5. 标量类型：整数、浮点、布尔、字符
    let integer: i64 = 1_000_000;
    let float: f64 = 2.5;
    let is_rust_fun = true;
    let letter: char = '中'; // char 是 4 字节的 Unicode 标量值
    println!("{integer} {float} {is_rust_fun} {letter}");

    // 6. 整数溢出：debug 构建会 panic，release 构建会回绕
    let max_u8 = u8::MAX;
    println!("u8::MAX = {max_u8}");
    // let overflow = max_u8 + 1; // debug 下 panic: attempt to add with overflow

    // 7. 复合类型：元组与数组
    let person: (&str, u32, bool) = ("小明", 28, true);
    let (name, age, active) = person;
    println!(
        "{name} {age} 岁，在职: {active}，也可以写 person.0 = {}",
        person.0
    );

    let scores: [i32; 5] = [90, 85, 77, 60, 99];
    let zeros = [0u8; 3];
    println!(
        "scores = {scores:?}, 长度 {}, 第一项 {}",
        scores.len(),
        scores[0]
    );
    println!("zeros = {zeros:?}");

    // 8. 字符串：&str 是借用的 UTF-8 视图，String 是可增长的堆内存
    let literal: &str = "字面量";
    let owned: String = String::from("堆上的 String");
    let combined = format!("{literal} + {owned}");
    println!("{combined}");

    // 9. 常量在编译期确定，必须有类型，命名约定全大写
    const MAX_POINTS: u32 = 100_000;
    println!("MAX_POINTS = {MAX_POINTS}");

    // 10. 显式转换不会隐式发生
    let big: i64 = 300;
    let small: i32 = big as i32;
    let truncated: u8 = 300u16 as u8; // 300 % 256 = 44
    println!("big = {big}, small = {small}, truncated = {truncated}");
}
