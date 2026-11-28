// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Delete Face Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::FeatureId;

/// Saved authored settings of an existing Delete Face feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeleteFaceEditSeed {
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub heal_remaining_faces: bool,
    pub select_lump_or_void: bool,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Delete Face.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeleteFacePick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
}

/// Generation-scoped Delete Face create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeleteFaceRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub faces: Vec<Vec<u8>>,
    pub heal_remaining_faces: bool,
    pub select_lump_or_void: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Delete Face preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDeleteFacePreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Delete Face preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeleteFacePreviewState {
    pub request: GuiDeleteFaceRequest,
    pub result: GuiDeleteFacePreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Delete Face feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDeleteFaceAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiDeleteFaceRequest>),
    Commit(Box<GuiDeleteFaceRequest>),
}

impl GuiDeleteFaceAction {
    /// Whether this Delete Face action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.delete_face_edits.contains_key(feature))
            }
            Self::Clear => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && state.delete_face_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && preview.result == GuiDeleteFacePreview::Ready
                    })
            }
        }
    }
}
