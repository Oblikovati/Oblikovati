// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Draft Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    draft::{DraftAngleMode, DraftMode, DraftPartingTool},
};

/// Which input selector in the Draft panel receives the next viewport or browser pick.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiDraftPickTarget {
    #[default]
    PullDirection,
    PartingTool,
    Faces,
}

/// Saved authored settings of an existing Draft feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDraftEditSeed {
    pub body: Vec<u8>,
    pub mode: DraftMode,
    pub pull_direction: Vec<u8>,
    pub parting_tool: Option<DraftPartingTool>,
    pub faces: Vec<Vec<u8>>,
    pub angle: String,
    pub angle_mode: DraftAngleMode,
    pub direction_flipped: bool,
    pub automatic_blending: bool,
}

/// Generation-scoped viewport face pick resolved by the desktop owner for Draft.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDraftPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
}

/// Generation-scoped Draft create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDraftRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub mode: DraftMode,
    pub pull_direction: Option<Vec<u8>>,
    pub parting_tool: Option<DraftPartingTool>,
    pub faces: Vec<Vec<u8>>,
    pub angle: String,
    pub angle_mode: DraftAngleMode,
    pub direction_flipped: bool,
    pub automatic_blending: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Draft preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDraftPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Draft preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDraftPreviewState {
    pub request: GuiDraftRequest,
    pub result: GuiDraftPreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Draft feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiDraftAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiDraftRequest>),
    Commit(Box<GuiDraftRequest>),
}

impl GuiDraftAction {
    /// Whether this Draft action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.draft_edits.contains_key(feature))
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
                    && state.draft_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiDraftPreview::Ready
                    })
            }
        }
    }
}
