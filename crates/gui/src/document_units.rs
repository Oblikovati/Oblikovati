// SPDX-License-Identifier: GPL-2.0-only
//! Document unit controls send partial updates through the public units contract.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, TabItemFlags, Ui};
use oblikovati_api::wire::DocumentUnitsInfo;

#[derive(Default)]
pub(crate) struct GuiDocumentUnits {
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    draft: Option<crate::document_units_draft::UnitDraft>,
    select_standard: bool,
}
impl DockableWindow for GuiDocumentUnits {}
impl GuiDocumentUnits {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Document Settings", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.draft = None;
        self.select_standard = document.is_some();
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn request(&mut self, document: u64) {
        if self.document == Some(document) {
            return;
        }
        self.set_document(Some(document));
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, home_active: bool) {
        if self.document != state.document || home_active {
            self.set_document(None);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        home_active: bool,
    ) {
        self.sync_before_docking(state, home_active);
        let (Some(document), Some(units)) = (self.document, state.document_units.as_ref()) else {
            return;
        };
        let draft = self
            .draft
            .get_or_insert_with(|| crate::document_units_draft::UnitDraft::new(units));
        draft.sync(units);
        Self::center_dockable(ui, "Document Settings", [580., 360.]);
        let mut open = true;
        let mut done = false;
        Self::dockable_window(ui, "Document Settings", &mut open).build(|| {
            settings_tabs(ui, state, draft, &mut self.select_standard);
            if draft.stale(units) {
                ui.text_colored(
                    [1., 0.3, 0.3, 1.],
                    "Settings changed elsewhere. Cancel and reopen.",
                );
            }
            if let Some(error) = &draft.error {
                ui.text_colored([1., 0.3, 0.3, 1.], error);
            }
            ui.separator();
            done = unit_actions(ui, document, units, draft, actions, state.transaction_open);
            done |= ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape);
        });
        if !open || done {
            self.set_document(None);
        }
    }
    pub(crate) fn acknowledge(&mut self, document: u64, result: Result<(), String>) {
        if self.document == Some(document)
            && let Some(draft) = &mut self.draft
        {
            draft.acknowledge(result);
        }
    }
}
fn settings_tabs(
    ui: &Ui,
    state: &GuiCommandState,
    draft: &mut crate::document_units_draft::UnitDraft,
    select_standard: &mut bool,
) {
    let Some(_tabs) = ui.tab_bar("##document-settings-pages") else {
        return;
    };
    let flags = if std::mem::take(select_standard) {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    if let Some(_standard) = ui.tab_item_with_flags("Standard", None, flags) {
        crate::ribbon_part_assets::document_asset_summary(ui, state);
    }
    if let Some(_units) = ui.tab_item("Units") {
        unit_page(ui, draft, state.transaction_open);
    }
    unavailable_settings_tabs(ui);
}

fn unavailable_settings_tabs(ui: &Ui) {
    let _disabled = ui.begin_disabled(true);
    for label in [
        "Sketch",
        "Modeling",
        "Bill of Materials",
        "Default Tolerance",
    ] {
        let _tab = ui.tab_item(label);
    }
}

fn unit_page(ui: &Ui, draft: &mut crate::document_units_draft::UnitDraft, transaction_open: bool) {
    let disabled = ui.begin_disabled(transaction_open || draft.pending());
    unit_controls(ui, &mut draft.value);
    precision_controls(ui, &mut draft.value);
    drop(disabled);
    ui.text_disabled("Angle as DMS: unavailable");
}

fn unit_controls(ui: &Ui, units: &mut DocumentUnitsInfo) {
    edit_unit(
        ui,
        "Length",
        &mut units.length_unit,
        &["µm", "mm", "cm", "m", "in", "ft"],
    );
    edit_unit(ui, "Angle", &mut units.angle_unit, &["deg", "rad"]);
    edit_unit(ui, "Mass", &mut units.mass_unit, &["kg", "g", "lb"]);
    edit_unit(ui, "Time", &mut units.time_unit, &["s", "ms", "min", "hr"]);
    edit_unit(
        ui,
        "Length format",
        &mut units.length_display_format,
        &["decimal", "fractional", "architectural"],
    );
}
fn edit_unit(ui: &Ui, label: &str, value: &mut String, options: &[&str]) {
    if let Some(selected) = select_unit(ui, label, value, options) {
        *value = selected.to_string();
    }
}
fn select_unit<'a>(ui: &Ui, label: &str, current: &str, options: &'a [&str]) -> Option<&'a str> {
    let _combo = ui.begin_combo(label, current)?;
    options.iter().copied().find(|name| {
        ui.selectable_config(name)
            .selected(*name == current)
            .build()
            && *name != current
    })
}
fn precision_controls(ui: &Ui, units: &mut DocumentUnitsInfo) {
    for (label, current) in [
        ("Length precision", &mut units.length_display_precision),
        ("Angle precision", &mut units.angle_display_precision),
    ] {
        let Ok(mut value) = i32::try_from(*current) else {
            ui.text_disabled(format!("{label}: {current}"));
            continue;
        };
        if ui.input_int(label, &mut value).build() {
            *current = i64::from(value);
        }
    }
}
fn unit_actions(
    ui: &Ui,
    document: u64,
    units: &DocumentUnitsInfo,
    draft: &mut crate::document_units_draft::UnitDraft,
    actions: &mut Vec<GuiShellAction>,
    transaction_open: bool,
) -> bool {
    let close_disabled = ui.begin_disabled(draft.dirty() || draft.pending());
    let close = crate::widget_abi::button(ui, "Close", [0.; 2]);
    drop(close_disabled);
    ui.same_line();
    let cancel_disabled = ui.begin_disabled(!draft.dirty() || draft.pending());
    let cancel = crate::widget_abi::button(ui, "Cancel", [0.; 2]);
    drop(cancel_disabled);
    ui.same_line();
    let apply_disabled = ui.begin_disabled(transaction_open || !draft.ready(units));
    let apply = crate::widget_abi::button(ui, "Apply", [0.; 2]);
    drop(apply_disabled);
    if apply && let Some(request) = draft.submit(units) {
        actions.push(GuiShellAction::SetDocumentUnits { document, request });
    }
    close || cancel
}

#[cfg(test)]
mod dock_tests {
    use super::*;
    #[test]
    fn document_change_closes_settings_before_docking() {
        let mut dialog = GuiDocumentUnits::default();
        dialog.request(7);
        dialog.sync_before_docking(&GuiCommandState::default(), false);
        assert!(dialog.document.is_none());
        dialog.request(7);
        let state = GuiCommandState {
            document: Some(7),
            ..Default::default()
        };
        dialog.sync_before_docking(&state, true);
        assert!(dialog.document.is_none());
    }
    #[test]
    fn reopening_same_document_preserves_draft() {
        let mut dialog = GuiDocumentUnits::default();
        dialog.request(7);
        let units = DocumentUnitsInfo::default();
        let mut draft = crate::document_units_draft::UnitDraft::new(&units);
        draft.value.length_unit = "cm".into();
        dialog.draft = Some(draft);
        dialog.request(7);
        assert_eq!(dialog.draft.unwrap().value.length_unit, "cm");
    }
}
