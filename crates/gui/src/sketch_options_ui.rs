// SPDX-License-Identifier: GPL-2.0-only
//! Edit sketch preferences through the application preference owner.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::application_options::SketchApplicationOptions;

type ToggleField = fn(&mut SketchApplicationOptions) -> &mut bool;
const TOGGLES: [(&str, ToggleField); 13] = [
    ("Axes", |o| &mut o.axes_visible),
    ("Show grid in sketch", |o| &mut o.grid_visible),
    ("Sketch HUD", |o| &mut o.heads_up_display),
    ("Pointer input", |o| &mut o.pointer_input),
    ("Pointer input Cartesian", |o| {
        &mut o.pointer_input_cartesian
    }),
    ("Dimension input", |o| &mut o.dimension_input),
    ("Dimension input Cartesian", |o| {
        &mut o.dimension_input_cartesian
    }),
    ("Create dimensions on value input", |o| {
        &mut o.create_dimensions_on_value_input
    }),
    ("Snap to points", |o| &mut o.snap_to_points),
    ("Snap to grid", |o| &mut o.snap_to_grid),
    ("Relax mode", |o| &mut o.relax_mode),
    ("Auto-project origin", |o| &mut o.auto_project_origin),
    ("Suppress format overrides", |o| {
        &mut o.suppress_format_overrides
    }),
];

#[derive(Default)]
pub(crate) struct SketchOptionsDraft {
    original: Option<SketchApplicationOptions>,
    edited: Option<SketchApplicationOptions>,
    hud: crate::sketch_options_hud::HudSettings,
    constraint: crate::constraint_settings_ui::ConstraintSettingsWindow,
}
impl SketchOptionsDraft {
    fn synchronize(&mut self, accepted: &SketchApplicationOptions) {
        if self.original == self.edited || self.edited.as_ref() == Some(accepted) {
            self.original = Some(accepted.clone());
            self.edited = Some(accepted.clone());
        }
    }
    pub(crate) fn clear(&mut self) {
        self.original = None;
        self.edited = None;
        self.hud.close();
        self.constraint.close();
    }
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.hud.register_dockable(registry)?;
        self.constraint.register_dockable(registry)
    }
    pub(crate) fn draw_constraint(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        if let Some(edited) = self.edited.as_mut() {
            self.constraint.panel(ui, state, edited, actions, icons);
        }
    }
    pub(crate) fn validate_constraint_settings(
        &self,
        action: &GuiShellAction,
    ) -> Result<(), String> {
        let options = self
            .edited
            .as_ref()
            .ok_or("Application options: expected an open draft")?;
        self.constraint.validate_pending(action, options)
    }
    pub(crate) fn acknowledge_constraint_settings(
        &mut self,
        action: &GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
    ) {
        if let Some(options) = self.edited.as_mut() {
            self.constraint
                .acknowledge(action, active_document, result, options);
        }
    }
    pub(crate) fn draw_hud(&mut self, ui: &Ui) {
        if let Some(edited) = self.edited.as_mut() {
            self.hud.panel(ui, edited);
        }
    }
    pub(crate) fn has_changes(&self, accepted: Option<&SketchApplicationOptions>) -> bool {
        matches!((self.edited.as_ref(), accepted), (Some(edited), Some(accepted)) if edited != accepted)
    }
    pub(crate) fn has_conflict(&self, accepted: Option<&SketchApplicationOptions>) -> bool {
        matches!((self.original.as_ref(), accepted), (Some(original), Some(accepted)) if original != accepted)
    }
    pub(crate) fn apply(
        &mut self,
        accepted: Option<&SketchApplicationOptions>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let (Some(edited), Some(accepted)) = (self.edited.as_ref(), accepted)
            && edited != accepted
            && !self.has_conflict(Some(accepted))
        {
            actions.push(GuiShellAction::SetSketchOptions(edited.clone()));
        }
    }
    pub(crate) fn draw(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(accepted) = &state.sketch_options else {
            self.clear();
            return;
        };
        self.synchronize(accepted);
        let Some(edited) = self.edited.as_mut() else {
            return;
        };
        ui.separator();
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Sketch");
        }
        if self.original.as_ref() != Some(accepted) {
            ui.text_colored(
                crate::theme::error_accent(ui),
                "Sketch settings changed. Cancel and reopen before applying.",
            );
        }
        if let (Some(document), Some(settings)) =
            (state.document, state.document_sketch_settings.as_ref())
            && crate::theme::secondary_button(ui, "Constraint Settings", [0., 0.])
        {
            self.constraint.request(document, settings, edited);
        }
        if state.document.is_some() && state.document_sketch_settings.is_some() {
            ui.same_line_with_spacing(0., 6.);
        }
        if crate::theme::secondary_button(ui, "HUD Settings", [0., 0.]) {
            self.hud.request(edited);
        }
        draw_grid(ui, edited);
        for (label, field) in TOGGLES {
            ui.checkbox(label, field(edited));
        }
    }
}

fn draw_grid(ui: &Ui, options: &mut SketchApplicationOptions) {
    let mut spacing = options.grid_spacing_cm as f32;
    crate::theme::property_label(ui, "Grid spacing (cm)");
    {
        let _numeric = crate::ui_scale::numeric_font(ui);
        if ui.input_float("##grid-spacing", &mut spacing).build()
            && spacing.is_finite()
            && spacing > 0.
        {
            options.grid_spacing_cm = f64::from(spacing);
        }
    }
    let Ok(mut interval) = i32::try_from(options.grid_major_every) else {
        ui.text("Major line interval is outside the desktop input range.");
        return;
    };
    crate::theme::property_label(ui, "Major line every");
    let _numeric = crate::ui_scale::numeric_font(ui);
    if ui.input_int("##grid-major-every", &mut interval).build() && interval > 0 {
        options.grid_major_every = i64::from(interval);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn preference_controls_have_unique_labels_and_independent_fields() {
        let original = SketchApplicationOptions::default();
        let mut labels = std::collections::BTreeSet::new();
        let mut variants = Vec::new();
        for (label, field) in TOGGLES {
            assert!(labels.insert(label));
            let mut edited = original.clone();
            *field(&mut edited) = !*field(&mut original.clone());
            assert_ne!(edited, original);
            assert!(
                !variants.contains(&edited),
                "{label}: expected a distinct preference"
            );
            variants.push(edited);
        }
    }
    #[test]
    fn draft_retains_pending_edits_and_detects_external_changes() {
        let mut draft = SketchOptionsDraft::default();
        let accepted = SketchApplicationOptions::default();
        draft.synchronize(&accepted);
        draft.edited.as_mut().unwrap().grid_visible = false;
        draft.synchronize(&accepted);
        assert!(!draft.edited.as_ref().unwrap().grid_visible);
        let mut external = accepted.clone();
        external.snap_to_grid = false;
        draft.synchronize(&external);
        assert_ne!(draft.original.as_ref(), Some(&external));
        assert!(draft.edited.as_ref().unwrap().snap_to_grid);
        draft.clear();
        draft.synchronize(&external);
        assert_eq!(draft.edited, Some(external));
    }
}
