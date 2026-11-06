// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::GuiDimensionLabelRole;
use oblikovati_model::sketch::SketchPlane;

/// A model-plane annotation; example: project `annotation.position` through its current view.
#[derive(Clone, Debug, PartialEq)]
pub struct HeadDimensionAnnotation {
    pub plane: SketchPlane,
    pub position: [f64; 2],
    pub text: String,
    pub role: GuiDimensionLabelRole,
}
/// Local annotation failures retain other labels; example: show `annotations.declines` in command feedback.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadDimensionAnnotations {
    pub labels: Vec<HeadDimensionAnnotation>,
    pub declines: Vec<String>,
}
