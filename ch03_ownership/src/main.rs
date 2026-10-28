//! 第 03 章：所有权、借用与生命周期
//! 运行：cargo run -p ch03_ownership

fn main() {
    // 1. 移动（move）：String 的所有权从 s1 转给 s2
    let s1 = String::from("hello");
    let s2 = s1;
    // println!("{s1}"); // error[E0382]: borrow of moved value: `s1`
    println!("s2 = {s2}");

    // 2. 需要两份数据时显式 clone（堆数据被复制，有开销）
    let s3 = s2.clone();
    println!("s3 = {s3}, s2 仍然可用 = {s2}");

    // 3. 实现 Copy 的类型（如 i32）赋值时复制而不是移动
    let n1 = 5;
    let n2 = n1;
    println!("n1 = {n1}, n2 = {n2}");

    // 4. 借用：传 &T 给函数，不转移所有权
    let len = calc_len(&s3);
    println!("s3 = {s3}, 长度 = {len}");

    // 5. 可变借用：同一作用域内，可变借用是排他的
    let mut s4 = String::from("hello");
    append_world(&mut s4);
    println!("s4 = {s4}");

    // let r1 = &mut s4;
    // let r2 = &mut s4; // error[E0499]: cannot borrow `s4` as mutable more than once
    // println!("{r1} {r2}");

    // 6. 切片是「借用一段连续内存」的视图，没有复制
    let first = first_word(&s4);
    println!("第一个单词: {first}");

    // 7. 生命周期标注：告诉编译器返回的引用来自哪个入参
    let longer = longest("短", "一个更长的字符串");
    println!("更长的字符串: {longer}");

    // 8. 函数可以把所有权拿走进再还回来
    let s5 = takes_and_gives_back(s4);
    println!("s5 = {s5}");
}

fn calc_len(s: &str) -> usize {
    s.len()
}

fn append_world(s: &mut String) {
    s.push_str(", world");
}

fn first_word(s: &str) -> &str {
    s.split_whitespace().next().unwrap_or("")
}

fn longest<'a>(a: &'a str, b: &'a str) -> &'a str {
    if a.len() >= b.len() { a } else { b }
}

fn takes_and_gives_back(s: String) -> String {
    s
}

// 返回局部变量的引用会被拒绝：
// fn dangle() -> &String {
//     &String::from("x") // error[E0106]: missing lifetime specifier / borrowed value does not live long enough
// }
