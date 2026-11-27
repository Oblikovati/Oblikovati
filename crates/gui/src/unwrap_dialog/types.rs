// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Unwrap Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{FeatureId, unwrap::UnwrapAlignment};

/// Active edge-selection sub-tool inside the Unwrap Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiUnwrapEdgeMode {
    #[default]
    None,
    LinearResult,
    RigidResult,
    SeamEdges,
}

impl GuiUnwrapEdgeMode {
    /// Status-bar prompt displayed while this edge selector is active.
    pub const fn status_prompt(self) -> Option<&'static str> {
        match self {
            Self::None => None,
            Self::LinearResult => Some("Select a contiguous edge set to create one linear edge"),
            Self::RigidResult => Some("Select contiguous planar edges that will not deform"),
            Self::SeamEdges => Some("Select edges to act as seams"),
        }
    }
}

/// Resolved face-chain and automatic alignment vertex produced by a viewport face pick.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiUnwrapFacePick {
    pub document: u64,
    pub generation: u64,
    pub seed_face: Vec<u8>,
    pub auto_face_chain: bool,
    pub target_body: String,
    pub faces: Vec<Vec<u8>>,
    pub alignment_vertex: Option<Vec<u8>>,
}

/// Authored Unwrap options shown when reopening an existing Unwrap feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiUnwrapEditSeed {
    pub target_body: Option<String>,
    pub faces: Vec<Vec<u8>>,
    pub alignment_vertex: Option<Vec<u8>>,
    pub auto_face_chain: bool,
    pub merge_result_body: bool,
    pub alignment: UnwrapAlignment,
    pub linear_result_edges: Vec<Vec<u8>>,
    pub rigid_result_edges: Vec<Vec<u8>>,
    pub seam_edges: Vec<Vec<u8>>,
}

impl Default for GuiUnwrapEditSeed {
    fn default() -> Self {
        Self {
            target_body: None,
            faces: Vec::new(),
            alignment_vertex: None,
            auto_face_chain: true,
            merge_result_body: true,
            alignment: UnwrapAlignment::OriginXY,
            linear_result_edges: Vec::new(),
            rigid_result_edges: Vec::new(),
            seam_edges: Vec::new(),
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiUnwrapRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub target_body: Option<String>,
    pub faces: Vec<Vec<u8>>,
    pub alignment_vertex: Option<Vec<u8>>,
    pub auto_face_chain: bool,
    pub merge_result_body: bool,
    pub alignment: UnwrapAlignment,
    pub linear_result_edges: Vec<Vec<u8>>,
    pub rigid_result_edges: Vec<Vec<u8>>,
    pub seam_edges: Vec<Vec<u8>>,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for an Unwrap draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiUnwrapPreview {
    Ready,
    Incomplete,
    Failed(String),
}

/// Last certified or declined Unwrap draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiUnwrapPreviewState {
    pub request: GuiUnwrapRequest,
    pub result: GuiUnwrapPreview,
}

/// Actions emitted by the Unwrap ribbon button and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiUnwrapAction {
    Open,
    OpenEdit(FeatureId),
    PickFace {
        document: u64,
        generation: u64,
        face: Vec<u8>,
        auto_face_chain: bool,
    },
    Preview(Box<GuiUnwrapRequest>),
    Clear,
    Commit(Box<GuiUnwrapRequest>),
}

impl GuiUnwrapAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiUnwrapRequest| {
            can_start
                && state.document == Some(request.document)
                && state.document_generation == Some(request.generation)
        };
        match self {
            Self::Open => can_start,
            Self::OpenEdit(feature) => {
                can_start
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.unwrap_edits.contains_key(feature))
            }
            Self::PickFace {
                document,
                generation,
                ..
            } => {
                can_start
                    && state.document == Some(*document)
                    && state.document_generation == Some(*generation)
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.unwrap_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiUnwrapPreview::Ready
                    })
            }
        }
    }
}
