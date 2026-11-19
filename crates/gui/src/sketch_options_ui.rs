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
}
impl SketchOptionsDraft {
    fn synchronize(&mut self, accepted: &SketchApplicationOptions) {
        if self.original == self.edited || self.edited.as_ref() == Some(accepted) {
            self.original = Some(accepted.clone());
            self.edited = Some(accepted.clone());
        }
    }
    pub(crate) fn clear(&mut self) {
        *self = Self::default();
    }
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(accepted) = &state.sketch_options else {
            self.clear();
            return false;
        };
        self.synchronize(accepted);
        let Some(edited) = self.edited.as_mut() else {
            return false;
        };
        ui.separator();
        ui.text("Sketch");
        let conflict = self.original.as_ref() != Some(accepted);
        if conflict {
            ui.text("Sketch settings changed. Revert before applying.");
        }
        let close = draw_actions(ui, edited, accepted, conflict, actions);
        draw_grid(ui, edited);
        for (label, field) in TOGGLES {
            ui.checkbox(label, field(edited));
        }
        close
    }
}

fn draw_actions(
    ui: &Ui,
    edited: &SketchApplicationOptions,
    accepted: &SketchApplicationOptions,
    conflict: bool,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let changed = edited != accepted;
    let disabled = ui.begin_disabled(!changed || conflict);
    if ui.button("Apply") {
        actions.push(GuiShellAction::SetSketchOptions(edited.clone()));
    }
    disabled.end();
    ui.same_line();
    if !changed {
        return ui.button("Close");
    }
    let disabled = ui.begin_disabled(conflict);
    let ok = ui.button("OK");
    disabled.end();
    if ok {
        actions.push(GuiShellAction::SetSketchOptions(edited.clone()));
    }
    ui.same_line();
    ok || ui.button("Cancel")
}

fn draw_grid(ui: &Ui, options: &mut SketchApplicationOptions) {
    let mut spacing = options.grid_spacing_cm as f32;
    if ui.input_float("Grid spacing (cm)", &mut spacing).build()
        && spacing.is_finite()
        && spacing > 0.
    {
        options.grid_spacing_cm = f64::from(spacing);
    }
    let Ok(mut interval) = i32::try_from(options.grid_major_every) else {
        ui.text("Major line interval is outside the desktop input range.");
        return;
    };
    if ui.input_int("Major line every", &mut interval).build() && interval > 0 {
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
