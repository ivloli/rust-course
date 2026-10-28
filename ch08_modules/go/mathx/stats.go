package mathx

// Mean 对应 Rust 的 math::stats::mean。
//
// Rust 用嵌套模块 math::stats 表达层级；Go 里同一个目录下的文件都属于
// 同一个包，所以这里把 stats 摊平成 stats.go。若要保留嵌套路径，
// 可以新建 mathx/stats 子目录，形成 ch08_go/mathx/stats 包。
//
// Rust 的空切片返回 Option<f64>（None 避免除零）；
// Go 没有 Option，用 (float64, bool) 的 comma-ok 模式表达。
func Mean(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values)), true
}
