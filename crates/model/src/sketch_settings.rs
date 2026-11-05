// SPDX-License-Identifier: GPL-2.0-only
//! Per-document authoring defaults; the public value type belongs to the Apache API.
use crate::history::DocumentSnapshot;
pub use oblikovati_api::types::SketchSettings;
use oblikovati_api::types::{OVER_CONSTRAINED_APPLY_DRIVEN, PRIORITY_HORIZONTAL_VERTICAL};

/// Match the Go document defaults, rather than the API DTO's zero-value default (#3772).
/// Example: `let spacing_cm = default_sketch_settings().x_snap_spacing;`.
pub fn default_sketch_settings() -> SketchSettings {
    SketchSettings {
        infer_constraints: true,
        auto_apply_constraints: true,
        constraint_priority: PRIORITY_HORIZONTAL_VERTICAL,
        x_snap_spacing: 0.1,
        y_snap_spacing: 0.1,
        snaps_per_minor_grid: 1,
        minor_lines_per_major_grid_line: 10,
        persist_inferred_constraints: true,
        display_constraints_on_creation: true,
        edit_dimensions_when_created: true,
        over_constrained_behavior: OVER_CONSTRAINED_APPLY_DRIVEN,
        enable_relax_mode: false,
        keep_dimensions_with_equation_in_relax_mode: true,
    }
}
impl DocumentSnapshot {
    /// Read authoring defaults without making an absent record explicit or dirty.
    /// Example: `let snapping = snapshot.effective_sketch_settings().infer_constraints;`.
    pub fn effective_sketch_settings(&self) -> SketchSettings {
        self.sketch_settings
            .clone()
            .unwrap_or_else(default_sketch_settings)
    }
}

mod inference;
pub use inference::{SketchInferenceError, inference_options_view, with_inference_options};
