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
