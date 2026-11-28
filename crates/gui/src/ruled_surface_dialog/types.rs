// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Surface Ruled Surface task dialog (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    ruled_surface::{RuledSurfaceType, RuledSurfaceVector},
};

/// Active viewport selection slot inside the Ruled Surface dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiRuledSurfacePickTarget {
    #[default]
    Edges,
    Vector,
}

/// Saved authored settings of an existing Ruled Surface feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiRuledSurfaceEditSeed {
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub ruled_type: RuledSurfaceType,
    /// Saved distance in model units (cm).
    pub distance: f64,
    /// Saved angle in degrees.
    pub angle_deg: f64,
    pub flip: bool,
    pub automatic_edge_chain: bool,
    pub vector: Option<RuledSurfaceVector>,
}

/// Generation-scoped viewport pick resolved by the desktop owner for Ruled Surface.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRuledSurfacePick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub target: GuiRuledSurfacePickTarget,
    pub target_body: Option<Vec<u8>>,
    pub edges: Vec<Vec<u8>>,
    pub vector: Option<RuledSurfaceVector>,
}

/// 3D world-space and projected 2D screen-space preview direction arrow.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GuiRuledSurfaceArrow {
    pub origin_world: [f64; 3],
    pub tip_world: [f64; 3],
    pub direction: [f64; 3],
    pub origin_pixel: Option<[i32; 2]>,
    pub tip_pixel: Option<[i32; 2]>,
    pub flipped: bool,
}

/// Generation-scoped Ruled Surface create/edit draft sent from the dialog to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiRuledSurfaceRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub target_body: Option<Vec<u8>>,
    pub edges: Vec<Vec<u8>>,
    pub ruled_type: RuledSurfaceType,
    /// Parsed distance in model units (cm) when valid (> 0), else None.
    pub distance: Option<f64>,
    /// Parsed angle in degrees when valid (in [-180, 180]), else None.
    pub angle_deg: Option<f64>,
    pub flip: bool,
    pub automatic_edge_chain: bool,
    pub vector: Option<RuledSurfaceVector>,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Ruled Surface preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRuledSurfacePreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Ruled Surface preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiRuledSurfacePreviewState {
    pub request: GuiRuledSurfaceRequest,
    pub result: GuiRuledSurfacePreview,
    pub arrow: Option<GuiRuledSurfaceArrow>,
}

/// Shell actions for opening, picking, previewing, and committing a Part Surface Ruled Surface feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiRuledSurfaceAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetAutomaticEdgeChain(bool),
    SetPickTarget(GuiRuledSurfacePickTarget),
    Preview(Box<GuiRuledSurfaceRequest>),
    Commit(Box<GuiRuledSurfaceRequest>),
}

impl GuiRuledSurfaceAction {
    /// Whether this Ruled Surface action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let part_open = state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Open => part_open,
            Self::OpenEdit(feature) => {
                part_open
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.ruled_surface_edits.contains_key(feature))
            }
            Self::Clear | Self::SetAutomaticEdgeChain(_) | Self::SetPickTarget(_) => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && request.distance.is_some()
                    && request.angle_deg.is_some()
                    && !request.edges.is_empty()
                    && (request.ruled_type != RuledSurfaceType::Sweep || request.vector.is_some())
                    && state.ruled_surface_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && preview.result == GuiRuledSurfacePreview::Ready
                    })
            }
        }
    }
}
