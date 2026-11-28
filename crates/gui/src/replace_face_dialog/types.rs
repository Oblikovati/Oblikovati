// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Surface Replace Face Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{FeatureId, replace_face::ReplaceFaceTarget};

/// Active face-selection target in the Replace Face Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiReplaceFacePickMode {
    #[default]
    ExistingFaces,
    NewFaces,
}

/// Saved authored settings of an existing Replace Face feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiReplaceFaceEditSeed {
    pub body: Vec<u8>,
    pub existing_faces: Vec<Vec<u8>>,
    pub new_faces: Vec<ReplaceFaceTarget>,
    pub automatic_face_chain: bool,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Replace Face.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiReplaceFacePick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub is_surface: bool,
}

/// Generation-scoped Replace Face create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiReplaceFaceRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub existing_faces: Vec<Vec<u8>>,
    pub new_faces: Vec<ReplaceFaceTarget>,
    pub automatic_face_chain: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Replace Face preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiReplaceFacePreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Replace Face preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiReplaceFacePreviewState {
    pub request: GuiReplaceFaceRequest,
    pub result: GuiReplaceFacePreview,
}

/// Shell actions for opening, picking, previewing, and committing a Part Surface Replace Face feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiReplaceFaceAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetPickMode(GuiReplaceFacePickMode),
    SetAutomaticFaceChain(bool),
    Preview(Box<GuiReplaceFaceRequest>),
    Commit(Box<GuiReplaceFaceRequest>),
}

impl GuiReplaceFaceAction {
    /// Whether this Replace Face action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let part_open = state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Open => part_open && state.fillet_available,
            Self::OpenEdit(feature) => {
                part_open
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.replace_face_edits.contains_key(feature))
            }
            Self::Clear | Self::SetPickMode(_) | Self::SetAutomaticFaceChain(_) => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && state.replace_face_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && preview.result == GuiReplaceFacePreview::Ready
                    })
            }
        }
    }
}
