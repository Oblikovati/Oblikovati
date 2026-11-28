// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for Part Modify Fillet v2 (variable radius, continuity, corner setbacks, selection modes).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    fillet_options::{FilletAdvancedOptions, FilletContinuity, FilletMode},
};

/// Viewport selection mode inside the Fillet Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiFilletSelectionMode {
    #[default]
    Edge,
    Loop,
    Feature,
    Solid,
    CornerVertex,
}

/// One constant-radius edge set in the Fillet Properties panel.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletEdgeSet {
    pub edges: Vec<Vec<u8>>,
    pub radius: String,
    pub is_loop: bool,
}

/// One `(position, radius)` station row in Variable Radius Fillets.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletVariablePoint {
    pub position: String,
    pub radius: String,
}

/// Corner setback vertex state in the Fillet Properties panel.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletCornerSetback {
    pub vertex: Vec<u8>,
    pub minimal: bool,
    pub distances: Vec<String>,
}

/// Shared vertex candidate exposed by the desktop owner for Corner Setbacks.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletSharedVertex {
    pub key: Vec<u8>,
    pub point: [f64; 3],
    pub incident_edges: Vec<Vec<u8>>,
}

/// Saved authored settings of an existing Fillet feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletV2EditSeed {
    pub generation: u64,
    pub body: Vec<u8>,
    pub mode: FilletMode,
    pub continuity: FilletContinuity,
    pub smooth_radius_transition: bool,
    pub edge_sets: Vec<GuiFilletEdgeSet>,
    pub variable_points: Vec<GuiFilletVariablePoint>,
    pub corner_setback: Option<GuiFilletCornerSetback>,
    pub advanced: FilletAdvancedOptions,
}

/// Generation-scoped viewport pick resolved by the desktop owner according to active Fillet selection mode.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletV2Pick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub mode: GuiFilletSelectionMode,
    pub body: Vec<u8>,
    pub edge_groups: Vec<Vec<Vec<u8>>>,
    pub vertex: Option<GuiFilletSharedVertex>,
}

/// Generation-scoped Fillet v2 create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletV2Request {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub mode: FilletMode,
    pub continuity: FilletContinuity,
    pub smooth_radius_transition: bool,
    pub edge_sets: Vec<GuiFilletEdgeSet>,
    pub variable_points: Vec<GuiFilletVariablePoint>,
    pub corner_setback: Option<GuiFilletCornerSetback>,
    pub advanced: FilletAdvancedOptions,
    pub solid_selected: bool,
}

/// Owner evaluation outcome for a Fillet v2 preview request.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiFilletV2Preview {
    Incomplete,
    Ready,
    PartialFailure {
        failed_count: usize,
        total_count: usize,
        kept_sets: Vec<GuiFilletEdgeSet>,
    },
    Failed(String),
}

/// Active Fillet v2 preview state and available shared vertices synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFilletV2PreviewState {
    pub request: GuiFilletV2Request,
    pub result: GuiFilletV2Preview,
    pub available_vertices: Vec<GuiFilletSharedVertex>,
}

/// Shell actions for Fillet v2 selection mode, preview, and commit.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiFilletV2Action {
    SetSelectionMode(GuiFilletSelectionMode),
    Clear,
    Preview(Box<GuiFilletV2Request>),
    Commit(Box<GuiFilletV2Request>),
}

impl GuiFilletV2Action {
    /// Whether this Fillet v2 action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let base = state.fillet_available
            && state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::SetSelectionMode(_) => base,
            Self::Clear => true,
            Self::Preview(request) => {
                base && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                base && state.document == Some(request.document)
                    && state.fillet_v2_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && matches!(
                                preview.result,
                                GuiFilletV2Preview::Ready
                                    | GuiFilletV2Preview::PartialFailure { .. }
                            )
                    })
            }
        }
    }
}
