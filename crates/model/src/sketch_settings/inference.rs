// SPDX-License-Identifier: GPL-2.0-only
use super::SketchSettings;
use oblikovati_api::{types::*, wire::InferenceOptionsView};
use std::{error::Error, fmt};

/// A declined authoring preference with its offending wire spelling.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchInferenceError(pub String);
impl fmt::Display for SketchInferenceError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "inference priority {:?}: expected parallelPerpendicular|horizontalVertical|none or empty to retain the current priority",
            self.0
        )
    }
}
impl Error for SketchInferenceError {}
/// Project the frozen enum to the canonical inference view, including Go's unknown-enum marker.
/// Example: `let view = inference_options_view(&snapshot.effective_sketch_settings());`.
pub fn inference_options_view(settings: &SketchSettings) -> InferenceOptionsView {
    let priority = match settings.constraint_priority {
        PRIORITY_PARALLEL_PERPENDICULAR => "parallelPerpendicular",
        PRIORITY_HORIZONTAL_VERTICAL => "horizontalVertical",
        PRIORITY_NONE => "none",
        _ => "enum(?)",
    };
    InferenceOptionsView {
        infer_enabled: settings.infer_constraints,
        constrain_enabled: settings.auto_apply_constraints,
        priority: priority.into(),
    }
}
/// Replace inference fields atomically while retaining grid, creation and relax options (#3773).
/// Example: `let candidate = with_inference_options(&settings, &request)?;`.
pub fn with_inference_options(
    settings: &SketchSettings,
    request: &InferenceOptionsView,
) -> Result<SketchSettings, SketchInferenceError> {
    let priority = match request.priority.as_str() {
        "" => settings.constraint_priority,
        "parallelPerpendicular" => PRIORITY_PARALLEL_PERPENDICULAR,
        "horizontalVertical" => PRIORITY_HORIZONTAL_VERTICAL,
        "none" => PRIORITY_NONE,
        _ => return Err(SketchInferenceError(request.priority.clone())),
    };
    let mut candidate = settings.clone();
    candidate.infer_constraints = request.infer_enabled;
    candidate.auto_apply_constraints = request.constrain_enabled;
    candidate.constraint_priority = priority;
    Ok(candidate)
}
