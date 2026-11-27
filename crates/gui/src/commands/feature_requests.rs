// SPDX-License-Identifier: GPL-2.0-only
/// A drawing view chosen in the desktop dialog.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBaseViewRequest {
    pub drawing: u64,
    pub source: u64,
    pub orientation: String,
    pub scale: f64,
    pub center: [f64; 2],
}
/// Transient profile preview request owned by the current document.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtrudePreview {
    pub generation: Option<u64>,
    pub document: u64,
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub profile: i64,
    pub distance: String,
    pub second_distance: Option<String>,
    pub direction: crate::GuiExtrudeDirection,
    pub reference_extent: Option<GuiExtrudeReferenceExtent>,
}
/// Bound a new extrusion using the document's selected semantic face keys.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiExtrudeReferenceExtent {
    ToFace { to: String },
    FromTo { from: String, to: String },
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
/// Selected opening faces and wall thickness for a Shell draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiShellFeatureRequest {
    pub document: u64,
    pub generation: u64,
    pub faces: Vec<String>,
    pub thickness: String,
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
    pub second_distance: Option<String>,
}

/// Sheet-space display geometry and any model-resolution failure.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiDrawingProjection {
    pub lines: Vec<GuiDrawingLine>,
    pub view_bounds_mm: std::collections::BTreeMap<String, [[f64; 2]; 2]>,
    pub parts_lists: Vec<oblikovati_model::drawing::DrawingPartsListPresentation>,
    pub error: Option<String>,
}
/// One sampled edge in sheet millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDrawingLine {
    pub points: Vec<[f64; 2]>,
    pub hidden: bool,
    pub preview: bool,
}

/// Owner validation for one extrusion expression and active document.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtrudeDistanceValidation {
    pub generation: u64,
    pub document: u64,
    pub distance: String,
    pub second_distance: Option<String>,
    pub engineering: Option<crate::GuiParameterEngineering>,
    pub error: Option<String>,
}

/// A component selection retains the open document ID until its text is changed.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiLayoutComponent {
    Open(u64),
    Named(String),
}

/// Owner actions for choosing and confirming a populated drawing layout.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiDrawingLayoutAction {
    CaptureSources,
    Confirm {
        template: Box<crate::GuiProjectTemplate>,
        component: GuiLayoutComponent,
    },
    Acknowledge {
        suppress: bool,
    },
    Cancel,
}
