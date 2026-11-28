// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the floating Direct Edit toolbar (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    direct_edit::{
        DirectEditCoordinateMode, DirectEditMode, DirectEditOperation, DirectEditScaleMode,
        DirectEditSelectionFilter,
    },
};

/// Saved authored settings of an existing Direct Edit feature for browser hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDirectEditEditSeed {
    pub body: Vec<u8>,
    pub operations: Vec<DirectEditOperation>,
}

/// Generation-scoped viewport face or solid pick resolved by the desktop owner for Direct Edit.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDirectEditPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub filter: DirectEditSelectionFilter,
    pub body: Vec<u8>,
    pub face: Option<Vec<u8>>,
    pub anchor_pixels: Option<[i32; 2]>,
}

/// Generation-scoped Direct Edit draft sent from the floating toolbar to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDirectEditRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub editing_op_index: Option<usize>,
    pub body: Option<Vec<u8>>,
    pub faces: Vec<Vec<u8>>,
    pub mode: DirectEditMode,
    pub filter: DirectEditSelectionFilter,
    pub scale_mode: DirectEditScaleMode,
    pub uniform_factor: String,
    pub scale_factors: [String; 3],
    pub coordinate_mode: DirectEditCoordinateMode,
    pub snap_parallel: bool,
}

/// Owner evaluation outcome for a Direct Edit preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDirectEditPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Direct Edit preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDirectEditPreviewState {
    pub request: GuiDirectEditRequest,
    pub result: GuiDirectEditPreview,
    pub anchor_pixels: Option<[i32; 2]>,
    pub last_committed_feature: Option<FeatureId>,
}

/// Shell actions for opening, picking, previewing, and applying Direct Edit operations.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiDirectEditAction {
    Open,
    OpenEdit {
        feature: FeatureId,
        operation: Option<usize>,
    },
    Clear,
    SetFilter(DirectEditSelectionFilter),
    ClearSelection,
    Preview(Box<GuiDirectEditRequest>),
    Commit(Box<GuiDirectEditRequest>),
}

impl GuiDirectEditAction {
    /// Whether this Direct Edit action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let part_open = state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Open => part_open,
            Self::OpenEdit { feature, .. } => {
                part_open
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.direct_edit_edits.contains_key(feature))
            }
            Self::Clear | Self::SetFilter(_) | Self::ClearSelection => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.direct_edit_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && preview.result == GuiDirectEditPreview::Ready
                    })
            }
        }
    }
}
