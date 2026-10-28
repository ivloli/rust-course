// 第 06 章的 Go 平行版本。
// 每个片段都对应 ch06_errors/src/main.rs 中的 Rust 写法。
// 运行：cd ch06_errors/go && go run .
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrEmptyInput 是哨兵错误（sentinel error），对应 Rust 枚举变体
// AppError::EmptyInput。Go 用「一个包级 error 变量」表示一种固定错误，
// 调用方用 errors.Is 判断，而不是 match 枚举。
var ErrEmptyInput = errors.New("输入不能为空")

// OutOfRangeError 对应 AppError::OutOfRange(i64)。
// Rust 的枚举可以携带数据；Go 用实现了 error 接口的结构体携带数据。
type OutOfRangeError struct {
	Value int
}

func (e *OutOfRangeError) Error() string {
	return fmt.Sprintf("数值 %d 超出 1..=100 范围", e.Value)
}

// NotANumberError 对应 AppError::NotANumber(ParseIntError)。
// Rust 在 source() 里返回底层错误；Go 用 Unwrap() 暴露底层错误，
// 这样 errors.Is / errors.As 可以沿错误链继续查找。
type NotANumberError struct {
	Err error
}

func (e *NotANumberError) Error() string {
	// Rust 的 Display：write!(f, "不是合法整数: {e}")
	return fmt.Sprintf("不是合法整数: %v", e.Err)
}

// Unwrap 等价于 Rust 的 std::error::Error::source。
func (e *NotANumberError) Unwrap() error {
	return e.Err
}

// parseScore 对应 Rust 的 parse_score。
//
// Rust 版本：
//
//	fn parse_score(input: &str) -> Result<u32, AppError>
//
// 失败时用 `?` 自动把 ParseIntError 通过 From 转成 AppError；
// Go 没有 `?`，每一步都要显式 `if err != nil { return ..., err }`。
func parseScore(input string) (uint32, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		// Rust: return Err(AppError::EmptyInput)
		return 0, ErrEmptyInput
	}

	// Atoi 的第二个返回值就是 Go 的「错误分支」；没有异常，也没有 ?
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		// Rust: `input.trim().parse()?` 靠 From<ParseIntError> 自动包装；
		// Go 必须手动包装。注意底层文案来自 strconv，和 Rust 不同，见文档。
		return 0, &NotANumberError{Err: err}
	}

	if value < 1 || value > 100 {
		// Rust: return Err(AppError::OutOfRange(value))
		return 0, &OutOfRangeError{Value: value}
	}
	return uint32(value), nil
}

// describe 演示 errors.Is / errors.As，对应 Rust 对 AppError 的 match。
//
// Rust 可以直接 match 枚举变体；Go 只能：
//   - errors.Is 判断某个哨兵错误是否在错误链上；
//   - errors.As 把错误链上某个具体类型取出来。
//
// 这里返回的字符串与 Rust 的 Display 输出一致。
func describe(err error) string {
	var outOfRange *OutOfRangeError
	var notANumber *NotANumberError

	switch {
	case errors.Is(err, ErrEmptyInput):
		return err.Error()
	case errors.As(err, &outOfRange):
		return outOfRange.Error()
	case errors.As(err, &notANumber):
		return notANumber.Error()
	default:
		return err.Error()
	}
}

// get 模拟 Rust 的 slice::get：返回值是否存在用 comma-ok 表示，
// 对应 Option<&T>。Go 没有 Option，这是最常用的替代写法。
func get(values []int, index int) (int, bool) {
	if index < 0 || index >= len(values) {
		return 0, false
	}
	return values[index], true
}

func main() {
	// 1. Option：可能不存在的值。
	// Rust 的 numbers.get(1) 返回 Option<&i32>，这里用 (值, ok) 模拟。
	numbers := []int{1, 2, 3}
	if value, ok := get(numbers, 1); ok {
		fmt.Printf("get(1) = Some(%d)\n", value)
	} else {
		fmt.Println("get(1) = None")
	}
	if value, ok := get(numbers, 9); ok {
		fmt.Printf("get(9) = Some(%d)\n", value)
	} else {
		fmt.Println("get(9) = None")
	}

	// Rust: iter().find(..).copied().map(|n| n * 10).unwrap_or(0)
	// Go: 先找，再判断 ok；没有链式 Option 组合子。
	firstEven, found := 0, false
	for _, n := range numbers {
		if n%2 == 0 {
			firstEven, found = n, true
			break
		}
	}
	scaled := 0
	if found {
		scaled = firstEven * 10
	}
	fmt.Println("first_even * 10 =", scaled)

	// 2. Result + ? 运算符：Go 用 (T, error) 加显式 err 检查。
	for _, input := range []string{"42", "", "abc", "150", " 7 "} {
		score, err := parseScore(input)
		if err != nil {
			fmt.Printf("%q -> 错误: %s\n", input, describe(err))
		} else {
			fmt.Printf("%q -> %d\n", input, score)
		}
	}

	// 3. unwrap 系列。
	// Rust 的 unwrap() 确定不会失败；Go 直接忽略 error 或显式检查。
	parsed, _ := strconv.Atoi("10")
	// Rust 的 unwrap_or(-1)；Go 用 if err != nil 给默认值。
	fallback, err := strconv.Atoi("x")
	if err != nil {
		fallback = -1
	}
	// Rust 的 map(|n| n * 2).unwrap_or_default()；Go 显式写 if/else。
	computed, err := strconv.Atoi("8")
	if err != nil {
		computed = 0
	} else {
		computed *= 2
	}
	fmt.Printf("parsed = %d, fallback = %d, computed = %d\n", parsed, fallback, computed)

	// 4. Box<dyn Error> ↔ error 接口。
	// Rust 程序从 workspace 根目录运行，读 "Cargo.toml"；
	// Go 程序从 ch06_errors/go 运行，要读 "../../Cargo.toml" 才能指向同一个文件。
	if length, err := readTextLen("../../Cargo.toml"); err != nil {
		fmt.Printf("读取失败: %v\n", err)
	} else {
		fmt.Printf("Cargo.toml 长度 = %d 字节\n", length)
	}

	// 5. main 返回 Result：Rust 会打印错误并设置退出码；
	// Go 的 main 不能返回 error，通常打印后 os.Exit(1)。
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "run() 失败: %v\n", err)
		os.Exit(1)
	}
}

// readTextLen 对应 Rust 的 read_text_len。
// Rust 返回 Result<usize, Box<dyn Error>>；Go 直接返回 (int, error)，
// 因为 error 本身就是接口，不需要再装箱。
func readTextLen(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return len(content), nil
}

// run 对应 Rust 的 run：内部用 ? 提前返回，成功时打印结果。
func run() error {
	score, err := parseScore("100")
	if err != nil {
		return err
	}
	fmt.Printf("run() 解析结果 = %d\n", score)
	return nil
}
