// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Finish Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::FeatureId;
pub use oblikovati_model::feature::finish::FinishType as GuiFinishType;

/// Active face-picking target inside the Finish Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiFinishSelectionMode {
    #[default]
    Include,
    Exclude,
}

/// Saved authored settings of an existing Finish feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFinishEditSeed {
    pub body: Option<Vec<u8>>,
    pub included_faces: Vec<Vec<u8>>,
    pub excluded_faces: Vec<Vec<u8>>,
    pub finish_type: GuiFinishType,
    pub process: String,
    pub appearance: String,
    pub thickness: String,
    pub hardness: String,
    pub depth: String,
    pub description: String,
    pub short_description: String,
    pub comment: String,
    pub export_all_parameters: bool,
    pub has_warning: bool,
    pub area_mm2: f64,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Finish.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFinishPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
}

/// Generation-scoped Finish create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFinishRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub included_faces: Vec<Vec<u8>>,
    pub excluded_faces: Vec<Vec<u8>>,
    pub finish_type: GuiFinishType,
    pub process: String,
    pub appearance: String,
    pub thickness: String,
    pub hardness: String,
    pub depth: String,
    pub description: String,
    pub short_description: String,
    pub comment: String,
    pub export_all_parameters: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Finish preview request.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiFinishPreview {
    Incomplete,
    Ready { area_mm2: f64 },
    Failed(String),
}

/// Active Finish preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFinishPreviewState {
    pub request: GuiFinishRequest,
    pub result: GuiFinishPreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Finish feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiFinishAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiFinishRequest>),
    Commit(Box<GuiFinishRequest>),
}

impl GuiFinishAction {
    /// Whether this Finish action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.finish_edits.contains_key(feature))
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
                    && state.finish_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && matches!(preview.result, GuiFinishPreview::Ready { .. })
                    })
            }
        }
    }
}
