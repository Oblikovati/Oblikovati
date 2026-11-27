// SPDX-License-Identifier: GPL-2.0-only
//! Generic session-scoped feature preset bar (`@inherits_common_feature_properties_presets`).
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::widget_abi::{self, IconButtonPresentation};
use imgui::Ui;

const NO_PRESET: &str = "No Preset";

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

struct PresetEntry<F> {
    name: String,
    fields: F,
    recent: u64,
    modified: u64,
}

/// Reusable preset row and management menu for any Properties-based Part command.
pub(crate) struct SessionPresets<F> {
    pub(crate) shown: bool,
    entries: Vec<Option<PresetEntry<F>>>,
    selected: Option<usize>,
    sort: SortOrder,
    initial: NewDefault,
    last: Option<F>,
    order: u64,
}

impl<F> Default for SessionPresets<F> {
    fn default() -> Self {
        Self {
            shown: false,
            entries: Vec::new(),
            selected: None,
            sort: SortOrder::Recent,
            initial: NewDefault::None,
            last: None,
            order: 0,
        }
    }
}

impl<F: Clone> SessionPresets<F> {
    pub(crate) fn clear_selection(&mut self) {
        self.selected = None;
    }

    pub(crate) fn record_accepted(&mut self, fields: F) {
        self.last = Some(fields);
    }

    pub(crate) fn initial_fields(&mut self) -> Option<F> {
        self.selected = None;
        match self.initial {
            NewDefault::None => None,
            NewDefault::LastUsed => self.last.clone(),
            NewDefault::Named(index) => self.select(index),
        }
    }

    fn select(&mut self, index: usize) -> Option<F> {
        let entry = self.entries.get_mut(index)?.as_mut()?;
        self.order += 1;
        entry.recent = self.order;
        self.selected = Some(index);
        Some(entry.fields.clone())
    }

