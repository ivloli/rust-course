# 06 错误处理：Result 与自定义错误

```bash
cargo run -p ch06_errors
```

> Go 主栈读者：每个例子的 Go 平行写法见 [GO_COMPARISON.md](GO_COMPARISON.md)，可运行代码在 `go/main.go`（输出结构一致；`abc` 的解析错误原文不同，见文档）。

## 学习目标

- 区分「可恢复错误」`Result<T, E>` 与「不可恢复错误」`panic!`
- 理解 `?` 运算符的展开方式与 `From` 转换
- 为库/服务编写自定义错误类型（实现 `Display` + `Error` + `source`）
- 知道什么时候可以用 `unwrap`，什么时候绝对不行

## 代码导读（src/main.rs）

1. `Option` 用于「值可能不存在」，`Result` 用于「操作可能失败」。不要用 `Option` 表达错误原因。
2. `parse_score` 的返回类型明确列出所有失败方式；调用方必须处理。
3. `input.trim().parse()?` 等价于 `match ... { Ok(v) => v, Err(e) => return Err(e.into()) }`，`From` 实现让错误类型自动转换。
4. `source()` 保留底层错误链，日志/链路追踪才能看到根因。
5. `Box<dyn std::error::Error>` 适合应用层快速汇总错误；库更推荐具体错误枚举（或用 `thiserror`）。
6. `read_text_len` 读取 workspace 根目录的 `Cargo.toml`，从其他目录运行可能失败——这正好让你观察错误输出。

## 动手练习

1. 给 `AppError` 增加 `Io(std::io::Error)` 变体，并让 `read_text_len` 返回 `AppError`。
2. 用 `thiserror`（可选依赖）重写错误类型，对比手写 `Display` 的代码量。
3. 在 `parse_score` 中加入「必须是偶数」的规则，设计新的错误变体与文案。
4. 把 `unwrap()` 全部替换为 `expect("...")`，思考为什么 `expect` 的信息更利于排障。
5. 在服务端场景中思考：用户输入错误应返回 400 还是 500？错误类型如何映射到 HTTP 状态码？（第 12、13 章实践）

## 工程提醒

- 库代码不要 `panic` 来表达可预期失败；`panic` 用于 bug 与不可恢复状态。
- `unwrap` 在测试、原型、已证明不可能失败的地方可以接受。
- 自定义错误应实现 `Send + Sync + 'static`（多线程/异步场景需要，编译器会提示）。
