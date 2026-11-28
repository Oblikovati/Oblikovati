// SPDX-License-Identifier: GPL-2.0-only
//! Keep the content selection when the user works with the adjacent editor controls.
use super::{GuiSketchTextDraft, GuiSketchTextFieldSources};
use crate::widget_abi::MultilineCallbackData;
use imgui::Ui;
use oblikovati_model::sketch::SketchTextFieldProjection;
use std::ops::Range;

#[path = "text_character_ranges.rs"]
mod character_ranges;
use character_ranges::{TextFaceFlag, selection_has};
#[path = "text_numbering.rs"]
mod numbering;
#[path = "text_case.rs"]
mod text_case;
#[path = "text_color.rs"]
mod text_color;
#[path = "text_field_projection.rs"]
mod text_field_projection;
#[path = "text_fraction.rs"]
mod text_fraction;
#[path = "text_linked_fields.rs"]
mod text_linked_fields;
use numbering::NumberingStyle;
use text_field_projection::{authored_selection, project_position, projected_field_edit};

const CONTENT_PICKER_WIDTH: f32 = 130.;

pub(super) fn apply_text_width(draft: &mut GuiSketchTextDraft, percent: f64) -> bool {
    let Some(range) = all_scalars(&draft.content).filter(|range| !range.is_empty()) else {
        return false;
    };
    let runs = draft.formatting.runs.as_deref().unwrap_or_default();
    let edited = character_ranges::set_text_width(runs, range, percent);
    draft.formatting.runs = (!edited.is_empty()).then_some(edited);
    true
}

#[derive(Default)]
pub(super) struct TextContentEditor {
    selection: Range<usize>,
    cursor: usize,
    restore: Option<Range<usize>>,
    color_dialog: crate::application::model_browser::color_dialog::ColorDialog,
    linked_fields: text_linked_fields::LinkedFieldControls,
}

