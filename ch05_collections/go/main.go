// 第 05 章的 Go 平行版本。
// 每个例子都对应 ch05_collections/src/main.rs 中的 Rust 写法，
// 注释从 Go 视角解释两者的差异。
package main

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type wordCount struct {
	word  string
	count int
}

func main() {
	// 1. Rust 的 Vec<i32> 对应 Go 的 []int。
	// Rust 的 push 原地修改；Go 的 append 可能扩容并返回新 slice，所以必须接返回值。
	numbers := []int{3, 1, 2}
	slices.Sort(numbers)
	numbers = append(numbers, 5)
	fmt.Println("numbers =", formatInts(numbers))

	// Rust 的 numbers.get(10) 返回 Option<&i32>，越界不会 panic。
	// Go 没有 Option：用「值 + ok」两个返回值，或者先判断下标范围。
	if value, ok := get(numbers, 10); ok {
		fmt.Printf("get(10) = Some(%d)\n", value)
	} else {
		fmt.Println("get(10) = None")
	}
	// 直接 numbers[10] 会 panic，和 Rust 的 Vec 下标行为一样。

	// Rust 的 iter().map().collect() 在 Go 里通常就是一个 for 循环。
	// Go 标准库没有 Iterator::map 那样的链式 API，这是最核心的差异。
	doubled := make([]int, 0, len(numbers))
	for _, n := range numbers {
		doubled = append(doubled, n*2)
	}
	fmt.Println("doubled =", formatInts(doubled))

	// 2. Rust 的 String 是可增长的堆字符串，push_str 原地修改；
	// Go 的 string 不可变，+= 会生成新字符串。频繁拼接请用 strings.Builder。
	text := "hello"
	text += " rust"
	text += "!"
	fmt.Printf("%s | 大写: %s\n", text, strings.ToUpper(text))

	// 两者都用 UTF-8：len 是字节数，字符数要单独算。
	// Rust: chars().count()；Go: utf8.RuneCountInString。
	fmt.Println("字符数:", utf8.RuneCountInString("中文abc"))
	fmt.Println("字节数:", len("中文abc"))

	// 3. Rust 的 entry().or_insert(0) += 1，在 Go 里就是 m[k]++。
	// 因为 map 的零值就是 0，这是 Go 比 Rust 更简洁的一个地方。
	sentence := "the quick brown fox jumps over the lazy dog the fox"
	counts := make(map[string]int)
	for _, word := range strings.Fields(sentence) {
		counts[word]++
	}

	// Rust 的 HashMap 和 Go 的 map 遍历顺序都不保证；输出前要排序。
	ranked := make([]wordCount, 0, len(counts))
	for word, count := range counts {
		ranked = append(ranked, wordCount{word, count})
	}
	slices.SortFunc(ranked, func(a, b wordCount) int {
		// 对应 Rust 的 b.1.cmp(&a.1).then(a.0.cmp(b.0))
		return cmp.Or(
			cmp.Compare(b.count, a.count),
			cmp.Compare(a.word, b.word),
		)
	})
	for _, item := range ranked[:min(3, len(ranked))] {
		fmt.Printf("%s: %d\n", item.word, item.count)
	}

	// 4. Rust 的 filter().map().collect() 是惰性迭代器链；
	// Go 的等价写法是 for + if + append（可读性往往更好）。
	evensSquared := make([]int, 0, 5)
	for n := 1; n <= 10; n++ {
		if n%2 == 0 {
			evensSquared = append(evensSquared, n*n)
		}
	}
	fmt.Println("evens_squared =", formatInts(evensSquared))

	// Rust 有 Iterator::sum / product；Go 没有泛型版的 sum，直接循环。
	sum := 0
	for n := 1; n <= 100; n++ {
		sum += n
	}
	product := 1
	for n := 1; n <= 10; n++ {
		product *= n
	}
	fmt.Printf("1..=100 求和 = %d, 1..=10 求积 = %d\n", sum, product)

	// Rust 用 iter().max()；Go 1.21+ 的 slices.Max 直接接收 []int。
	fmt.Println("max =", slices.Max([]int{3, 9, 2}))

	// Rust 的 map + collect + join；Go 用循环 + strings.Join。
	names := []string{"alice", "bob"}
	scores := []int{90, 82}
	upperNames := make([]string, len(names))
	for i, name := range names {
		upperNames[i] = strings.ToUpper(name)
	}
	fmt.Println("joined =", strings.Join(upperNames, ", "))

	// Rust 的 zip().enumerate()；Go 用下标同时访问两个 slice。
	// Go 没有内置 zip，长度不一致时要自己处理边界。
	for i := range names {
		fmt.Printf("#%d: %s -> %d\n", i, names[i], scores[i])
	}

	// Rust 的 fold 在 Go 里就是「声明累加变量 + 循环」。
	csvParts := make([]string, len(scores))
	for i, score := range scores {
		csvParts[i] = strconv.Itoa(score)
	}
	fmt.Println("csv =", strings.Join(csvParts, ","))

	// 5. 闭包捕获环境：
	// Rust 会按需要推导为 Fn / FnMut / FnOnce，所以 calls 需要 mut；
	// Go 的闭包总是按引用捕获外层变量，不需要 mut 关键字。
	threshold := 5
	above := make([]int, 0)
	for n := 1; n <= 10; n++ {
		if n > threshold {
			above = append(above, n)
		}
	}
	fmt.Printf("大于 %d 的数: %s\n", threshold, formatInts(above))

	calls := 0
	increment := func() { calls++ } // Go 闭包直接修改外层 calls
	for range 3 {
		increment()
	}
	fmt.Println("calls =", calls)
}

// get 模拟 Rust 的 Vec::get：安全访问，返回 (值, 是否存在)。
// 这是 Go 里对应 Option 的最常见写法（value, ok 模式）。
func get(values []int, index int) (int, bool) {
	if index < 0 || index >= len(values) {
		return 0, false
	}
	return values[index], true
}

// formatInts 只是为了输出和 Rust 的 {:?} 格式对齐：[1, 2, 3]。
func formatInts(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
