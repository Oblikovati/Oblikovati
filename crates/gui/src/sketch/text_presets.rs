// SPDX-License-Identifier: GPL-2.0-only
//! Session presets outlive any one Format Text draft.
use super::{GuiSketchTextDraft, GuiSketchTextPresetId};
use imgui::{DrawListMut, Key, StyleColor, Ui};

const NONE: &str = "No Preset";
const LAST: &str = "Last Used";

#[derive(Clone, Default, PartialEq, Eq)]
enum PresetChoice {
    #[default]
    None,
    LastUsed,
    Named(GuiSketchTextPresetId),
}

#[derive(Default)]
pub(super) struct TextPresets {
    entries: Vec<PresetEntry>,
    next_id: u64,
    current: PresetChoice,
    initial: PresetChoice,
    last: Option<AcceptedPreset>,
    submitted: Option<AcceptedPreset>,
    alphabetical: bool,
    name: Option<PresetName>,
    open: bool,
    starting: bool,
}

struct PresetEntry {
    id: GuiSketchTextPresetId,
    name: String,
    fields: GuiSketchTextDraft,
}

struct AcceptedPreset {
    fields: GuiSketchTextDraft,
    origin: Option<GuiSketchTextPresetId>,
}

struct PresetName {
    value: String,
    rename: bool,
    focus: bool,
}

impl TextPresets {
    pub(super) fn synchronize(&mut self, open: bool) {
        if open && !self.open {
            self.starting = true;
        } else if !open && self.open {
            if let Some(committed) = self.submitted.take() {
                self.last = Some(committed);
            }
            self.name = None;
            self.starting = false;
        } else if open {
            // The owner keeps the draft open after a refused submission.
            self.submitted = None;
        }
        self.open = open;
    }

    pub(super) fn begin(
        &mut self,
        existing: bool,
        preset: Option<GuiSketchTextPresetId>,
        draft: &mut GuiSketchTextDraft,
    ) -> bool {
        if !std::mem::take(&mut self.starting) {
            return false;
        }
        self.current = if existing {
            preset
                .filter(|id| self.entries.iter().any(|entry| entry.id == *id))
                .map_or(PresetChoice::None, PresetChoice::Named)
        } else {
            self.initial.clone()
        };
        !existing && self.apply(draft)
    }

    pub(super) fn selected_preset(&self) -> Option<GuiSketchTextPresetId> {
        self.associated_entry().map(|entry| entry.id)
    }

    fn associated_entry(&self) -> Option<&PresetEntry> {
        let origin = match self.current {
            PresetChoice::Named(id) => Some(id),
            PresetChoice::LastUsed => self.last.as_ref().and_then(|last| last.origin),
            PresetChoice::None => None,
        }?;
        self.entry(origin)
    }

    pub(super) fn submit(&mut self, draft: &GuiSketchTextDraft) {
        self.submitted = (!draft.content.trim().is_empty()).then(|| AcceptedPreset {
            fields: draft.clone(),
            origin: self.selected_preset(),
        });
    }

    pub(super) fn cancel(&mut self) {
        self.submitted = None;
    }

    pub(super) fn draw(&mut self, ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
        let changed = self.draw_choices(ui, draft);
        ui.same_line();
        if ui.button("+##create-text-preset") {
            self.name = Some(PresetName {
                value: suggest_name(&draft.content),
                rename: false,
                focus: true,
            });
        }
        ui.same_line();
        if ui.button("Preset settings") {
            ui.open_popup("##text-preset-settings");
        }
        if let Some(_settings) = ui.begin_popup("##text-preset-settings") {
            self.draw_settings(ui, draft);
        }
        changed
    }