    /// Draw the preset selector, `+` save button, and management gear menu.
    /// Returns chosen preset fields when the user picks a saved preset.
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        id_prefix: &str,
        new_default_label: &str,
        current_fields: Option<F>,
        default_name: &str,
    ) -> Option<F> {
        if !self.shown {
            return None;
        }
        let icon = icons.logical[IconSize::Small.index()];
        let button_size = [icon[0] + 10., icon[1] + 6.];
        let spacing = ui.clone_style().item_spacing[0];
        let combo_width = (ui.content_region_avail()[0] - 2. * (button_size[0] + spacing)).max(40.);
        ui.set_next_item_width(combo_width);
        let preview = self
            .selected
            .and_then(|i| self.entries.get(i)?.as_ref())
            .map(|entry| entry.name.as_str())
            .unwrap_or(NO_PRESET);
        let mut chosen = None;
        let combo_id = format!("##{id_prefix}-preset-select");
        if let Some(_combo) = ui.begin_combo(&combo_id, preview) {
            if ui
                .selectable_config(NO_PRESET)
                .selected(self.selected.is_none())
                .build()
            {
                self.selected = None;
            }
            for index in self.ordered_indices() {
                let name = self.entries[index].as_ref().map(|e| e.name.clone());
                if let Some(name) = name
                    && ui
                        .selectable_config(&name)
                        .selected(self.selected == Some(index))
                        .build()
                {
                    chosen = self.select(index);
                }
            }
        }
        ui.same_line();
        {
            let _disabled = ui.begin_disabled(current_fields.is_none());
            if widget_abi::button(ui, "+", button_size)
                && let Some(fields) = current_fields.clone()
            {
                self.order += 1;
                let index = self.entries.len();
                self.entries.push(Some(PresetEntry {
                    name: default_name.to_owned(),
                    fields,
                    recent: self.order,
                    modified: self.order,
                }));
                self.selected = Some(index);
            }
        }
        ui.same_line();
        let settings_popup = format!("##{id_prefix}-preset-settings");
        let gear = IconButtonPresentation {
            texture: icons.key(IconGlyph::Preferences, IconSize::Small),
            size: button_size,
            icon,
            large: false,
            show_caption: false,
        };
        if widget_abi::icon_button(ui, "Preset Settings", gear) {
            ui.open_popup(&settings_popup);
        }
        self.draw_settings_popup(ui, &settings_popup, new_default_label, current_fields);
        chosen
    }

    fn draw_settings_popup(
        &mut self,
        ui: &Ui,
        popup_id: &str,
        new_default_label: &str,
        current_fields: Option<F>,
    ) {
        let Some(_popup) = ui.begin_popup(popup_id) else {
            return;
        };
        let has_selected = self
            .selected
            .and_then(|i| self.entries.get(i)?.as_ref())
            .is_some();
        if ui
            .menu_item_config("Save Current")
            .enabled(has_selected && current_fields.is_some())
            .build()
            && let (Some(index), Some(fields)) = (self.selected, current_fields)
            && let Some(Some(entry)) = self.entries.get_mut(index)
        {
            self.order += 1;
            entry.fields = fields;
            entry.modified = self.order;
        }
        let _ = ui
            .menu_item_config("Rename Current")
            .enabled(has_selected)
            .build();
        if ui
            .menu_item_config("Delete Current")
            .enabled(has_selected)
            .build()
            && let Some(index) = self.selected.take()
        {
            self.entries[index] = None;
            if self.initial == NewDefault::Named(index) {
                self.initial = NewDefault::None;
            }
        }
        ui.separator();
        if let Some(_menu) = ui.begin_menu("Sort Order") {
            for (order, label) in [
                (SortOrder::Recent, "Recently Used"),
                (SortOrder::Created, "Created"),
                (SortOrder::Alphabetical, "Alphabetical"),
                (SortOrder::Modified, "Modified"),
            ] {
                if ui
                    .menu_item_config(label)
                    .selected(self.sort == order)
                    .build()
                {
                    self.sort = order;
                }
            }
        }
        if let Some(_menu) = ui.begin_menu(new_default_label) {
            if ui
                .menu_item_config(NO_PRESET)
                .selected(self.initial == NewDefault::None)
                .build()
            {
                self.initial = NewDefault::None;
            }
            if ui
                .menu_item_config("Last Used")
                .selected(self.initial == NewDefault::LastUsed)
                .build()
            {
                self.initial = NewDefault::LastUsed;
            }
            for index in self.ordered_indices() {
                if let Some(entry) = &self.entries[index]
                    && ui
                        .menu_item_config(&entry.name)
                        .selected(self.initial == NewDefault::Named(index))
                        .build()
                {
                    self.initial = NewDefault::Named(index);
                }
            }
        }
    }

    fn ordered_indices(&self) -> Vec<usize> {
        let mut indices: Vec<usize> = self
            .entries
            .iter()
            .enumerate()
            .filter_map(|(i, entry)| entry.as_ref().map(|_| i))
            .collect();
        match self.sort {
            SortOrder::Created => {}
            SortOrder::Recent => indices.sort_by(|a, b| {
                let ra = self.entries[*a].as_ref().map_or(0, |e| e.recent);
                let rb = self.entries[*b].as_ref().map_or(0, |e| e.recent);
                rb.cmp(&ra)
            }),
            SortOrder::Modified => indices.sort_by(|a, b| {
                let ma = self.entries[*a].as_ref().map_or(0, |e| e.modified);
                let mb = self.entries[*b].as_ref().map_or(0, |e| e.modified);
                mb.cmp(&ma)
            }),
            SortOrder::Alphabetical => indices.sort_by(|a, b| {
                let na = self.entries[*a].as_ref().map_or("", |e| e.name.as_str());
                let nb = self.entries[*b].as_ref().map_or("", |e| e.name.as_str());
                na.cmp(nb)
            }),
        }
        indices
    }
}
