// SPDX-License-Identifier: GPL-2.0-only
//! Keep the content selection when the user works with the adjacent editor controls.
use imgui::{InputTextCallbackHandler, InputTextMultilineCallback, TextCallbackData, Ui};
use std::ops::Range;

#[path = "text_numbering.rs"]
mod numbering;
use numbering::NumberingStyle;

#[derive(Default)]
pub(super) struct TextContentEditor {
    selection: Range<usize>,
    cursor: usize,
    restore: Option<Range<usize>>,
}

impl TextContentEditor {
    pub(super) fn draw(&mut self, ui: &Ui, content: &mut String) -> bool {
        self.keep_valid_selection(content);
        if self.restore.is_some() {
            ui.set_keyboard_focus_here();
        }
        let typed = ui
            .input_text_multiline("Content", content, [295., 100.])
            .callback(InputTextMultilineCallback::ALWAYS, SelectionCallback(self))
            .build();
        typed | self.controls(ui, content)
    }

    fn controls(&mut self, ui: &Ui, content: &mut String) -> bool {
        let numbering = numbering_choice(ui);
        ui.same_line();
        let symbol = symbol_choice(ui);
        ui.same_line();
        let bullet = NumberingStyle::Bullet;
        let bullets = ui.button(bullet.label()).then_some(bullet);
        if let Some(style) = numbering.or(bullets) {
            return self.number_paragraphs(content, style);
        }
        symbol.is_some_and(|symbol| self.insert_symbol(content, symbol))
    }

    fn number_paragraphs(&mut self, content: &mut String, style: NumberingStyle) -> bool {
        let selected = self.selection_or_cursor();
        let Some((range, replacement)) = numbering::replace_numbering(content, selected, style)
        else {
            return false;
        };
        let end = range.start + replacement.len();
        self.restore = Some(range.start..end);
        content.replace_range(range, &replacement);
        true
    }

    fn insert_symbol(&mut self, content: &mut String, symbol: &str) -> bool {
        let range = self.selection_or_cursor();
        if content.get(range.clone()).is_none() {
            return false;
        }
        let next = range.start + symbol.len();
        content.replace_range(range, symbol);
        self.restore = Some(next..next);
        true
    }

    fn selection_or_cursor(&self) -> Range<usize> {
        if self.selection.is_empty() {
            self.cursor..self.cursor
        } else {
            self.selection.clone()
        }
    }

    fn keep_valid_selection(&mut self, content: &str) {
        if content.get(self.selection.clone()).is_none() || !content.is_char_boundary(self.cursor) {
            self.cursor = content.len();
            self.selection = self.cursor..self.cursor;
        }
    }
}

struct SelectionCallback<'a>(&'a mut TextContentEditor);
impl InputTextCallbackHandler for SelectionCallback<'_> {
    fn on_always(&mut self, mut input: TextCallbackData) {
        if let Some(range) = self
            .0
            .restore
            .take()
            .filter(|range| input.str().get(range.clone()).is_some())
            && let (Ok(start), Ok(end)) = (i32::try_from(range.start), i32::try_from(range.end))
        {
            input.set_cursor_pos(range.end);
            *input.selection_start_mut() = start;
            *input.selection_end_mut() = end;
        }
        self.0.selection = input.selection();
        self.0.cursor = input.cursor_pos();
    }
}

fn numbering_choice(ui: &Ui) -> Option<NumberingStyle> {
    ui.set_next_item_width(145.);
    let _list = ui.begin_combo("##text-numbering", "Numbering")?;
    NumberingStyle::CHOICES
        .into_iter()
        .find(|style| ui.selectable(style.label()))
}

fn symbol_choice(ui: &Ui) -> Option<&'static str> {
    ui.set_next_item_width(145.);
    let _symbols = ui.begin_combo("##text-symbols", "Symbols")?;
    [("Diameter", "⌀"), ("Degree", "°"), ("Plus or minus", "±")]
        .into_iter()
        .find_map(|(name, symbol)| ui.selectable(name).then_some(symbol))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn symbol_insertion_replaces_a_complete_unicode_selection() {
        let mut editor = TextContentEditor {
            selection: 1..4,
            cursor: 4,
            ..Default::default()
        };
        let mut content = "a界z".to_owned();
        assert!(editor.insert_symbol(&mut content, "⌀"));
        assert_eq!(content, "a⌀z");
        assert_eq!(editor.restore, Some(4..4));
    }
}
