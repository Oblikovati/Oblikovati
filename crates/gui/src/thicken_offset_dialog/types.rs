// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Thicken/Offset Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOutputMode,
    },
};

/// Saved authored settings of an existing Thicken/Offset feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThickenOffsetEditSeed {
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub output: ThickenOutputMode,
    /// Saved distance in model units (cm).
    pub distance: f64,
    pub direction: ThickenDirection,
    pub boolean: ThickenBoolean,
    pub automatic_face_chain: bool,
    pub automatic_blending: bool,
    pub allow_approximation: bool,
    pub approximation_type: ThickenApproximationType,
    pub approximation_tolerance: ThickenApproximationTolerance,
    pub approximation_percent: Option<f64>,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Thicken/Offset.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiThickenOffsetPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
}

/// Generation-scoped Thicken/Offset create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThickenOffsetRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub target_body: Option<Vec<u8>>,
    pub faces: Vec<Vec<u8>>,
    pub output: ThickenOutputMode,
    /// Parsed distance in model units (cm) when valid, else None.
    pub distance: Option<f64>,
    pub direction: ThickenDirection,
    pub boolean: ThickenBoolean,
    pub automatic_face_chain: bool,
    pub automatic_blending: bool,
    pub allow_approximation: bool,
    pub approximation_type: ThickenApproximationType,
    pub approximation_tolerance: ThickenApproximationTolerance,
    pub approximation_percent: Option<f64>,
    pub approximation_valid: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Thicken/Offset preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiThickenOffsetPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Thicken/Offset preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThickenOffsetPreviewState {
    pub request: GuiThickenOffsetRequest,
    pub result: GuiThickenOffsetPreview,
}

/// Shell actions for opening, picking, previewing, and committing a Part Modify Thicken/Offset feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiThickenOffsetAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetAutomaticFaceChain(bool),
    Preview(Box<GuiThickenOffsetRequest>),
    Commit(Box<GuiThickenOffsetRequest>),
}

impl GuiThickenOffsetAction {
    /// Whether this Thicken/Offset action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.thicken_offset_edits.contains_key(feature))
            }
            Self::Clear | Self::SetAutomaticFaceChain(_) => true,
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
                    && request.approximation_valid
                    && state
                        .thicken_offset_preview
                        .as_ref()
                        .is_some_and(|preview| {
                            preview.request == **request
                                && preview.result == GuiThickenOffsetPreview::Ready
                        })
            }
        }
    }
}
