// SPDX-License-Identifier: GPL-2.0-only
//! Edit sketch preferences through the application preference owner.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::application_options::SketchApplicationOptions;

type ToggleField = fn(&mut SketchApplicationOptions) -> &mut bool;
const TOGGLES: [(&str, ToggleField); 12] = [
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

/// Offer the current sketch options; example: `draw(ui, state, actions)` in the preferences window.
pub(crate) fn draw(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(options) = &state.sketch_options else {
        return;
    };
    ui.separator();
    ui.text("Sketch");
    draw_grid(ui, options, actions);
    for (label, field) in TOGGLES {
        let mut edited = options.clone();
        if ui.checkbox(label, field(&mut edited)) {
            actions.push(GuiShellAction::SetSketchOptions(edited));
        }
    }
}

fn draw_grid(ui: &Ui, options: &SketchApplicationOptions, actions: &mut Vec<GuiShellAction>) {
    let mut spacing = options.grid_spacing_cm as f32;
    if ui.input_float("Grid spacing (cm)", &mut spacing).build()
        && spacing.is_finite()
        && spacing > 0.
    {
        let mut edited = options.clone();
        edited.grid_spacing_cm = f64::from(spacing);
        actions.push(GuiShellAction::SetSketchOptions(edited));
    }
    let Ok(mut interval) = i32::try_from(options.grid_major_every) else {
        ui.text("Major line interval is outside the desktop input range.");
        return;
    };
    if ui.input_int("Major line every", &mut interval).build() && interval > 0 {
        let mut edited = options.clone();
        edited.grid_major_every = i64::from(interval);
        actions.push(GuiShellAction::SetSketchOptions(edited));
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
}
