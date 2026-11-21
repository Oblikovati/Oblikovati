// SPDX-License-Identifier: GPL-2.0-only
use super::{LineScanner, LuaLineState, LuaTokenKind};
impl LineScanner {
    pub(super) fn resume_long(&mut self, incoming: LuaLineState) -> bool {
        let (level, kind) = match incoming {
            LuaLineState::Plain => return false,
            LuaLineState::LongString { level } => (level, LuaTokenKind::String),
            LuaLineState::LongComment { level } => (level, LuaTokenKind::Comment),
        };
        let (end, closed) = long_tail(&self.source, 0, level);
        self.emit(kind, 0, end);
        self.cursor = end;
        !closed
    }
    pub(super) fn scan_comment(&mut self) {
        if bracket_level(&self.source, self.cursor + 2).is_some() {
            self.scan_long(true);
            return;
        }
        self.emit(LuaTokenKind::Comment, self.cursor, self.source.len());
        self.cursor = self.source.len();
    }
    pub(super) fn scan_long(&mut self, comment: bool) {
        let open = self.cursor + if comment { 2 } else { 0 };
        let level = bracket_level(&self.source, open).expect("classified long bracket");
        let (end, closed) = long_tail(&self.source, open + level + 2, level);
        let kind = if comment {
            LuaTokenKind::Comment
        } else {
            LuaTokenKind::String
        };
        self.emit(kind, self.cursor, end);
        self.cursor = end;
        if closed {
            return;
        }
        self.carry = if comment {
            LuaLineState::LongComment { level }
        } else {
            LuaLineState::LongString { level }
        };
    }
}
pub(super) fn bracket_level(source: &[char], start: usize) -> Option<usize> {
    if source.get(start) != Some(&'[') {
        return None;
    }
    let mut end = start + 1;
    while source.get(end) == Some(&'=') {
        end += 1;
    }
    (source.get(end) == Some(&'[')).then_some(end - start - 1)
}
fn long_tail(source: &[char], start: usize, level: usize) -> (usize, bool) {
    for index in start..source.len() {
        if source[index] != ']' {
            continue;
        }
        let mut end = index + 1;
        while source.get(end) == Some(&'=') {
            end += 1;
        }
        if end - index - 1 == level && source.get(end) == Some(&']') {
            return (end + 1, true);
        }
    }
    (source.len(), false)
}
