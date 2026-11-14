// SPDX-License-Identifier: GPL-2.0-only
//! Line-oriented Lua highlighting; token spans count Unicode characters, not bytes.
mod long;
mod scan;
use crate::vocabulary::{BUILTINS, KEYWORDS, is_identifier};
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LuaTokenKind {
    Identifier,
    Keyword,
    Builtin,
    String,
    Number,
    Comment,
    Operator,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct LuaToken {
    pub kind: LuaTokenKind,
    pub start: usize,
    pub end: usize,
}
/// Only long-bracket constructs carry lexical context across lines.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum LuaLineState {
    #[default]
    Plain,
    LongString {
        level: usize,
    },
    LongComment {
        level: usize,
    },
}
/// Highlight one line, carrying its state into the next; example: `tokenize_line("local x=1", LuaLineState::Plain)`.
/// This liberal lexer supplies colors; Lua parsing supplies validity diagnostics.
pub fn tokenize_line(line: &str, incoming: LuaLineState) -> (Vec<LuaToken>, LuaLineState) {
    let mut scanner = LineScanner {
        source: line.chars().collect(),
        cursor: 0,
        tokens: Vec::new(),
        carry: LuaLineState::Plain,
    };
    if scanner.resume_long(incoming) {
        return (scanner.tokens, incoming);
    }
    while scanner.cursor < scanner.source.len() {
        scanner.scan_next();
    }
    (scanner.tokens, scanner.carry)
}
struct LineScanner {
    source: Vec<char>,
    cursor: usize,
    tokens: Vec<LuaToken>,
    carry: LuaLineState,
}
impl LineScanner {
    fn scan_next(&mut self) {
        let character = self.source[self.cursor];
        match character {
            ' ' | '\t' => self.cursor += 1,
            '-' if self.peek_next() == Some('-') => self.scan_comment(),
            '\'' | '"' => self.scan_quoted(character),
            '[' if long::bracket_level(&self.source, self.cursor).is_some() => {
                self.scan_long(false)
            }
            character
                if character.is_ascii_digit()
                    || (character == '.'
                        && self.peek_next().is_some_and(|next| next.is_ascii_digit())) =>
            {
                self.scan_number()
            }
            character if is_identifier(character) => self.scan_identifier(),
            _ => self.scan_operator(),
        }
    }
    fn peek_next(&self) -> Option<char> {
        self.source.get(self.cursor + 1).copied()
    }
    fn emit(&mut self, kind: LuaTokenKind, start: usize, end: usize) {
        if end > start {
            self.tokens.push(LuaToken { kind, start, end });
        }
    }
    fn scan_identifier(&mut self) {
        let start = self.cursor;
        self.consume_while(is_identifier);
        let word: String = self.source[start..self.cursor].iter().collect();
        let kind = if KEYWORDS.contains(&word.as_str()) {
            LuaTokenKind::Keyword
        } else if BUILTINS.contains(&word.as_str()) {
            LuaTokenKind::Builtin
        } else {
            LuaTokenKind::Identifier
        };
        self.emit(kind, start, self.cursor);
    }
    fn scan_operator(&mut self) {
        self.emit(LuaTokenKind::Operator, self.cursor, self.cursor + 1);
        self.cursor += 1;
    }
    fn consume_while(&mut self, predicate: fn(char) -> bool) {
        while self.source.get(self.cursor).copied().is_some_and(predicate) {
            self.cursor += 1;
        }
    }
}
