// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Trim Surface dialog (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    surface_trim::{SurfaceTrimSide, SurfaceTrimTool},
};

/// Active geometry selector in the Trim Surface dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiSurfaceTrimPickMode {
    #[default]
    CuttingTool,
    Remove,
}

/// Saved authored settings of an existing Surface Trim feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSurfaceTrimEditSeed {
    pub tool: SurfaceTrimTool,
    pub target_body: Vec<u8>,
    pub remove_side: SurfaceTrimSide,
}

/// One resolved viewport entity pick for the active Trim Surface dialog.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSurfaceTrimPickKind {
    CuttingTool(SurfaceTrimTool),
    RemoveRegion {
        target_body: Vec<u8>,
        remove_side: SurfaceTrimSide,
    },
}

/// Generation-scoped viewport pick delivered from the desktop owner to Trim Surface.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSurfaceTrimPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub kind: GuiSurfaceTrimPickKind,
}

/// Generation-scoped Trim Surface create/edit draft sent from the dialog to the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSurfaceTrimRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub tool: Option<SurfaceTrimTool>,
    pub target_body: Option<Vec<u8>>,
    pub remove_side: Option<SurfaceTrimSide>,
}

/// Owner evaluation outcome for a Trim Surface preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSurfaceTrimPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Trim Surface preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSurfaceTrimPreviewState {
    pub request: GuiSurfaceTrimRequest,
    pub result: GuiSurfaceTrimPreview,
}

/// Shell actions for opening, picking, previewing, and committing a Surface Trim feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSurfaceTrimAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetPickMode(GuiSurfaceTrimPickMode),
    Preview(Box<GuiSurfaceTrimRequest>),
    Commit(Box<GuiSurfaceTrimRequest>),
}

impl GuiSurfaceTrimAction {
    /// Whether this Surface Trim action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.surface_trim_edits.contains_key(feature))
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
                    && state.surface_trim_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && preview.result == GuiSurfaceTrimPreview::Ready
                    })
            }
        }
    }
}
