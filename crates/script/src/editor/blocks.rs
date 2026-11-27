// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const LUA_INDENT: &str = "    ";
impl ScriptEditor {
    /// Indent touched lines as one edit and retain their selection; example: `editor.indent_selection()`.
    pub fn indent_selection(&mut self) {
        let (first, last) = self.selected_lines();
        let replacement = self
            .block_lines(first, last)
            .into_iter()
            .map(|line| format!("{LUA_INDENT}{line}"))
            .collect();
        self.replace_block(first, last, replacement);
    }
    /// Remove up to four leading spaces or one tab; example: `editor.outdent_selection()`.
    pub fn outdent_selection(&mut self) {
        let (first, last) = self.selected_lines();
        let replacement = self
            .block_lines(first, last)
            .iter()
            .map(|line| outdent_line(line))
            .collect();
        self.replace_block(first, last, replacement);
    }
    /// Toggle Lua comments at shared indentation; example: `editor.toggle_line_comment()`.
    pub fn toggle_line_comment(&mut self) {
        let (first, last) = self.selected_lines();
        let lines = self.block_lines(first, last);
        let uncomment = all_commented(&lines);
        let indentation = minimum_indent(&lines);
        let replacement = lines
            .iter()
            .map(|line| {
                if uncomment {
                    uncomment_line(line)
                } else {
                    comment_line(line, indentation)
                }
            })
            .collect();
        self.replace_block(first, last, replacement);
    }
    fn selected_lines(&self) -> (usize, usize) {
        let (first, last) = self.selection.ordered();
        let exclude_last = last.column == 0 && last.line > first.line;
        (first.line, last.line - usize::from(exclude_last))
    }
    fn block_lines(&self, first: usize, last: usize) -> Vec<String> {
        self.buffer.lines[first..=last]
            .iter()
            .map(|line| line.iter().collect())
            .collect()
    }
    fn replace_block(&mut self, first: usize, last: usize, lines: Vec<String>) {
        let anchor = EditorPosition {
            line: first,
            column: 0,
        };
        let old_end = EditorPosition {
            line: last,
            column: self.buffer.lines[last].len(),
        };
        self.replace(anchor, old_end, &lines.join("\n"));
        let caret = EditorPosition {
            line: first + lines.len() - 1,
            column: lines.last().expect("one selected line").chars().count(),
        };
        self.selection = EditorSelection { anchor, caret };
        self.goal_column = caret.column;
    }
}
fn all_commented(lines: &[String]) -> bool {
    let nonblank: Vec<&str> = lines
        .iter()
        .map(|line| line.trim_start_matches([' ', '\t']))
        .filter(|line| !line.is_empty())
        .collect();
    !nonblank.is_empty() && nonblank.iter().all(|line| line.starts_with("--"))
}
fn minimum_indent(lines: &[String]) -> usize {
    lines
        .iter()
        .filter(|line| !line.trim().is_empty())
        .map(|line| {
            line.chars()
                .take_while(|character| *character == ' ' || *character == '\t')
                .count()
        })
        .min()
        .unwrap_or(0)
}
fn comment_line(line: &str, indentation: usize) -> String {
    if line.trim().is_empty() {
        return line.into();
    }
    let split = line
        .char_indices()
        .nth(indentation)
        .map_or(line.len(), |(offset, _)| offset);
    format!("{}-- {}", &line[..split], &line[split..])
}
fn uncomment_line(line: &str) -> String {
    let body = line.trim_start_matches([' ', '\t']);
    let Some(uncommented) = body.strip_prefix("--") else {
        return line.into();
    };
    let indentation = &line[..line.len() - body.len()];
    format!(
        "{indentation}{}",
        uncommented.strip_prefix(' ').unwrap_or(uncommented)
    )
}
fn outdent_line(line: &str) -> String {
    if let Some(body) = line.strip_prefix('\t') {
        return body.into();
    }
    let width = line
        .chars()
        .take(4)
        .take_while(|character| *character == ' ')
        .count();
    line[width..].into()
}
