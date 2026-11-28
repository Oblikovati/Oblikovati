// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for Part Modify Chamfer v2 (modes, partial edges, Apply, error recovery, #4046).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    chamfer_options::{ChamferDrivenDimension, ChamferMode},
};

/// Active tab inside the Chamfer Properties dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiChamferTab {
    #[default]
    Chamfer,
    Partial,
}

/// Partial-edge chamfer state inside the Partial tab table.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiChamferPartialState {
    pub edge_length_cm: f64,
    pub to_start: String,
    pub chamfer_length: String,
    pub to_end: String,
    pub driven: ChamferDrivenDimension,
}

/// Saved authored settings of an existing Chamfer feature for Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiChamferV2EditSeed {
    pub generation: u64,
    pub body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub mode: ChamferMode,
    pub distance: String,
    pub distance2: String,
    pub angle: String,
    pub reference_face: Option<Vec<u8>>,
    pub reversed: bool,
    pub edge_chain_second: bool,
    pub setback_second: bool,
    pub partial: Option<GuiChamferPartialState>,
}

/// Viewport pick mode owned by the Chamfer Properties dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiChamferPickMode {
    #[default]
    None,
    Face,
    PartialEdge,
}

/// Generation-scoped pick resolved by the desktop owner when Chamfer face or partial-edge pick is active.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiChamferV2Pick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub mode: GuiChamferPickMode,
    pub body: Vec<u8>,
    pub face: Option<Vec<u8>>,
}

/// Generation-scoped Chamfer v2 create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiChamferV2Request {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub edges: Vec<Vec<u8>>,
    pub mode: ChamferMode,
    pub distance: String,
    pub distance2: String,
    pub angle: String,
    pub reference_face: Option<Vec<u8>>,
    pub reversed: bool,
    pub edge_chain_second: bool,
    pub setback_second: bool,
    pub partial: Option<GuiChamferPartialState>,
}

/// Owner evaluation outcome for a Chamfer v2 preview request.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiChamferV2Preview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Live owner state for the active Chamfer v2 draft, including measured edge span for Partial mode.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiChamferV2PreviewState {
    pub request: GuiChamferV2Request,
    pub result: GuiChamferV2Preview,
    pub edge_length_cm: Option<f64>,
    pub edge_endpoints: Option<[[f64; 3]; 2]>,
}

/// Actions emitted by the Chamfer v2 Properties panel to the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiChamferV2Action {
    SetPickMode(GuiChamferPickMode),
    Clear,
    Preview(Box<GuiChamferV2Request>),
    Commit(Box<GuiChamferV2Request>),
    Apply(Box<GuiChamferV2Request>),
}

impl GuiChamferV2Action {
    /// True when the action is valid for the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        if state.authoring != GuiAuthoringContext::Part {
            return false;
        }
        match self {
            Self::SetPickMode(_) | Self::Clear => state.document.is_some(),
            Self::Preview(req) => {
                state.document == Some(req.document)
                    && state.document_generation == Some(req.generation)
            }
            Self::Commit(req) | Self::Apply(req) => {
                state.document == Some(req.document)
                    && state.document_generation == Some(req.generation)
                    && !req.edges.is_empty()
                    && state.chamfer_v2_preview.as_ref().is_some_and(|preview| {
                        preview.request == **req && preview.result == GuiChamferV2Preview::Ready
                    })
            }
        }
    }
}
