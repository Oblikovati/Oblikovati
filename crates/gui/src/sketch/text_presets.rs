// SPDX-License-Identifier: GPL-2.0-only
//! Session presets outlive any one Format Text draft.
use super::{GuiSketchTextDraft, GuiSketchTextPresetId};
use imgui::{Key, Ui};

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
    last: Option<GuiSketchTextDraft>,
    submitted: Option<GuiSketchTextDraft>,
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
        match self.current {
            PresetChoice::Named(id) => Some(id),
            _ => None,
        }
    }

    pub(super) fn submit(&mut self, draft: &GuiSketchTextDraft) {
        self.submitted = (!draft.content.trim().is_empty()).then(|| draft.clone());
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
        let caption = self.caption(draft);
        ui.set_next_item_width(145.);
        let Some(_list) = ui.begin_combo("##text-preset-selector", &caption) else {
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
        match &self.current {
            PresetChoice::None => NONE.into(),
            PresetChoice::LastUsed => LAST.into(),
            PresetChoice::Named(_) => self.current_index().map_or_else(
                || NONE.into(),
                |index| {
                    let entry = &self.entries[index];
                    if &entry.fields == draft {
                        entry.name.clone()
                    } else {
                        format!("*{}", entry.name)
                    }
                },
            ),
        }
    }

    fn fields(&self, choice: &PresetChoice) -> Option<&GuiSketchTextDraft> {
        match choice {
            PresetChoice::None => None,
            PresetChoice::LastUsed => self.last.as_ref(),
            PresetChoice::Named(id) => self
                .entries
                .iter()
                .find(|entry| entry.id == *id)
                .map(|entry| &entry.fields),
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
        }
        if checked_preset_option(ui, LAST, self.initial == PresetChoice::LastUsed) {
            self.initial = PresetChoice::LastUsed;
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
        let id = self.selected_preset()?;
        self.entries.iter().position(|entry| entry.id == id)
    }
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
mod tests {
    use super::*;
    #[test]
    fn saved_preset_can_apply_to_a_new_draft() {
        let saved = GuiSketchTextDraft {
            content: "ab".into(),
            ..Default::default()
        };
        let mut shelf = TextPresets::default();
        assert!(shelf.save_name(&suggest_name(&saved.content), &saved));
        assert!(!shelf.save_name("AB", &saved));
        shelf.initial = shelf.current.clone();
        shelf.synchronize(true);
        let mut next = GuiSketchTextDraft::default();
        assert!(shelf.begin(false, None, &mut next));
        assert_eq!(next, saved);
    }

    #[test]
    fn last_used_requires_a_submitted_draft_and_survives_later_cancel() {
        let draft = GuiSketchTextDraft {
            content: "ab".into(),
            ..Default::default()
        };
        let mut shelf = TextPresets::default();
        shelf.synchronize(true);
        shelf.submit(&draft);
        shelf.synchronize(false);
        shelf.synchronize(true);
        shelf.submit(&GuiSketchTextDraft::default());
        shelf.cancel();
        shelf.synchronize(false);
        assert_eq!(shelf.fields(&PresetChoice::LastUsed), Some(&draft));
    }

    #[test]
    fn reopened_modified_text_keeps_its_preset_through_rename() {
        let mut shelf = TextPresets::default();
        let original = GuiSketchTextDraft {
            content: "ab".into(),
            ..Default::default()
        };
        assert!(shelf.save_name("f", &original));
        let preset = shelf.selected_preset();
        let mut changed = original.clone();
        changed.content = "c".into();
        shelf.synchronize(true);
        assert!(!shelf.begin(true, preset, &mut changed));
        assert_eq!(shelf.caption(&changed), "*f");
        shelf.name = Some(PresetName {
            value: "g".into(),
            rename: true,
            focus: false,
        });
        assert!(shelf.save_name("g", &changed));
        assert_eq!(shelf.selected_preset(), preset);
        assert_eq!(shelf.caption(&changed), "*g");
        assert_eq!(shelf.fields(&shelf.current), Some(&original));
    }

    #[test]
    fn matching_content_does_not_assign_a_preset_to_unrelated_text() {
        let mut shelf = TextPresets::default();
        let mut draft = GuiSketchTextDraft {
            content: "ab".into(),
            ..Default::default()
        };
        assert!(shelf.save_name("f", &draft));
        shelf.synchronize(true);
        assert!(!shelf.begin(true, None, &mut draft));
        assert_eq!(shelf.caption(&draft), NONE);
    }
}
