// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Surface Boundary Patch Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::FeatureId;
pub use oblikovati_model::feature::boundary_patch::{
    BoundaryCondition as GuiBoundaryCondition, BoundaryCurveRef as GuiBoundaryCurveRef,
    BoundaryEdgeSet as GuiBoundaryEdgeSet,
};

/// Active viewport curve-picking target inside the Boundary Patch Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiBoundaryPatchSelectionMode {
    #[default]
    Boundary,
    GuideRails,
}

/// Saved authored settings of an existing Boundary Patch feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBoundaryPatchEditSeed {
    pub automatic_edge_chain: bool,
    pub edge_sets: Vec<GuiBoundaryEdgeSet>,
    pub guide_rails: Vec<GuiBoundaryCurveRef>,
}

/// Generation-scoped viewport curve pick resolved by the desktop owner for Boundary Patch.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiBoundaryPatchPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub mode: GuiBoundaryPatchSelectionMode,
    pub edges: Vec<GuiBoundaryCurveRef>,
}

/// Generation-scoped Boundary Patch create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBoundaryPatchRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub automatic_edge_chain: bool,
    pub edge_sets: Vec<GuiBoundaryEdgeSet>,
    pub guide_rails: Vec<GuiBoundaryCurveRef>,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Boundary Patch preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiBoundaryPatchPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Boundary Patch preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBoundaryPatchPreviewState {
    pub request: GuiBoundaryPatchRequest,
    pub result: GuiBoundaryPatchPreview,
}

/// Shell actions for opening, picking, previewing, and committing a Surface Boundary Patch feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiBoundaryPatchAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetAutomaticEdgeChain(bool),
    SetSelectionMode(GuiBoundaryPatchSelectionMode),
    Preview(Box<GuiBoundaryPatchRequest>),
    Commit(Box<GuiBoundaryPatchRequest>),
}

impl GuiBoundaryPatchAction {
    /// Whether this Boundary Patch action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let part_open = state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Open => part_open && state.certified_part_feature_available,
            Self::OpenEdit(feature) => {
                part_open
                    && state.certified_part_feature_available
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.boundary_patch_edits.contains_key(feature))
            }
            Self::Clear | Self::SetAutomaticEdgeChain(_) | Self::SetSelectionMode(_) => true,
            Self::Preview(request) => {
                part_open
                    && state.certified_part_feature_available
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.certified_part_feature_available
                    && state.document == Some(request.document)
                    && state
                        .boundary_patch_preview
                        .as_ref()
                        .is_some_and(|preview| {
                            preview.request == **request
                                && preview.result == GuiBoundaryPatchPreview::Ready
                        })
            }
        }
    }
}
