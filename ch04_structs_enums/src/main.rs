//! 第 04 章：结构体、枚举与 Option
//! 运行：cargo run -p ch04_structs_enums

#[derive(Debug, Clone)]
struct User {
    name: String,
    age: u32,
    active: bool,
}

impl User {
    fn new(name: &str, age: u32) -> Self {
        Self {
            name: name.to_string(),
            age,
            active: true,
        }
    }

    fn birthday(&mut self) {
        self.age += 1;
    }

    fn is_adult(&self) -> bool {
        self.age >= 18
    }

    fn greeting(&self) -> String {
        let status = if self.active { "在职" } else { "离职" };
        format!("你好，我叫{}，{} 岁，{}", self.name, self.age, status)
    }
}

#[derive(Debug, Clone, Copy)]
enum Shape {
    Circle { radius: f64 },
    Rect { width: f64, height: f64 },
    Triangle { a: f64, b: f64, c: f64 },
}

impl Shape {
    fn area(&self) -> f64 {
        match self {
            Shape::Circle { radius } => std::f64::consts::PI * radius * radius,
            Shape::Rect { width, height } => width * height,
            Shape::Triangle { a, b, c } => {
                let s = (a + b + c) / 2.0;
                (s * (s - a) * (s - b) * (s - c)).sqrt()
            }
        }
    }

    fn name(&self) -> &'static str {
        match self {
            Shape::Circle { .. } => "圆形",
            Shape::Rect { .. } => "矩形",
            Shape::Triangle { .. } => "三角形",
        }
    }
}

enum Command {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
}

fn main() {
    // 1. 结构体 + 方法
    let mut user = User::new("小明", 17);
    println!("{}", user.greeting());
    println!("成年了吗？{}", user.is_adult());
    user.birthday(); // 需要 &mut self
    println!("生日后: {:?}", user);

    // 2. 枚举可以携带不同形状的数据
    let shapes = [
        Shape::Circle { radius: 1.0 },
        Shape::Rect {
            width: 2.0,
            height: 3.0,
        },
        Shape::Triangle {
            a: 3.0,
            b: 4.0,
            c: 5.0,
        },
    ];
    for shape in &shapes {
        println!("{:>6} 面积 = {:.2}", shape.name(), shape.area());
    }
    let total: f64 = shapes.iter().map(Shape::area).sum();
    println!("总面积 = {total:.2}");

    // 3. Option<T>：Rust 没有 null，用类型表达「可能没有」
    let users = [User::new("Alice", 20), User::new("Bob", 30)];
    let found: Option<&User> = users.iter().find(|u| u.name == "Bob");
    match found {
        Some(u) => println!("找到 {}，{} 岁", u.name, u.age),
        None => println!("没找到"),
    }

    // 常用组合子：map / unwrap_or
    let bob_age = found.map(|u| u.age).unwrap_or(0);
    println!("bob_age = {bob_age}");

    // if let：只关心 Some 分支
    if let Some(u) = users.iter().find(|u| u.name == "Carol") {
        println!("Carol 存在: {}", u.greeting());
    } else {
        println!("Carol 不存在");
    }

    // while let：栈非空就一直弹出
    let mut stack = vec![1, 2, 3];
    while let Some(top) = stack.pop() {
        print!("{top} ");
    }
    println!();

    // 4. 枚举解构 + match
    let commands = [
        Command::Move { x: 10, y: 20 },
        Command::Write("保存文件".to_string()),
        Command::Quit,
    ];
    for command in commands {
        match command {
            Command::Quit => println!("退出"),
            Command::Move { x, y } => println!("移动到 ({x}, {y})"),
            Command::Write(text) => println!("写入: {text}"),
        }
    }
}
