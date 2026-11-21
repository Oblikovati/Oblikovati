// SPDX-License-Identifier: GPL-2.0-only
//! Apply editable list markers to the paragraphs intersecting the current selection.
use std::ops::Range;

const INDENT: &str = "  ";
const BULLET_PREFIX: &str = "• ";

#[derive(Clone, Copy)]
pub(super) enum NumberingStyle {
    Bullet,
    Numeric,
    LowerAlphabetic,
    UpperAlphabetic,
    LowerRoman,
    UpperRoman,
}

impl NumberingStyle {
    pub(super) const CHOICES: [Self; 5] = [
        Self::Numeric,
        Self::LowerAlphabetic,
        Self::UpperAlphabetic,
        Self::LowerRoman,
        Self::UpperRoman,
    ];

    pub(super) fn label(self) -> &'static str {
        match self {
            Self::Bullet => "Bullets",
            Self::Numeric => "Numeric",
            Self::LowerAlphabetic => "Lower alphabetic",
            Self::UpperAlphabetic => "Upper alphabetic",
            Self::LowerRoman => "Lower Roman",
            Self::UpperRoman => "Upper Roman",
        }
    }

    fn prefix(self, number: usize) -> String {
        let marker = match self {
            Self::Bullet => return BULLET_PREFIX.into(),
            Self::Numeric => number.to_string(),
            Self::LowerAlphabetic => alphabetical(number),
            Self::UpperAlphabetic => alphabetical(number).to_ascii_uppercase(),
            Self::LowerRoman => roman(number).to_ascii_lowercase(),
            Self::UpperRoman => roman(number),
        };
        format!("{marker}. ")
    }
}

pub(super) fn replace_numbering(
    content: &str,
    selection: Range<usize>,
    style: NumberingStyle,
) -> Option<(Range<usize>, String)> {
    let range = paragraph_bounds(content, selection)?;
    let paragraphs = content[range.clone()].split('\n');
    let next = paragraphs
        .enumerate()
        .map(|(index, paragraph)| {
            format!(
                "{INDENT}{}{}",
                style.prefix(index + 1),
                without_list_marker(paragraph, index + 1)
            )
        })
        .collect::<Vec<_>>()
        .join("\n");
    Some((range, next))
}

fn paragraph_bounds(content: &str, selection: Range<usize>) -> Option<Range<usize>> {
    let selected = content.get(selection.clone())?;
    let first = content[..selection.start]
        .rfind('\n')
        .map_or(0, |newline| newline + 1);
    let last = if selected.ends_with('\n') {
        selection.end - 1
    } else {
        content[selection.end..]
            .find('\n')
            .map_or(content.len(), |offset| selection.end + offset)
    };
    Some(first..last)
}

fn without_list_marker(paragraph: &str, number: usize) -> &str {
    let Some(indented) = paragraph.strip_prefix(INDENT) else {
        return paragraph;
    };
    if let Some(content) = indented.strip_prefix(BULLET_PREFIX) {
        return content;
    }
    NumberingStyle::CHOICES
        .into_iter()
        .find_map(|style| indented.strip_prefix(&style.prefix(number)))
        .unwrap_or(paragraph)
}

fn alphabetical(mut number: usize) -> String {
    let mut letters = Vec::new();
    while number > 0 {
        number -= 1;
        letters.push(char::from(b'a' + (number % 26) as u8));
        number /= 26;
    }
    letters.into_iter().rev().collect()
}

fn roman(mut number: usize) -> String {
    let mut marker = String::new();
    const DIGITS: [(usize, &str); 13] = [
        (1000, "M"),
        (900, "CM"),
        (500, "D"),
        (400, "CD"),
        (100, "C"),
        (90, "XC"),
        (50, "L"),
        (40, "XL"),
        (10, "X"),
        (9, "IX"),
        (5, "V"),
        (4, "IV"),
        (1, "I"),
    ];
    for (value, glyph) in DIGITS {
        for _ in 0..number / value {
            marker.push_str(glyph);
        }
        number %= value;
    }
    marker
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn numbering_preserves_unselected_paragraphs_and_unicode() {
        let content = "before\n界a\nβ\nafter";
        let (range, replacement) =
            replace_numbering(content, 10..15, NumberingStyle::Numeric).unwrap();
        let mut edited = content.to_owned();
        edited.replace_range(range, &replacement);
        assert_eq!(edited, "before\n  1. 界a\n  2. β\nafter");
    }

    #[test]
    fn numbered_text_keeps_an_authored_word_followed_by_a_period() {
        let content = "  Hint. Keep this text";
        let (_, replacement) =
            replace_numbering(content, 0..content.len(), NumberingStyle::Numeric).unwrap();
        assert_eq!(replacement, "  1.   Hint. Keep this text");
    }

    #[test]
    fn switching_style_replaces_existing_markers() {
        let content = "  1. a\n  2. b";
        let (_, replacement) =
            replace_numbering(content, 0..content.len(), NumberingStyle::LowerRoman).unwrap();
        assert_eq!(replacement, "  i. a\n  ii. b");
        assert_eq!(NumberingStyle::LowerAlphabetic.prefix(27), "aa. ");
        assert_eq!(NumberingStyle::UpperAlphabetic.prefix(27), "AA. ");
        assert_eq!(NumberingStyle::UpperRoman.prefix(49), "XLIX. ");
    }

    #[test]
    fn numbering_replaces_bullets_without_removing_paragraph_content() {
        let content = "  • a\n  • β";
        let (_, replacement) =
            replace_numbering(content, 0..content.len(), NumberingStyle::Numeric).unwrap();
        assert_eq!(replacement, "  1. a\n  2. β");
    }

    #[test]
    fn bullets_replace_numbering_only_for_selected_paragraphs() {
        let content = "  1. a\n  2. β\nkeep";
        let selection = 0..content.rfind('\n').unwrap();
        let (range, replacement) =
            replace_numbering(content, selection, NumberingStyle::Bullet).unwrap();
        assert_eq!(&content[range.end..], "\nkeep");
        assert_eq!(replacement, "  • a\n  • β");
    }
}