    fn draw_choices(&mut self, ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
        ui.set_next_item_width(145.);
        let Some(_list) = self.begin_selector(ui, draft) else {
            return false;
        };
        if ui
            .selectable_config(NONE)
            .selected(self.current == PresetChoice::None)
            .build()
        {
            self.current = PresetChoice::None;
        } else if ui
            .selectable_config(LAST)
            .selected(self.current == PresetChoice::LastUsed)
            .disabled(self.last.is_none())
            .build()
        {
            self.current = PresetChoice::LastUsed;
            return self.apply(draft);
        }
        if let Some(id) = self.choose_named(ui) {
            self.current = PresetChoice::Named(id);
            return self.apply(draft);
        }
        false
    }

    fn choose_named(&self, ui: &Ui) -> Option<GuiSketchTextPresetId> {
        let mut entries: Vec<_> = self.entries.iter().rev().collect();
        if self.alphabetical {
            entries.sort_by_cached_key(|entry| entry.name.to_lowercase());
        }
        entries
            .into_iter()
            .find(|entry| {
                ui.selectable_config(&entry.name)
                    .selected(self.current == PresetChoice::Named(entry.id))
                    .build()
            })
            .map(|entry| entry.id)
    }

    fn caption(&self, draft: &GuiSketchTextDraft) -> String {
        let Some(entry) = self.associated_entry() else {
            return if self.current == PresetChoice::LastUsed {
                LAST.into()
            } else {
                NONE.into()
            };
        };
        if &entry.fields == draft {
            entry.name.clone()
        } else {
            format!("*{}", entry.name)
        }
    }