impl TextContentEditor {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        draft: &mut GuiSketchTextDraft,
        row_factor: f64,
        sources: Option<&GuiSketchTextFieldSources>,
        projection: Option<&SketchTextFieldProjection>,
    ) -> bool {
        self.keep_valid_selection(&draft.content);
        if self.restore.is_some() {
            ui.set_keyboard_focus_here();
        }
        let runs = draft.formatting.runs.as_deref().unwrap_or_default();
        let whole_text = all_scalars(&draft.content);
        let face = [TextFaceFlag::Bold, TextFaceFlag::Italic].map(|flag| {
            whole_text
                .clone()
                .is_some_and(|range| selection_has(runs, range, flag))
        });
        let before = (!runs.is_empty()
            || draft.formatting.stacked_fractions.is_some()
            || draft.formatting.linked_fields.is_some())
        .then(|| draft.content.clone());
        let previous_selection = self.selection_or_cursor();
        let typed = self.input(ui, &mut draft.content, row_factor, face, projection);
        let control_changed = self.controls(ui, &mut draft.content);
        if let Some(before) = before.filter(|text| text != &draft.content) {
            let retained = character_ranges::retain_unchanged_runs(runs, &before, &draft.content);
            draft.formatting.runs = (!retained.is_empty()).then_some(retained);
            text_fraction::retain_unchanged(draft, &before);
            if typed && !control_changed {
                text_linked_fields::retain_after_input(
                    draft,
                    &before,
                    previous_selection,
                    self.cursor,
                );
            } else {
                text_linked_fields::retain_unchanged(draft, &before);
            }
        }
        typed
            | control_changed
            | self.face_control(ui, draft, TextFaceFlag::Bold)
            | self.face_control(ui, draft, TextFaceFlag::Italic)
            | self.case_control(ui, draft)
            | text_color::draw(ui, draft, &self.selection, &mut self.color_dialog)
            | self.fraction_control(ui, draft)
            | self.field_control(ui, draft, sources)
    }

    fn field_control(
        &mut self,
        ui: &Ui,
        draft: &mut GuiSketchTextDraft,
        sources: Option<&GuiSketchTextFieldSources>,
    ) -> bool {
        let Some(field) = sources.and_then(|sources| self.linked_fields.draw(ui, sources)) else {
            return false;
        };
        self.restore = text_linked_fields::insert_field(draft, self.selection_or_cursor(), field);
        self.restore.is_some()
    }

    fn fraction_control(&mut self, ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
        if !text_fraction::draw(ui, draft, &self.selection) {
            return false;
        }
        self.restore = Some(self.selection.clone());
        true
    }

    fn case_control(&mut self, ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
        if !crate::widget_abi::button(ui, "UPPERCASE", [0.; 2]) {
            return false;
        }
        self.restore = text_case::uppercase_selection(draft, self.selection.clone());
        self.restore.is_some()
    }

    fn input(
        &mut self,
        ui: &Ui,
        content: &mut String,
        row_factor: f64,
        [bold, italic]: [bool; 2],
        projection: Option<&SketchTextFieldProjection>,
    ) -> bool {
        let typed = {
            let _font = crate::ui_scale::sketch_content_font(ui, bold, italic)
                .map(|font| ui.push_font(font));
            let normal = ui.current_font_size();
            // Keep the native editor within its pixel range. The canvas retains model units.
            let height = (row_factor * f64::from(normal)).clamp(1., f64::from(CONTENT_SIZE[1]));
            let _advance = crate::widget_abi::text_line_advance(ui, height as f32 / normal);
            self.content_field(ui, content, projection)
        };
        ui.same_line();
        ui.text("Content");
        typed
    }

    fn content_field(
        &mut self,
        ui: &Ui,
        content: &mut String,
        projection: Option<&SketchTextFieldProjection>,
    ) -> bool {
        let Some(projection) = projection else {
            let mut callback = SelectionCallback(self);
            return native_content(ui, content, |input| callback.on_always(input));
        };
        let mut displayed = projection.content.clone();
        let mut changed = false;
        let mut callback = ProjectedContentCallback {
            editor: self,
            projection,
            authored: content,
            changed: &mut changed,
        };
        native_content(ui, &mut displayed, |input| callback.on_always(input));
        if !changed {
            crate::widget_abi::shade_linked_text(ui, c"##Content", projection);
        }
        changed
    }

    fn face_control(
        &mut self,
        ui: &Ui,
        draft: &mut GuiSketchTextDraft,
        flag: TextFaceFlag,
    ) -> bool {
        ui.same_line();
        let range = all_scalars(&draft.content).filter(|range| !range.is_empty());
        // Partial changes need the mixed-font layout tracked in Martensite #576.
        let enabled = range.is_some() && self.selection == (0..draft.content.len());
        let runs = draft.formatting.runs.as_deref().unwrap_or_default();
        let active = range
            .clone()
            .is_some_and(|range| selection_has(runs, range, flag));
        let _disabled = ui.begin_disabled(!enabled);
        let clicked = crate::widget_abi::toggle_button(ui, flag.label(), [0.; 2], active);
        let Some(range) = range.filter(|_| enabled && clicked) else {
            return false;
        };
        let edited = character_ranges::toggle_face_flag(runs, range, flag);
        draft.formatting.runs = (!edited.is_empty()).then_some(edited);
        self.restore = Some(self.selection.clone());
        true
    }

    fn controls(&mut self, ui: &Ui, content: &mut String) -> bool {
        let numbering = numbering_choice(ui);
        ui.same_line();
        let symbol = symbol_choice(ui);
        ui.same_line();
        let bullet = NumberingStyle::Bullet;
        let bullets = crate::widget_abi::button(ui, bullet.label(), [0.; 2]).then_some(bullet);
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

fn all_scalars(text: &str) -> Option<Range<u32>> {
    let end = u32::try_from(text.chars().count()).ok()?;
    Some(0..end)
}

const CONTENT_LABEL: &str = "##Content";
const CONTENT_SIZE: [f32; 2] = [295., 100.];

fn native_content(
    ui: &Ui,
    content: &mut String,
    mut on_always: impl FnMut(&mut MultilineCallbackData),
) -> bool {
    crate::widget_abi::input_text_multiline_always(
        ui,
        CONTENT_LABEL,
        content,
        CONTENT_SIZE,
        &mut on_always,
    )
}

struct ProjectedContentCallback<'a> {
    editor: &'a mut TextContentEditor,
    projection: &'a SketchTextFieldProjection,
    authored: &'a mut String,
    changed: &'a mut bool,
}

