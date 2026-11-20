// SPDX-License-Identifier: GPL-2.0-only
//! Stage constraint preferences until the user accepts or cancels them.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;
use oblikovati_api::types::*;
use oblikovati_model::application_options::SketchApplicationOptions;

#[path = "constraint_settings_grid.rs"]
mod grid;

const INFERENCE_TYPES: [(&str, SketchInferenceTypes); 9] = [
    ("Horizontal", SKETCH_INFER_HORIZONTAL),
    ("Vertical", SKETCH_INFER_VERTICAL),
    ("Parallel", SKETCH_INFER_PARALLEL),
    ("Perpendicular", SKETCH_INFER_PERPENDICULAR),
    ("Intersection", SKETCH_INFER_INTERSECTION),
    ("Midpoint", SKETCH_INFER_MIDPOINT),
    ("On Curve", SKETCH_INFER_ON_CURVE),
    ("Tangent", SKETCH_INFER_TANGENT),
    ("Coincident", SKETCH_INFER_COINCIDENT),
];
const RELAX_TYPES: [(&str, SketchRelaxTypes); 12] = [
    ("Coincident", SKETCH_RELAX_COINCIDENT),
    ("Tangent", SKETCH_RELAX_TANGENT),
    ("Smooth(G2)", SKETCH_RELAX_SMOOTH),
    ("Symmetric", SKETCH_RELAX_SYMMETRIC),
    ("Collinear", SKETCH_RELAX_COLLINEAR),
    ("Concentric", SKETCH_RELAX_CONCENTRIC),
    ("Horizontal", SKETCH_RELAX_HORIZONTAL),
    ("Vertical", SKETCH_RELAX_VERTICAL),
    ("Parallel", SKETCH_RELAX_PARALLEL),
    ("Perpendicular", SKETCH_RELAX_PERPENDICULAR),
    ("Equal", SKETCH_RELAX_EQUAL),
    ("Fix", SKETCH_RELAX_FIX),
];
const ALL_INFERENCE: u32 = (1 << 9) - 1;
const ALL_RELAX: u32 = (1 << 12) - 1;
const DEFAULT_RELAX: u32 = ALL_RELAX
    & !(SKETCH_RELAX_COINCIDENT.0
        | SKETCH_RELAX_TANGENT.0
        | SKETCH_RELAX_SMOOTH.0
        | SKETCH_RELAX_SYMMETRIC.0);

