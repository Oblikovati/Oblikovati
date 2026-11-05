// SPDX-License-Identifier: GPL-2.0-only
use super::{LineScanner, LuaTokenKind};
impl LineScanner {
    pub(super) fn scan_quoted(&mut self, quote: char) {
        let start = self.cursor;
        self.cursor += 1;
        while let Some(character) = self.source.get(self.cursor).copied() {
            if character == '\\' {
                self.cursor = (self.cursor + 2).min(self.source.len());
                continue;
            }
            self.cursor += 1;
            if character == quote {
                break;
            }
        }
        self.emit(LuaTokenKind::String, start, self.cursor);
    }
    pub(super) fn scan_number(&mut self) {
        let start = self.cursor;
        let hex = self.source[self.cursor] == '0' && matches!(self.peek_next(), Some('x' | 'X'));
        if hex {
            self.cursor += 2;
            self.consume_while(|character| character.is_ascii_hexdigit() || character == '.');
        } else {
            self.consume_while(|character| character.is_ascii_digit() || character == '.');
        }
        self.consume_exponent(if hex { ['p', 'P'] } else { ['e', 'E'] });
        self.emit(LuaTokenKind::Number, start, self.cursor);
    }
    fn consume_exponent(&mut self, markers: [char; 2]) {
        if !self
            .source
            .get(self.cursor)
            .is_some_and(|character| markers.contains(character))
        {
            return;
        }
        self.cursor += 1;
        if matches!(self.source.get(self.cursor), Some('+' | '-')) {
            self.cursor += 1;
        }
        self.consume_while(|character| character.is_ascii_digit());
    }
}