    fn begin_selector<'ui>(
        &self,
        ui: &'ui Ui,
        draft: &GuiSketchTextDraft,
    ) -> Option<imgui::ComboBoxToken<'ui>> {
        let caption = self.caption(draft);
        if self.current != PresetChoice::LastUsed {
            return ui.begin_combo("##text-preset-selector", caption);
        }
        // Reserve caption space so the marker does not depend on a clock glyph in the UI font.
        let gutter = "   ";
        let [x, y] = ui.cursor_screen_pos();
        let center = [
            x + ui.calc_text_size(gutter)[0] * 0.5 + 2.,
            y + ui.frame_height() * 0.5,
        ];
        let draw = ui.get_window_draw_list();
        let combo = ui.begin_combo("##text-preset-selector", format!("{gutter}{caption}"));
        draw_history_clock(ui, &draw, center);
        combo
    }

    fn fields(&self, choice: &PresetChoice) -> Option<&GuiSketchTextDraft> {
        match choice {
            PresetChoice::None => None,
            PresetChoice::LastUsed => self.last.as_ref().map(|last| &last.fields),
            PresetChoice::Named(id) => self.entry(*id).map(|entry| &entry.fields),
        }
    }

    fn apply(&mut self, draft: &mut GuiSketchTextDraft) -> bool {
        let Some(fields) = self.fields(&self.current) else {
            return false;
        };
        draft.clone_from(fields);
        if let Some(index) = self.current_index() {
            let used = self.entries.remove(index);
            self.entries.push(used);
        }
        true
    }

    fn draw_settings(&mut self, ui: &Ui, draft: &GuiSketchTextDraft) {
        self.draw_current_actions(ui, draft);
        ui.separator();
        ui.text("Sort Order");
        if checked_preset_option(ui, "Recently Used", !self.alphabetical) {
            self.alphabetical = false;
        }
        if checked_preset_option(ui, "Alphabetical", self.alphabetical) {
            self.alphabetical = true;
        }
        ui.separator();
        ui.text("New Text Default");
        self.draw_default_actions(ui);
    }

    fn draw_current_actions(&mut self, ui: &Ui, draft: &GuiSketchTextDraft) {
        let index = self.current_index();
        if ui
            .menu_item_config("Save Current")
            .enabled(index.is_some())
            .build()
        {
            self.entries[index.unwrap()].fields.clone_from(draft);
        }
        if ui
            .menu_item_config("Rename Current")
            .enabled(index.is_some())
            .build()
        {
            self.name = Some(PresetName {
                value: self.entries[index.unwrap()].name.clone(),
                rename: true,
                focus: true,
            });
        }
        if ui
            .menu_item_config("Delete Current")
            .enabled(index.is_some())
            .build()
        {
            let entry = self.entries.remove(index.unwrap());
            if self.initial == PresetChoice::Named(entry.id) {
                self.initial = PresetChoice::None;
            }
            self.current = PresetChoice::None;
        }
    }

    fn draw_default_actions(&mut self, ui: &Ui) {
        if checked_preset_option(ui, NONE, self.initial == PresetChoice::None) {
            self.initial = PresetChoice::None;
            ui.close_current_popup();
        }
        if checked_preset_option(ui, LAST, self.initial == PresetChoice::LastUsed) {
            self.initial = PresetChoice::LastUsed;
            ui.close_current_popup();
        }
        let current = matches!(self.current, PresetChoice::Named(_));
        if ui
            .menu_item_config("Use Current Preset")
            .enabled(current)
            .selected(current && self.initial == self.current)
            .build()
        {
            self.initial = self.current.clone();
        }
    }

    pub(super) fn draw_name(&mut self, ui: &Ui, draft: &GuiSketchTextDraft) -> bool {
        let Some(name) = self.name.as_mut() else {
            return false;
        };
        if ui.is_key_pressed_no_repeat(Key::Escape) {
            self.name = None;
            return true;
        }
        if std::mem::take(&mut name.focus) {
            ui.set_keyboard_focus_here();
        }
        let entered = ui
            .input_text("Preset name", &mut name.value)
            .auto_select_all(true)
            .enter_returns_true(true)
            .build();
        let value = name.value.trim().to_owned();
        if entered && self.save_name(&value, draft) {
            self.name = None;
        } else if entered {
            ui.text("Enter a unique preset name.");
        }
        false
    }

    fn save_name(&mut self, value: &str, draft: &GuiSketchTextDraft) -> bool {
        let rename = self.name.as_ref().is_some_and(|name| name.rename);
        if value.is_empty()
            || value.contains('#')
            || [NONE, LAST]
                .iter()
                .any(|reserved| reserved.eq_ignore_ascii_case(value))
        {
            return false;
        }
        if self
            .entries
            .iter()
            .any(|entry| entry.name.eq_ignore_ascii_case(value))
        {
            return false;
        }
        if let Some(index) = self.current_index().filter(|_| rename) {
            self.entries[index].name = value.into();
        } else {
            let Some(next) = self.next_id.checked_add(1) else {
                return false;
            };
            self.next_id = next;
            let id = GuiSketchTextPresetId(next);
            self.entries.push(PresetEntry {
                id,
                name: value.into(),
                fields: draft.clone(),
            });
            self.current = PresetChoice::Named(id);
        }
        true
    }

    fn current_index(&self) -> Option<usize> {
        let PresetChoice::Named(id) = self.current else {
            return None;
        };
        self.entries.iter().position(|entry| entry.id == id)
    }

    fn entry(&self, id: GuiSketchTextPresetId) -> Option<&PresetEntry> {
        self.entries.iter().find(|entry| entry.id == id)
    }
}

fn draw_history_clock(ui: &Ui, draw: &DrawListMut<'_>, at: [f32; 2]) {
    let color = ui.style_color(StyleColor::Text);
    let radius = ui.current_font_size() / 3.;
    draw.add_circle(at, radius, color).build();
    draw.add_line(at, [at[0], at[1] - radius * 0.65], color)
        .build();
    draw.add_line(at, [at[0] + radius * 0.55, at[1]], color)
        .build();
}

fn checked_preset_option(ui: &Ui, label: &str, selected: bool) -> bool {
    let mut checked = selected;
    ui.checkbox(label, &mut checked) && checked
}

fn suggest_name(content: &str) -> String {
    let suggested: String = content
        .lines()
        .next()
        .unwrap_or_default()
        .trim()
        .chars()
        .filter(|&ch| ch != '#')
        .take(24)
        .collect();
    if suggested.is_empty() {
        "Preset".into()
    } else {
        suggested
    }
}

#[cfg(test)]
#[path = "text_presets_tests.rs"]
mod tests;