#[derive(Default)]
pub(crate) struct ConstraintSettingsWindow {
    open: crate::dockable_lifecycle::DockableState,
    draft: Option<(u64, SketchSettings)>,
    original: Option<SketchSettings>,
    input_original: bool,
    input_draft: bool,
    focus: bool,
    request_id: u64,
    pending: Option<(GuiShellAction, bool, bool)>,
    error: Option<String>,
}
impl DockableWindow for ConstraintSettingsWindow {}
impl ConstraintSettingsWindow {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Constraint Settings", &self.open)
    }
    pub(crate) fn request(
        &mut self,
        document: u64,
        settings: &SketchSettings,
        options: &SketchApplicationOptions,
    ) {
        self.pending = None;
        self.error = None;
        self.draft = Some((document, settings.clone()));
        self.original = Some(settings.clone());
        self.input_original = options.create_dimensions_on_value_input;
        self.input_draft = self.input_original;
        self.open.set_open(true);
        self.focus = true;
    }
    pub(crate) fn close(&mut self) {
        self.open.set_open(false);
        self.draft = None;
        self.original = None;
        self.focus = false;
        self.pending = None;
        self.error = None;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        options: &mut SketchApplicationOptions,
        actions: &mut Vec<GuiShellAction>,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        if !self.open.is_open() {
            return;
        }
        if self
            .draft
            .as_ref()
            .is_none_or(|(document, _)| state.document != Some(*document))
        {
            self.close();
            return;
        }
        let conflict = self.original.as_ref() != state.document_sketch_settings.as_ref()
            || self.input_original != options.create_dimensions_on_value_input;
        let (open, result) = self.draw_window(ui, conflict, icons);
        if result.accept {
            self.submit(actions);
        }
        if !open || result.close {
            self.close();
        }
    }
    fn draw_window(
        &mut self,
        ui: &Ui,
        conflict: bool,
        icons: &crate::icons::GuiRibbonIcons,
    ) -> (bool, SettingsOutcome) {
        let Some((_, draft)) = self.draft.as_mut() else {
            return (false, SettingsOutcome::default());
        };
        Self::center_dockable(ui, "Constraint Settings", [500., 520.]);
        let mut open = true;
        let focus = std::mem::take(&mut self.focus);
        let outcome = Self::dockable_window(ui, "Constraint Settings", &mut open)
            .focused(focus)
            .build(|| {
                draw_settings(
                    ui,
                    draft,
                    &mut self.input_draft,
                    conflict,
                    self.pending.is_some(),
                    self.error.as_deref(),
                    icons,
                )
            })
            .unwrap_or_default();
        (open, outcome)
    }
    fn submit(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.pending.is_some() {
            return;
        }
        let (Some((document, draft)), Some(expected)) = (&self.draft, &self.original) else {
            return;
        };
        let Some(request_id) = self.request_id.checked_add(1) else {
            self.error = Some("Constraint settings request ID exhausted.".into());
            return;
        };
        self.request_id = request_id;
        let action = GuiShellAction::SetDocumentSketchSettings {
            request_id,
            document: *document,
            expected: expected.clone(),
            settings: draft.clone(),
        };
        self.error = None;
        self.pending = Some((action.clone(), self.input_original, self.input_draft));
        actions.push(action);
    }
    pub(crate) fn validate_pending(
        &self,
        action: &GuiShellAction,
        options: &SketchApplicationOptions,
    ) -> Result<(), String> {
        let Some((pending, original, _)) = &self.pending else {
            return Err("Constraint settings request: expected a pending draft.".into());
        };
        if pending != action {
            return Err("Constraint settings request: expected the current draft.".into());
        }
        if options.create_dimensions_on_value_input != *original {
            return Err("Application options changed. Cancel and reopen this window.".into());
        }
        Ok(())
    }
    pub(crate) fn acknowledge(
        &mut self,
        action: &GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
        options: &mut SketchApplicationOptions,
    ) {
        let Some((pending, original, updated)) = self.pending.as_ref() else {
            return;
        };
        if pending != action {
            return;
        }
        let GuiShellAction::SetDocumentSketchSettings { document, .. } = pending else {
            return;
        };
        if active_document != Some(*document) {
            self.close();
            return;
        }
        let (original, updated) = (*original, *updated);
        self.pending = None;
        if let Err(error) = result {
            self.error = Some(error);
            return;
        }
        if options.create_dimensions_on_value_input != original {
            self.error = Some("Application options changed. Cancel and reopen this window.".into());
            return;
        }
        options.create_dimensions_on_value_input = updated;
        self.close();
    }
}
#[derive(Default)]
struct SettingsOutcome {
    accept: bool,
    close: bool,
}
fn draw_settings(
    ui: &Ui,
    settings: &mut SketchSettings,
    input_dimensions: &mut bool,
    conflict: bool,
    pending: bool,
    error: Option<&str>,
    icons: &crate::icons::GuiRibbonIcons,
) -> SettingsOutcome {
    if let Some(error) = error {
        ui.text_wrapped(error);
    }
    if conflict {
        ui.text("Sketch settings changed. Cancel and reopen this window.");
    }
    draw_settings_tabs(ui, settings, input_dimensions, icons);
    settings_buttons(ui, conflict || pending)
}
fn draw_settings_tabs(
    ui: &Ui,
    settings: &mut SketchSettings,
    input_dimensions: &mut bool,
    icons: &crate::icons::GuiRibbonIcons,
) {
    let Some(_tabs) = ui.tab_bar("##constraint-settings-tabs") else {
        return;
    };
    if let Some(_tab) = ui.tab_item("General") {
        draw_general(ui, settings, input_dimensions);
    }
    if let Some(_tab) = ui.tab_item("Inference") {
        draw_inference(ui, settings, icons);
    }
    if let Some(_tab) = ui.tab_item("Relax Mode") {
        draw_relax(ui, settings);
    }
}
fn settings_buttons(ui: &Ui, conflict: bool) -> SettingsOutcome {
    let disabled = ui.begin_disabled(conflict);
    let accept = ui.button("OK");
    disabled.end();
    ui.same_line();
    let cancel = ui.button("Cancel");
    let escape = ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
    SettingsOutcome {
        accept,
        close: cancel || escape,
    }
}
fn draw_general(ui: &Ui, settings: &mut SketchSettings, input_dimensions: &mut bool) {
    ui.checkbox(
        "Edit dimension when created",
        &mut settings.edit_dimensions_when_created,
    );
    ui.checkbox("Create dimensions from input values", input_dimensions);
    for (label, behavior) in [
        ("Warn of overconstrained condition", OVER_CONSTRAINED_PROMPT),
        ("Apply as driven dimension", OVER_CONSTRAINED_APPLY_DRIVEN),
        ("Apply as driving dimension", OVER_CONSTRAINED_APPLY_DRIVING),
    ] {
        if ui.radio_button_bool(label, settings.over_constrained_behavior == behavior) {
            settings.over_constrained_behavior = behavior;
        }
    }
}
fn set_inference(settings: &mut SketchSettings, enabled: bool) {
    settings.infer_constraints = enabled;
    if !enabled {
        settings.persist_inferred_constraints = false;
    }
}
fn draw_inference(ui: &Ui, settings: &mut SketchSettings, icons: &crate::icons::GuiRibbonIcons) {
    let mut enabled = settings.infer_constraints;
    if ui.checkbox("Infer constraints", &mut enabled) {
        set_inference(settings, enabled);
    }
    let disabled = ui.begin_disabled(!enabled);
    ui.checkbox(
        "Persist constraints",
        &mut settings.persist_inferred_constraints,
    );
    let original = if settings.custom_inference_types {
        settings.inference_types.0
    } else {
        ALL_INFERENCE
    };
    let mask = grid::inference_mask(ui, icons, original, ALL_INFERENCE, INFERENCE_TYPES);
    disabled.end();
    if mask != original {
        settings.custom_inference_types = true;
        settings.inference_types = SketchInferenceTypes(mask);
    }
    priority_choices(ui, settings);
}
fn priority_choices(ui: &Ui, settings: &mut SketchSettings) {
    for (label, priority) in [
        (
            "Horizontal and vertical priority",
            PRIORITY_HORIZONTAL_VERTICAL,
        ),
        (
            "Parallel and perpendicular priority",
            PRIORITY_PARALLEL_PERPENDICULAR,
        ),
        ("No priority", PRIORITY_NONE),
    ] {
        if ui.radio_button_bool(label, settings.constraint_priority == priority) {
            settings.constraint_priority = priority;
        }
    }
}
fn draw_relax(ui: &Ui, settings: &mut SketchSettings) {
    ui.checkbox("Enable Relax Mode", &mut settings.enable_relax_mode);
    let disabled = ui.begin_disabled(!settings.enable_relax_mode);
    ui.checkbox(
        "Preserve dimensions with equation",
        &mut settings.keep_dimensions_with_equation_in_relax_mode,
    );
    let original = if settings.custom_relax_types {
        settings.relax_types.0
    } else {
        DEFAULT_RELAX
    };
    let mask = edit_mask(
        ui,
        original,
        ALL_RELAX,
        "relax",
        RELAX_TYPES.map(|(name, bit)| (name, bit.0)),
    );
    disabled.end();
    if mask != original {
        settings.custom_relax_types = true;
        settings.relax_types = SketchRelaxTypes(mask);
    }
}
fn edit_mask(
    ui: &Ui,
    mut mask: u32,
    all: u32,
    id: &str,
    entries: impl IntoIterator<Item = (&'static str, u32)>,
) -> u32 {
    for (name, bit) in entries {
        toggle_bit(ui, name, &mut mask, bit);
    }
    if ui.button(format!("Select All##{id}")) {
        mask = all;
    }
    ui.same_line();
    if ui.button(format!("Clear All##{id}")) {
        mask = 0;
    }
    mask
}
fn toggle_bit(ui: &Ui, label: &str, mask: &mut u32, bit: u32) {
    let mut selected = *mask & bit != 0;
    if ui.checkbox(label, &mut selected) {
        if selected {
            *mask |= bit;
        } else {
            *mask &= !bit;
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn disabling_inference_clears_persistence_without_restoring_it() {
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        set_inference(&mut settings, false);
        assert!(!settings.infer_constraints && !settings.persist_inferred_constraints);
        set_inference(&mut settings, true);
        assert!(settings.infer_constraints && !settings.persist_inferred_constraints);
    }
    #[test]
    fn document_acknowledgement_controls_parent_option_staging() {
        let settings = oblikovati_model::sketch_settings::default_sketch_settings();
        let mut parent = SketchApplicationOptions::default();
        let original = parent.create_dimensions_on_value_input;
        let mut child = ConstraintSettingsWindow::default();
        child.request(7, &settings, &parent);
        child.input_draft = !original;
        let mut actions = Vec::new();
        child.submit(&mut actions);
        assert_eq!(parent.create_dimensions_on_value_input, original);
        child.acknowledge(
            &actions[0],
            Some(7),
            Err("write failed".into()),
            &mut parent,
        );
        assert_eq!(parent.create_dimensions_on_value_input, original);
        assert!(child.open.is_open());
        assert_eq!(child.error.as_deref(), Some("write failed"));
        child.submit(&mut actions);
        child.acknowledge(&actions[1], Some(7), Ok(()), &mut parent);
        assert_eq!(parent.create_dimensions_on_value_input, !original);
        assert!(!child.open.is_open());
    }
    #[test]
    fn parent_draft_change_is_rejected_before_document_write() {
        let settings = oblikovati_model::sketch_settings::default_sketch_settings();
        let mut parent = SketchApplicationOptions::default();
        let mut child = ConstraintSettingsWindow::default();
        child.request(7, &settings, &parent);
        let mut actions = Vec::new();
        child.submit(&mut actions);
        assert!(child.validate_pending(&actions[0], &parent).is_ok());
        parent.create_dimensions_on_value_input = !parent.create_dimensions_on_value_input;
        let error = child.validate_pending(&actions[0], &parent).unwrap_err();
        child.acknowledge(&actions[0], Some(7), Err(error.clone()), &mut parent);
        assert_eq!(child.error.as_deref(), Some(error.as_str()));
        assert!(child.open.is_open());
    }
    #[test]
    fn stale_acknowledgements_cannot_change_parent_drafts() {
        let settings = oblikovati_model::sketch_settings::default_sketch_settings();
        let mut parent = SketchApplicationOptions::default();
        let original = parent.create_dimensions_on_value_input;
        let mut child = ConstraintSettingsWindow::default();
        child.request(7, &settings, &parent);
        child.input_draft = !original;
        let mut actions = Vec::new();
        child.submit(&mut actions);
        child.close();
        child.request(7, &settings, &parent);
        child.input_draft = !original;
        child.submit(&mut actions);
        child.acknowledge(&actions[0], Some(7), Ok(()), &mut parent);
        assert_eq!(parent.create_dimensions_on_value_input, original);
        child.acknowledge(&actions[1], Some(8), Ok(()), &mut parent);
        assert_eq!(parent.create_dimensions_on_value_input, original);
        assert!(!child.open.is_open());
    }
    #[test]
    fn type_masks_are_distinct_and_cover_the_documented_choices() {
        assert_eq!(
            INFERENCE_TYPES.iter().fold(0, |all, (_, bit)| all | bit.0),
            ALL_INFERENCE
        );
        assert_eq!(
            RELAX_TYPES.iter().fold(0, |all, (_, bit)| all | bit.0),
            ALL_RELAX
        );
        assert_eq!(
            INFERENCE_TYPES
                .iter()
                .map(|(_, bit)| bit.0)
                .collect::<std::collections::BTreeSet<_>>()
                .len(),
            9
        );
        assert_eq!(
            RELAX_TYPES
                .iter()
                .map(|(_, bit)| bit.0)
                .collect::<std::collections::BTreeSet<_>>()
                .len(),
            12
        );
    }
}
