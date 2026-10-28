//! 第 08 章：模块与包结构示例
//!
//! - `math` / `text` 是 crate 内部的模块（对应 src/math.rs、src/text.rs）
//! - `pub use` 把常用项提升到 crate 根，调用方不必关心内部层级

pub mod math;
pub mod text;

pub use math::{gcd, stats::mean};
pub use text::word_count;
