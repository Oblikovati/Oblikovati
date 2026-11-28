// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Surface Extend Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    extend_surface::{ExtendSurfaceExtent, ExtendSurfaceMethod, ExtendSurfaceTermination},
};

/// Active extent mode selected in the Extend Surface Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiExtendSurfaceExtentMode {
    #[default]
    Distance,
    To,
}

/// Saved authored settings of an existing Extend Surface feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtendSurfaceEditSeed {
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub edge_chain: bool,
    pub extent: ExtendSurfaceExtent,
    pub method: ExtendSurfaceMethod,
}

/// Generation-scoped viewport pick resolved by the desktop owner for Extend Surface.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiExtendSurfacePick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub target_body: Option<Vec<u8>>,
    pub edges: Vec<Vec<u8>>,
    pub termination: Option<ExtendSurfaceTermination>,
}

/// Generation-scoped Extend Surface create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtendSurfaceRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub target_body: Option<Vec<u8>>,
    pub edges: Vec<Vec<u8>>,
    pub edge_chain: bool,
    pub extent_mode: GuiExtendSurfaceExtentMode,
    /// Parsed positive distance in model units (cm) when `Distance` extent is valid.
    pub distance: Option<f64>,
    /// Selected termination target when `To` extent is active.
    pub termination: Option<ExtendSurfaceTermination>,
    pub method: ExtendSurfaceMethod,
    pub preview_hidden: bool,
}

impl GuiExtendSurfaceRequest {
    /// Build a resolved model extent when all required extent inputs are supplied.
    pub fn resolved_extent(&self) -> Option<ExtendSurfaceExtent> {
        match self.extent_mode {
            GuiExtendSurfaceExtentMode::Distance => self
                .distance
                .filter(|d| d.is_finite() && *d > 0.0)
                .map(ExtendSurfaceExtent::Distance),
            GuiExtendSurfaceExtentMode::To => self.termination.clone().map(ExtendSurfaceExtent::To),
        }
    }
}

/// Owner evaluation outcome for an Extend Surface preview request.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiExtendSurfacePreview {
    Incomplete,
    Ready { area_mm2: f64 },
    Failed(String),
}

/// Active Extend Surface preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtendSurfacePreviewState {
    pub request: GuiExtendSurfaceRequest,
    pub result: GuiExtendSurfacePreview,
}

/// Shell actions for opening, picking, previewing, and committing Part Surface Extend.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiExtendSurfaceAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    SetEdgeChain(bool),
    SetPickTermination(bool),
    Preview(Box<GuiExtendSurfaceRequest>),
    Commit(Box<GuiExtendSurfaceRequest>),
}

impl GuiExtendSurfaceAction {
    /// Whether this Extend Surface action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.extend_surface_edits.contains_key(feature))
            }
            Self::Clear | Self::SetEdgeChain(_) | Self::SetPickTermination(_) => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && !request.edges.is_empty()
                    && request.resolved_extent().is_some()
                    && state
                        .extend_surface_preview
                        .as_ref()
                        .is_some_and(|preview| {
                            preview.request == **request
                                && matches!(preview.result, GuiExtendSurfacePreview::Ready { .. })
                        })
            }
        }
    }
}
