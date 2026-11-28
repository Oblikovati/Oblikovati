// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for Part Modify Shell v2 (types, directions, overrides, approximation, #4106).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    shell::{ShellDirection, ShellOptions},
};

/// One face-specific thickness override row in the Shell Properties panel.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiShellOverrideSet {
    pub face: Option<Vec<u8>>,
    pub thickness: String,
}

/// Saved authored settings of an existing Shell feature for Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiShellV2EditSeed {
    pub generation: u64,
    pub body: Vec<u8>,
    pub openings: Vec<Vec<u8>>,
    pub thickness: String,
    pub direction: ShellDirection,
    pub overrides: Vec<GuiShellOverrideSet>,
    pub options: ShellOptions,
}

/// Active viewport face-pick target owned by the Shell Properties dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiShellPickMode {
    #[default]
    None,
    Remove,
    Override(usize),
}

/// Generation-scoped face pick resolved by the desktop owner while Shell is open.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiShellV2Pick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub mode: GuiShellPickMode,
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
}

/// Generation-scoped Shell v2 create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiShellV2Request {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub openings: Vec<Vec<u8>>,
    pub thickness: String,
    pub direction: ShellDirection,
    pub overrides: Vec<(Vec<u8>, String)>,
    pub options: ShellOptions,
}

/// Owner evaluation outcome for a Shell v2 preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiShellV2Preview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Live owner state for the active Shell v2 draft.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiShellV2PreviewState {
    pub request: GuiShellV2Request,
    pub result: GuiShellV2Preview,
}

/// Actions emitted by the Shell v2 Properties panel to the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellV2Action {
    SetPickMode(GuiShellPickMode),
    ClearPreview,
    Clear,
    Preview(Box<GuiShellV2Request>),
    Commit(Box<GuiShellV2Request>),
}

impl GuiShellV2Action {
    /// True when the action is valid for the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        if state.authoring != GuiAuthoringContext::Part {
            return false;
        }
        match self {
            Self::SetPickMode(_) | Self::ClearPreview | Self::Clear => state.document.is_some(),
            Self::Preview(req) => {
                state.document == Some(req.document)
                    && state.document_generation == Some(req.generation)
            }
            Self::Commit(req) => {
                state.document == Some(req.document)
                    && state.document_generation == Some(req.generation)
                    && !req.openings.is_empty()
                    && state.shell_v2_preview.as_ref().is_some_and(|preview| {
                        preview.request == **req && preview.result == GuiShellV2Preview::Ready
                    })
            }
        }
    }
}
