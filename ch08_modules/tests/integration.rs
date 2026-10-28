//! 集成测试：只能使用 crate 的公开 API。
//! 这里验证 `pub use` 暴露的接口是否可用。

use ch08_modules::{mean, word_count};

#[test]
fn end_to_end_metrics() {
    let text = "rust makes systems programming fun";
    assert_eq!(word_count(text), 5);
    assert_eq!(mean(&[1.0, 2.0, 3.0]), Some(2.0));
}
