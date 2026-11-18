// SPDX-License-Identifier: GPL-2.0-only
/// Transient profile preview request owned by the current document.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtrudePreview {
    pub generation: Option<u64>,
    pub document: u64,
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub profile: i64,
    pub distance: String,
}
/// One generation-scoped face and size for a straight-hole draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiHoleRequest {
    pub document: u64,
    pub generation: u64,
    pub face: String,
    pub diameter: String,
    pub depth: String,
}
/// One generation-scoped, exactly identified body and its ordered motion draft.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiMoveBodyRequest {
    pub document: u64,
    pub generation: u64,
    pub body_key: Vec<u8>,
    pub input: oblikovati_api::wire::featureargs::MoveBody,
}
/// One immutable edge selection and radius for a fillet draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFilletRequest {
    pub document: u64,
    pub generation: u64,
    pub edges: Vec<Vec<u8>>,
    pub radius: String,
}
/// A generation-scoped radius draft for an existing Fillet.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFilletEditRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: oblikovati_model::feature::FeatureId,
    pub radius: String,
}
/// A document-scoped extrusion distance draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiExtrudeEditRequest {
    pub generation: Option<u64>,
    pub document: u64,
    pub feature: oblikovati_model::feature::FeatureId,
    pub distance: String,
}

/// Sheet-space display geometry and any model-resolution failure.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiDrawingProjection {
    pub lines: Vec<GuiDrawingLine>,
    pub error: Option<String>,
}
/// One sampled edge in sheet millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDrawingLine {
    pub points: Vec<[f64; 2]>,
    pub hidden: bool,
    pub preview: bool,
}
