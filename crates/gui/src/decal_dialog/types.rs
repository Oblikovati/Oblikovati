// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Decal Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::{
    document_sketch::DocumentSketchId, feature::FeatureId, sketch::SketchEntityId,
};

/// Visible 2D sketch image eligible as a Decal source.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDecalImageCandidate {
    pub sketch: DocumentSketchId,
    pub sketch_name: String,
    pub image: SketchEntityId,
    pub reference: String,
}

/// Authored Decal options shown when reopening an existing Decal feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDecalEditSeed {
    pub sketch: Option<DocumentSketchId>,
    pub image: Option<SketchEntityId>,
    pub target_face: Option<Vec<u8>>,
    pub target_body: Option<String>,
    pub chain_faces: bool,
    pub wrap_to_face: bool,
    pub wrap_to_face_available: bool,
}

impl Default for GuiDecalEditSeed {
    fn default() -> Self {
        Self {
            sketch: None,
            image: None,
            target_face: None,
            target_body: None,
            chain_faces: true,
            wrap_to_face: false,
            wrap_to_face_available: false,
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDecalRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub sketch: Option<DocumentSketchId>,
    pub image: Option<SketchEntityId>,
    pub target_face: Option<Vec<u8>>,
    pub target_body: Option<String>,
    pub chain_faces: bool,
    pub wrap_to_face: bool,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for a Decal draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDecalPreview {
    Ready,
    Incomplete,
    Failed(String),
}

/// Last certified or declined Decal draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDecalPreviewState {
    pub request: GuiDecalRequest,
    pub wrap_to_face_available: bool,
    pub result: GuiDecalPreview,
}

/// Actions emitted by the Decal ribbon button and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDecalAction {
    Open,
    OpenEdit(FeatureId),
    Preview(Box<GuiDecalRequest>),
    Clear,
    Commit(Box<GuiDecalRequest>),
}

impl GuiDecalAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiDecalRequest| {
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
                        .is_some_and(|browser| browser.decal_edits.contains_key(feature))
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.decal_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiDecalPreview::Ready
                    })
            }
        }
    }
}
