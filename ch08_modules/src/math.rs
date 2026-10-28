pub fn gcd(a: u64, b: u64) -> u64 {
    let (mut a, mut b) = (a, b);
    while b != 0 {
        let r = a % b;
        a = b;
        b = r;
    }
    a
}

pub mod stats {
    /// 空切片返回 None，避免除零。
    pub fn mean(values: &[f64]) -> Option<f64> {
        if values.is_empty() {
            return None;
        }
        Some(values.iter().sum::<f64>() / values.len() as f64)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gcd_works() {
        assert_eq!(gcd(48, 18), 6);
        assert_eq!(gcd(17, 13), 1);
    }

    #[test]
    fn gcd_with_zero() {
        assert_eq!(gcd(0, 5), 5);
    }

    #[test]
    fn mean_works() {
        assert_eq!(stats::mean(&[2.0, 4.0]), Some(3.0));
    }

    #[test]
    fn mean_of_empty_is_none() {
        assert_eq!(stats::mean(&[]), None);
    }
}
