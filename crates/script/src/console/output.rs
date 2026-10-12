// SPDX-License-Identifier: GPL-2.0-only
use std::{
    collections::VecDeque,
    fmt,
    io::{self, Write},
};
const RETAINED_LINES: usize = 5_000;
const RETAINED_BYTES: usize = 4 << 20;
#[derive(Default)]
pub(super) struct ConsoleOutput {
    lines: VecDeque<String>,
    bytes: usize,
    dropped: u64,
}
impl ConsoleOutput {
    pub(super) fn lines(&self) -> Vec<String> {
        self.lines.iter().cloned().collect()
    }
    pub(super) fn dropped(&self) -> u64 {
        self.dropped
    }
    pub(super) fn clear(&mut self) {
        self.lines.clear();
        self.bytes = 0;
        self.dropped = 0;
    }
    fn append(&mut self, record: String) {
        let record = record.strip_suffix('\n').unwrap_or(&record).to_owned();
        self.bytes = self.bytes.saturating_add(record.len());
        self.lines.push_back(record);
        self.trim_retained();
    }
    fn trim_retained(&mut self) {
        while self.lines.len() > RETAINED_LINES || self.bytes > RETAINED_BYTES {
            let Some(oldest) = self.lines.pop_front() else {
                break;
            };
            self.bytes -= oldest.len();
            self.dropped = self.dropped.saturating_add(1);
        }
    }
}
impl Write for ConsoleOutput {
    fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
        self.append(String::from_utf8_lossy(bytes).into_owned());
        Ok(bytes.len())
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn write_fmt(&mut self, arguments: fmt::Arguments<'_>) -> io::Result<()> {
        // One print callback is one retained record, even when its value contains embedded newlines.
        self.append(arguments.to_string());
        Ok(())
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn retained_records_drop_oldest_and_expose_the_drop_count() {
        let mut output = ConsoleOutput::default();
        for _ in 0..6_000 {
            writeln!(output, "line").unwrap();
        }
        writeln!(output, "last").unwrap();
        assert_eq!(output.lines().len(), RETAINED_LINES);
        assert_eq!(output.lines().last().unwrap(), "last");
        assert_eq!(output.dropped(), 1_001);
    }
    #[test]
    fn bytes_are_bounded_across_runs_and_clear_resets_retention_only() {
        let mut output = ConsoleOutput::default();
        output.write_all(&vec![b'x'; RETAINED_BYTES + 1]).unwrap();
        assert!(output.lines().is_empty());
        assert_eq!(output.dropped(), 1);
        output.flush().unwrap();
        output.clear();
        assert_eq!(output.dropped(), 0);
    }
    #[test]
    fn multiline_print_remains_one_record_and_views_are_copies() {
        let mut output = ConsoleOutput::default();
        writeln!(output, "a\nb").unwrap();
        let mut view = output.lines();
        assert_eq!(view, ["a\nb"]);
        view[0] = "tampered".into();
        assert_eq!(output.lines(), ["a\nb"]);
    }
}
