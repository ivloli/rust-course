//! 第 08 章：模块、测试与工程结构
//! 运行：cargo run -p ch08_modules
//! 测试：cargo test -p ch08_modules

use ch08_modules::{gcd, mean, word_count};

fn main() {
    println!("gcd(48, 18) = {}", gcd(48, 18));
    println!("mean([2.0, 4.0]) = {:?}", mean(&[2.0, 4.0]));
    println!("word_count(\"a b c\") = {}", word_count("a b c"));
    println!(
        "嵌套路径也能访问: {:?}",
        ch08_modules::math::stats::mean(&[1.0, 2.0, 3.0])
    );
}
