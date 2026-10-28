//! 第 02 章：函数、控制流与模式匹配
//! 运行：cargo run -p ch02_control_flow

fn main() {
    println!("sign(-3) = {}", sign(-3));
    println!("gcd(48, 18) = {}", gcd(48, 18));
    println!("fib(10) = {}", fib(10));

    // if 是表达式，可以直接给变量赋值
    let age = 20;
    let category = if age < 18 { "未成年" } else { "成年" };
    println!("{age} 岁：{category}");

    // loop 可以用 break 返回值
    let mut counter = 0;
    let doubled = loop {
        counter += 1;
        if counter == 5 {
            break counter * 2;
        }
    };
    println!("counter = {counter}, doubled = {doubled}");

    // while：条件为真时循环
    let mut countdown = 3;
    while countdown > 0 {
        print!("{countdown} ");
        countdown -= 1;
    }
    println!("发射！");

    // for 遍历范围、数组、迭代器
    for i in 1..=3 {
        print!("for: {i} ");
    }
    println!();

    for (index, score) in [88, 92, 60].iter().enumerate() {
        println!("第 {index} 位同学: {score}");
    }

    // 带标签的循环：从内层直接跳出外层
    'outer: for i in 0..5 {
        for j in 0..5 {
            if i * j == 6 {
                println!("找到 i={i}, j={j}");
                break 'outer;
            }
        }
    }

    // match 要求穷尽所有情况，编译器会检查
    let temp = -5;
    let advice = match temp {
        n if n < 0 => "穿厚点",
        0 => "刚刚好",
        1..=20 => "舒适",
        _ => "有点热",
    };
    println!("温度 {temp}：{advice}");

    // match 配合元组做组合判断
    for n in 1..=15 {
        println!("{n} -> {}", fizzbuzz(n));
    }

    // match 可以解构元组并绑定变量
    let command = ("move", 10);
    match command {
        ("move", step) => println!("向 step={step} 移动"),
        ("jump", height) => println!("跳 {height} 米"),
        (other, _) => println!("未知命令 {other}"),
    }
}

fn sign(x: i32) -> &'static str {
    if x < 0 {
        "负数"
    } else if x == 0 {
        "零"
    } else {
        "正数"
    }
}

fn gcd(mut a: u64, mut b: u64) -> u64 {
    while b != 0 {
        let remainder = a % b;
        a = b;
        b = remainder;
    }
    a
}

fn fib(n: u32) -> u64 {
    let (mut a, mut b) = (0u64, 1u64);
    for _ in 0..n {
        let next = a + b;
        a = b;
        b = next;
    }
    a
}

fn fizzbuzz(n: u32) -> String {
    match (n % 3, n % 5) {
        (0, 0) => "FizzBuzz".to_string(),
        (0, _) => "Fizz".to_string(),
        (_, 0) => "Buzz".to_string(),
        _ => n.to_string(),
    }
}