impl ProjectedContentCallback<'_> {
    fn on_always(&mut self, input: &mut MultilineCallbackData) {
        let shown = |source| project_position(self.projection, source);
        let previous = self.editor.selection_or_cursor();
        let previous = shown(previous.start)..shown(previous.end);
        let edit = projected_field_edit(self.projection, input.str(), previous, input.cursor_pos())
            .map(|edit| (edit.authored, edit.input, edit.replacement.to_owned()));
        if let Some((authored, displayed, replacement)) = edit {
            self.apply_edit(input, authored, displayed, replacement);
            return;
        }
        self.update_selection(input);
    }

    fn update_selection(&mut self, input: &mut MultilineCallbackData) {
        let shown = |source| project_position(self.projection, source);
        let native_caret = input.cursor_pos();
        let display = native_selection(input);
        let logical = self.editor.restore.take().unwrap_or_else(|| {
            let toward_end = native_caret >= shown(self.editor.cursor);
            authored_selection(self.projection, display.clone(), toward_end)
        });
        let caret = if native_caret == display.start && !display.is_empty() {
            logical.start
        } else {
            logical.end
        };
        set_native_selection(
            input,
            shown(logical.start)..shown(logical.end),
            shown(caret),
        );
        self.editor.selection = logical;
        self.editor.cursor = caret;
    }

    fn apply_edit(
        &mut self,
        input: &mut MultilineCallbackData,
        authored: Range<usize>,
        displayed: Range<usize>,
        replacement: String,
    ) {
        let caret = authored.start + replacement.len();
        self.authored.replace_range(authored, &replacement);
        self.editor.restore = None;
        self.editor.cursor = caret;
        self.editor.selection = caret..caret;
        *self.changed = true;
        if input.str()[displayed.clone()] != replacement {
            let count = input.str()[displayed.clone()].chars().count();
            input.remove_chars(displayed.start, count);
            input.insert_chars(displayed.start, &replacement);
        }
        let caret = displayed.start + replacement.len();
        set_native_selection(input, caret..caret, caret);
    }
}

fn native_selection(input: &MultilineCallbackData) -> Range<usize> {
    let selection = input.selection();
    if !selection.is_empty() {
        return selection;
    }
    // ImGui can retain an empty selection at the old offset after moving its caret.
    let caret = input.cursor_pos();
    caret..caret
}

fn set_native_selection(input: &mut MultilineCallbackData, selection: Range<usize>, caret: usize) {
    if let (Ok(start), Ok(end)) = (i32::try_from(selection.start), i32::try_from(selection.end)) {
        input.set_cursor_pos(caret);
        *input.selection_start_mut() = start;
        *input.selection_end_mut() = end;
    }
}

struct SelectionCallback<'a>(&'a mut TextContentEditor);
impl SelectionCallback<'_> {
    fn on_always(&mut self, input: &mut MultilineCallbackData) {
        if let Some(range) = self
            .0
            .restore
            .take()
            .filter(|range| input.str().get(range.clone()).is_some())
        {
            let caret = range.end;
            set_native_selection(input, range, caret);
        }
        self.0.selection = input.selection();
        self.0.cursor = input.cursor_pos();
    }
}

fn numbering_choice(ui: &Ui) -> Option<NumberingStyle> {
    ui.set_next_item_width(CONTENT_PICKER_WIDTH);
    let _list = ui.begin_combo("##text-numbering", "Numbering")?;
    NumberingStyle::CHOICES
        .into_iter()
        .find(|style| crate::widget_abi::selectable(ui, style.label(), false, false))
}

fn symbol_choice(ui: &Ui) -> Option<&'static str> {
    ui.set_next_item_width(CONTENT_PICKER_WIDTH);
    let _symbols = ui.begin_combo("##text-symbols", "Symbols")?;
    [("Diameter", "⌀"), ("Degree", "°"), ("Plus or minus", "±")]
        .into_iter()
        .find_map(|(name, symbol)| {
            crate::widget_abi::selectable(ui, name, false, false).then_some(symbol)
        })
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
