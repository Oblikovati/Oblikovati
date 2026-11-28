// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Combine Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{FeatureId, combine::CombineOperation};

/// Active solid selector in the Combine Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiCombinePickMode {
    #[default]
    Base,
    Toolbodies,
}

/// Saved authored settings of an existing Combine feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCombineEditSeed {
    pub base_body: Vec<u8>,
    pub tool_bodies: Vec<Vec<u8>>,
    pub operation: CombineOperation,
    pub keep_toolbodies: bool,
}

/// Generation-scoped viewport solid pick resolved by the desktop owner for Combine.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCombinePick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub body: Vec<u8>,
}

/// Generation-scoped Combine create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCombineRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub base_body: Option<Vec<u8>>,
    pub tool_bodies: Vec<Vec<u8>>,
    pub operation: CombineOperation,
    pub keep_toolbodies: bool,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Combine preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiCombinePreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Combine preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCombinePreviewState {
    pub request: GuiCombineRequest,
    pub result: GuiCombinePreview,
}

/// Shell actions for opening, picking, previewing, and committing a Part Modify Combine feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiCombineAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetPickMode(GuiCombinePickMode),
    Preview(Box<GuiCombineRequest>),
    Commit(Box<GuiCombineRequest>),
}

impl GuiCombineAction {
    /// Whether this Combine action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.combine_edits.contains_key(feature))
            }
            Self::Clear | Self::SetPickMode(_) => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && state.combine_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiCombinePreview::Ready
                    })
            }
        }
    }
}
