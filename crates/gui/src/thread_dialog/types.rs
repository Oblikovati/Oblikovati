// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Thread Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{FeatureId, thread::ThreadSpecification};

/// Saved authored settings of an existing Thread feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThreadEditSeed {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub full_depth: bool,
    pub measured_depth_mm: f64,
    pub depth: String,
    pub offset: String,
    pub display_in_model: bool,
    pub spec: ThreadSpecification,
    pub face_conical: bool,
    pub face_internal: bool,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Thread.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThreadPick {
    pub document: u64,
    pub generation: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub diameter_mm: f64,
    pub full_depth_mm: f64,
    pub conical: bool,
    pub internal: bool,
}

/// Generation-scoped Thread create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThreadRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub face: Option<Vec<u8>>,
    pub full_depth: bool,
    pub depth: String,
    pub offset: String,
    pub display_in_model: bool,
    pub spec: ThreadSpecification,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Thread preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiThreadPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Thread preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThreadPreviewState {
    pub request: GuiThreadRequest,
    pub result: GuiThreadPreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Thread feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiThreadAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiThreadRequest>),
    Commit(Box<GuiThreadRequest>),
}

impl GuiThreadAction {
    /// Whether this Thread action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.thread_edits.contains_key(feature))
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
                    && state.thread_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiThreadPreview::Ready
                    })
            }
        }
    }
}
