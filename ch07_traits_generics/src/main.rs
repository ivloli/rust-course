//! 第 07 章：Trait、泛型与多态
//! 运行：cargo run -p ch07_traits_generics

use std::fmt;

trait Summary {
    fn summarize_author(&self) -> String;

    // 默认方法：实现者可以不覆盖
    fn summarize(&self) -> String {
        format!("（阅读更多来自 {} 的内容…）", self.summarize_author())
    }
}

struct Article {
    title: String,
    author: String,
}

struct Tweet {
    username: String,
    body: String,
}

impl Summary for Article {
    fn summarize_author(&self) -> String {
        self.author.clone()
    }

    fn summarize(&self) -> String {
        format!("{} — {}", self.title, self.summarize_author())
    }
}

impl Summary for Tweet {
    fn summarize_author(&self) -> String {
        format!("@{}", self.username)
    }

    fn summarize(&self) -> String {
        format!("{}: {}", self.summarize_author(), self.body)
    }
}

// 让 Tweet 满足 notify_with_display 的 Display 约束
impl fmt::Display for Tweet {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.summarize())
    }
}

// 1. impl Trait 参数：静态分发
fn notify(item: &impl Summary) -> String {
    format!("Breaking! {}", item.summarize())
}

// 2. 泛型 + 多个 trait bound
fn notify_with_display<T: Summary + fmt::Display>(item: &T) -> String {
    format!("Breaking! {item} | {}", item.summarize())
}

// 3. where 子句：约束复杂时更清晰
fn summarize_all<T>(items: &[T]) -> Vec<String>
where
    T: Summary,
{
    items.iter().map(|item| item.summarize()).collect()
}

// 4. 返回 impl Trait：调用方只看到接口，看不到具体类型
fn make_tweet() -> impl Summary {
    Tweet {
        username: "rustlang".into(),
        body: "Rust 1.99 已发布".into(),
    }
}

// 5. 泛型函数：T 必须支持比较和复制
fn largest<T: PartialOrd + Copy>(list: &[T]) -> T {
    let mut largest = list[0];
    for &item in list {
        if item > largest {
            largest = item;
        }
    }
    largest
}

// 6. 自定义 trait 做动态分发
trait Area {
    fn area(&self) -> f64;
}

struct Square(f64);
struct Circle(f64);

impl Area for Square {
    fn area(&self) -> f64 {
        self.0 * self.0
    }
}

impl Area for Circle {
    fn area(&self) -> f64 {
        std::f64::consts::PI * self.0 * self.0
    }
}

// 7. 标准库 trait：From / Into / Display
#[derive(Debug, Clone, Copy)]
struct UserId(u64);

impl From<u32> for UserId {
    fn from(value: u32) -> Self {
        UserId(value as u64)
    }
}

impl fmt::Display for UserId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "user#{}", self.0)
    }
}

fn main() {
    let article = Article {
        title: "Rust 所有权入门".to_string(),
        author: "小明".to_string(),
    };
    let tweet = Tweet {
        username: "chuan".to_string(),
        body: "今天在学 trait".to_string(),
    };

    println!("{}", article.summarize());
    println!("{}", tweet.summarize());
    println!("{}", notify(&tweet));
    println!("{}", notify_with_display(&tweet));

    let summaries = summarize_all(&[make_tweet()]);
    println!("{summaries:?}");

    // 动态分发：一个 Vec 里放不同类型的值
    let items: Vec<Box<dyn Summary>> = vec![
        Box::new(Article {
            title: "动态分发".into(),
            author: "A".into(),
        }),
        Box::new(Tweet {
            username: "B".into(),
            body: "在运行时决定调用".into(),
        }),
    ];
    for item in &items {
        println!("dyn -> {}", item.summarize());
    }

    let shapes: Vec<Box<dyn Area>> = vec![Box::new(Square(2.0)), Box::new(Circle(1.0))];
    let total: f64 = shapes.iter().map(|shape| shape.area()).sum();
    println!("总面积 = {total:.2}");

    println!("largest(&[3, 9, 2]) = {}", largest(&[3, 9, 2]));
    println!("largest(&['a', 'z', 'm']) = {}", largest(&['a', 'z', 'm']));

    let id = UserId::from(42u32);
    let other: UserId = 7u32.into();
    println!("id = {id}, other = {other}");
}
