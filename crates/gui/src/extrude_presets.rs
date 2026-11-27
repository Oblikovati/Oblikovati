// SPDX-License-Identifier: GPL-2.0-only
//! Numeric Extrude settings reused within the desktop session.
use super::{ExtrudeDialog, GuiCommandState, GuiExtrudeDirection};
use crate::{
    GuiExtrudePreview,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
};
use imgui::Ui;

const NONE: &str = "No Preset";
const SETTINGS: &str = "Preset Settings";

#[derive(Clone)]
struct Fields {
    distance: String,
    second_distance: Option<String>,
    direction: GuiExtrudeDirection,
}

#[derive(Clone, Copy, Default, PartialEq, Eq)]
enum SortOrder {
    #[default]
    Recent,
    Created,
    Alphabetical,
    Modified,
}

#[derive(Clone, Default, PartialEq, Eq)]
enum NewDefault {
    #[default]
    None,
    LastUsed,
    Named(usize),
}

struct Preset {
    name: String,
    fields: Fields,
    recent: u64,
    modified: u64,
}

struct Naming {
    name: String,
    rename: bool,
    focus: bool,
}

#[derive(Default)]
pub(super) struct ExtrudePresets {
    shown: bool,
    entries: Vec<Option<Preset>>,
    selected: Option<usize>,
    sort: SortOrder,
    initial: NewDefault,
    last: Option<Fields>,
    pending: Option<(u64, u64, Fields)>,
    naming: Option<Naming>,
    order: u64,
}

impl ExtrudePresets {
    pub(super) fn draw_visibility_setting(&mut self, ui: &Ui) {
        if checked_item(ui, "Hide Presets", !self.shown, true) {
            self.shown = !self.shown;
        }
    }

    pub(super) fn clear_selection(&mut self) {
        self.selected = None;
        self.naming = None;
    }

    pub(super) fn submit(&mut self, draft: &GuiExtrudePreview) {
        self.pending = draft
            .generation
            .zip(Fields::from_preview(draft))
            .map(|(generation, fields)| (draft.document, generation, fields));
    }

    pub(super) fn reject_submission(&mut self) {
        self.pending = None;
    }

    pub(super) fn observe_command_state(&mut self, state: &GuiCommandState) {
        if let Some((document, generation, fields)) = self.pending.take()
            && state.document == Some(document)
            && state
                .document_generation
                .is_some_and(|current| current > generation)
        {
            self.last = Some(fields);
        }
    }

    fn open(&mut self) -> Option<Fields> {
        self.clear_selection();
        match self.initial {
            NewDefault::None => None,
            NewDefault::LastUsed => self.last.clone(),
            NewDefault::Named(index) => self.select(index),
        }
    }

    fn select(&mut self, index: usize) -> Option<Fields> {
        let preset = self.entries.get_mut(index)?.as_mut()?;
        self.order += 1;
        preset.recent = self.order;
        self.selected = Some(index);
        Some(preset.fields.clone())
    }

    fn current(&self) -> Option<&Preset> {
        self.selected.and_then(|index| self.entries[index].as_ref())
    }

    fn draw(&mut self, ui: &Ui, icons: &GuiRibbonIcons, fields: Option<Fields>) -> Option<Fields> {
        let size = icons.logical[IconSize::Small.index()];
        let spacing = ui.clone_style().item_spacing[0];
        ui.set_next_item_width(
            (ui.content_region_avail()[0] - 2. * (ui.frame_height() + spacing)).max(1.),
        );
        let chosen = self.draw_choices(ui);
        ui.same_line();
        let disabled = ui.begin_disabled(fields.is_none());
        if ui.button_with_size("+##extrude-preset-add", [ui.frame_height(); 2]) {
            self.naming = Some(Naming {
                name: String::new(),
                rename: false,
                focus: true,
            });
        }
        if ui.is_item_hovered() {
            ui.tooltip_text("Add Extrude Preset");
        }
        drop(disabled);
        ui.same_line();
        let button = crate::widget_abi::IconButtonPresentation {
            texture: icons.key(IconGlyph::Preferences, IconSize::Small),
            size: [ui.frame_height(); 2],
            icon: size,
            large: false,
            show_caption: false,
        };
        if crate::widget_abi::icon_button(ui, SETTINGS, button) {
            ui.open_popup(SETTINGS);
        }
        if ui.is_item_hovered() {
            ui.tooltip_text(SETTINGS);
        }
        if let Some(_menu) = ui.begin_popup(SETTINGS) {
            self.draw_settings(ui, fields.as_ref());
        }
        self.draw_name(ui, fields);
        chosen
    }

    fn draw_choices(&mut self, ui: &Ui) -> Option<Fields> {
        let _popup = ui.begin_combo(
            "##extrude-preset",
            self.current().map_or(NONE, |entry| &entry.name),
        )?;
        if ui
            .selectable_config(NONE)
            .selected(self.selected.is_none())
            .build()
        {
            self.selected = None;
        }
        let mut choices: Vec<_> = self
            .entries
            .iter()
            .enumerate()
            .filter_map(|(index, entry)| entry.as_ref().map(|entry| (index, entry)))
            .collect();
        match self.sort {
            SortOrder::Recent => choices.sort_by_key(|(_, entry)| std::cmp::Reverse(entry.recent)),
            SortOrder::Created => {}
            SortOrder::Alphabetical => {
                choices.sort_by_cached_key(|(_, entry)| entry.name.to_lowercase())
            }
            SortOrder::Modified => {
                choices.sort_by_key(|(_, entry)| std::cmp::Reverse(entry.modified))
            }
        }
        let selected = choices
            .iter()
            .find(|(index, entry)| {
                ui.selectable_config(&entry.name)
                    .selected(self.selected == Some(*index))
                    .build()
            })
            .map(|(index, _)| *index)?;
        self.select(selected)
    }

