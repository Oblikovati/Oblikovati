// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Split Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    split::{SplitKeepSide, SplitMode, SplitTool},
};

/// Active geometry selector in the Split Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiSplitPickMode {
    #[default]
    Tool,
    Target,
}

/// Saved authored settings of an existing Split feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSplitEditSeed {
    pub mode: SplitMode,
    pub tool: SplitTool,
    pub body: Vec<u8>,
    pub all_faces: bool,
    pub all_faces_count: usize,
    pub faces: Vec<Vec<u8>>,
    pub keep_side: SplitKeepSide,
}

/// One resolved viewport entity pick for the active Split Properties panel.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSplitPickKind {
    SketchCurve {
        tool: SplitTool,
        default_body: Option<Vec<u8>>,
        default_face_count: usize,
    },
    WorkPlane(FeatureId),
    Face {
        body: Vec<u8>,
        face: Vec<u8>,
    },
    Solid {
        body: Vec<u8>,
    },
}

/// Generation-scoped viewport pick delivered from the desktop owner to Split.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSplitPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub kind: GuiSplitPickKind,
}

/// Generation-scoped Split create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSplitRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub mode: SplitMode,
    pub tool: Option<SplitTool>,
    pub body: Option<Vec<u8>>,
    pub all_faces: bool,
    pub faces: Vec<Vec<u8>>,
    pub keep_side: SplitKeepSide,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Split preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSplitPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Split preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSplitPreviewState {
    pub request: GuiSplitRequest,
    pub result: GuiSplitPreview,
}

/// Shell actions for opening, picking, previewing, and committing a Part Modify Split feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSplitAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetPickMode(GuiSplitPickMode, SplitMode),
    Preview(Box<GuiSplitRequest>),
    Commit(Box<GuiSplitRequest>),
}

impl GuiSplitAction {
    /// Whether this Split action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.split_edits.contains_key(feature))
            }
            Self::Clear | Self::SetPickMode(..) => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && state.split_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiSplitPreview::Ready
                    })
            }
        }
    }
}
