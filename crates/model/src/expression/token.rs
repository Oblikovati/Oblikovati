// SPDX-License-Identifier: GPL-2.0-only

use super::ExpressionError;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(super) enum TokenKind {
    Number,
    Name,
    Plus,
    Minus,
    Star,
    Slash,
    Open,
    Close,
    Comma,
    End,
}

#[derive(Clone, Copy, Debug)]
pub(super) struct Token<'a> {
    pub kind: TokenKind,
    pub text: &'a str,
    pub position: usize,
}

impl Token<'_> {
    pub fn expected(self, expected: &'static str) -> ExpressionError {
        ExpressionError::Syntax {
            position: self.position,
            found: self.text.into(),
            expected,
        }
    }
}

pub(super) fn tokenize(source: &str) -> Result<Vec<Token<'_>>, ExpressionError> {
    let mut tokens = Vec::new();
    let mut position = 0;
    while position < source.len() {
        if matches!(source.as_bytes()[position], b' ' | b'\t') {
            position += 1;
            continue;
        }
        let token = scan_token(source, position)?;
        position += token.text.len();
        tokens.push(token);
    }
    tokens.push(Token {
        kind: TokenKind::End,
        text: "",
        position,
    });
    Ok(tokens)
}

fn scan_token(source: &str, position: usize) -> Result<Token<'_>, ExpressionError> {
    let tail = &source[position..];
    let (kind, length) = scan_span(tail).ok_or_else(|| ExpressionError::Syntax {
        position,
        found: tail.chars().next().unwrap().to_string(),
        expected: "a number, identifier, or arithmetic punctuation",
    })?;
    Ok(Token {
        kind,
        text: &tail[..length],
        position,
    })
}

fn scan_span(tail: &str) -> Option<(TokenKind, usize)> {
    let byte = tail.as_bytes()[0];
    if byte.is_ascii_digit() || byte == b'.' {
        return Some((TokenKind::Number, crate::units::numeric_prefix_len(tail)));
    }
    if identifier_start(byte) {
        let length = tail
            .bytes()
            .take_while(|byte| identifier_start(*byte) || byte.is_ascii_digit())
            .count();
        return Some((TokenKind::Name, length));
    }
    punctuation(byte).map(|kind| (kind, 1))
}

fn identifier_start(byte: u8) -> bool {
    byte == b'_' || byte >= 0x80 || byte.is_ascii_alphabetic()
}

fn punctuation(byte: u8) -> Option<TokenKind> {
    match byte {
        b'+' => Some(TokenKind::Plus),
        b'-' => Some(TokenKind::Minus),
        b'*' => Some(TokenKind::Star),
        b'/' => Some(TokenKind::Slash),
        b'(' => Some(TokenKind::Open),
        b')' => Some(TokenKind::Close),
        b',' => Some(TokenKind::Comma),
        _ => None,
    }
}