    fn draw_settings(&mut self, ui: &Ui, fields: Option<&Fields>) {
        self.draw_management(ui, fields);
        ui.separator();
        ui.text("Sort Order");
        for (choice, label) in [
            (SortOrder::Recent, "Recently Used"),
            (SortOrder::Created, "Date Created"),
            (SortOrder::Alphabetical, "Alphanumeric"),
            (SortOrder::Modified, "Last Modified"),
        ] {
            if checked_item(ui, label, self.sort == choice, true) {
                self.sort = choice;
            }
        }
        ui.separator();
        ui.text("New Command Default");
        self.draw_defaults(ui);
    }

    fn draw_management(&mut self, ui: &Ui, fields: Option<&Fields>) {
        let index = self.selected;
        if checked_item(
            ui,
            "Save Current",
            false,
            index.is_some() && fields.is_some(),
        ) && let (Some(index), Some(fields)) = (index, fields)
        {
            self.order += 1;
            if let Some(entry) = &mut self.entries[index] {
                entry.fields.clone_from(fields);
                entry.modified = self.order;
            }
        }
        if checked_item(ui, "Rename Current", false, index.is_some())
            && let Some(entry) = self.current()
        {
            self.naming = Some(Naming {
                name: entry.name.clone(),
                rename: true,
                focus: true,
            });
        }
        if checked_item(ui, "Delete Current", false, index.is_some())
            && let Some(index) = index
        {
            self.entries[index] = None;
            self.selected = None;
            if self.initial == NewDefault::Named(index) {
                self.initial = NewDefault::None;
            }
        }
    }

    fn draw_defaults(&mut self, ui: &Ui) {
        if checked_item(ui, "Last Used", self.initial == NewDefault::LastUsed, true) {
            self.initial = NewDefault::LastUsed;
        }
        let selected = self.selected.map(NewDefault::Named);
        if checked_item(
            ui,
            "Use Current Preset",
            selected.as_ref() == Some(&self.initial),
            selected.is_some(),
        ) && let Some(selected) = selected
        {
            self.initial = selected;
        }
        if checked_item(ui, NONE, self.initial == NewDefault::None, true) {
            self.initial = NewDefault::None;
        }
    }

    fn draw_name(&mut self, ui: &Ui, fields: Option<Fields>) {
        let Some(mut naming) = self.naming.take() else {
            return;
        };
        if std::mem::take(&mut naming.focus) {
            ui.set_keyboard_focus_here();
        }
        ui.input_text("Preset name", &mut naming.name).build();
        let name = naming.name.trim();
        let duplicate = self.entries.iter().enumerate().any(|(index, entry)| {
            !(naming.rename && Some(index) == self.selected)
                && entry
                    .as_ref()
                    .is_some_and(|entry| entry.name.eq_ignore_ascii_case(name))
        });
        let valid = !name.is_empty() && !duplicate && !name.eq_ignore_ascii_case(NONE);
        let disabled = ui.begin_disabled(!valid || (!naming.rename && fields.is_none()));
        let save = ui.button("Save preset");
        drop(disabled);
        ui.same_line();
        if save {
            self.save_name(name, naming.rename, fields);
        } else if !ui.button("Cancel preset") {
            self.naming = Some(naming);
        }
    }

    fn save_name(&mut self, name: &str, rename: bool, fields: Option<Fields>) {
        self.order += 1;
        if rename {
            if let Some(entry) = self.selected.and_then(|index| self.entries[index].as_mut()) {
                entry.name = name.to_owned();
                entry.modified = self.order;
            }
        } else if let Some(fields) = fields {
            self.selected = Some(self.entries.len());
            self.entries.push(Some(Preset {
                name: name.to_owned(),
                fields,
                recent: self.order,
                modified: self.order,
            }));
        }
    }
}

fn checked_item(ui: &Ui, caption: &str, checked: bool, enabled: bool) -> bool {
    ui.menu_item_config(caption)
        .selected(checked)
        .enabled(enabled)
        .build()
}

impl Fields {
    fn from_preview(draft: &GuiExtrudePreview) -> Option<Self> {
        draft.reference_extent.is_none().then(|| Self {
            distance: draft.distance.clone(),
            second_distance: draft.second_distance.clone(),
            direction: draft.direction,
        })
    }
}

impl ExtrudeDialog {
    pub(super) fn draw_presets(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        icons: &GuiRibbonIcons,
    ) -> bool {
        if !self.presets.shown {
            return false;
        }
        let fields = state
            .extrude_draft_preview
            .as_ref()
            .filter(|draft| {
                self.edit_feature.is_none()
                    && self.reference.numeric()
                    && !self.reference.needs_start_plane()
                    && Some(draft.document) == self.document
                    && draft.distance == self.distance
                    && draft.direction == self.direction
                    && self.active_second_distance() == draft.second_distance.as_deref()
            })
            .and_then(Fields::from_preview);
        let Some(preset) = self.presets.draw(ui, icons, fields) else {
            return false;
        };
        self.apply_preset(preset);
        true
    }

    pub(super) fn seed_preset(&mut self) {
        if let Some(preset) = self.presets.open() {
            self.apply_preset(preset);
        }
    }

    fn apply_preset(&mut self, preset: Fields) {
        self.distance = preset.distance;
        if let Some(distance) = preset.second_distance {
            self.second_distance = distance;
        }
        self.direction = preset.direction;
        self.reference = Default::default();
    }
}
