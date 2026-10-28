pub fn word_count(text: &str) -> usize {
    text.split_whitespace().count()
}

pub fn shout(text: &str) -> String {
    text.to_uppercase()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn counts_words() {
        assert_eq!(word_count("  hello   world  "), 2);
    }

    #[test]
    fn empty_text() {
        assert_eq!(word_count(""), 0);
    }

    #[test]
    fn shout_works() {
        assert_eq!(shout("ok"), "OK");
    }
}
