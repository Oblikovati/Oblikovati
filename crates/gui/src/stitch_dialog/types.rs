// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Surface Stitch dialog (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::FeatureId;

/// Saved authored settings of an existing Stitch feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiStitchEditSeed {
    pub surfaces: Vec<Vec<u8>>,
    pub maintain_as_surface: bool,
    pub maximum_tolerance_mm: f64,
    pub show_edge_conditions: bool,
    pub show_near_tangent: bool,
}

/// One remaining free boundary edge displayed in the Stitch dialog table.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiStitchFreeEdge {
    pub index: usize,
    pub label: String,
    pub length_mm: f64,
    pub max_gap_mm: Option<f64>,
}

/// Generation-scoped surface body pick resolved by the desktop owner for Stitch.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiStitchPick {
    pub document: u64,
    pub generation: u64,
    pub sequence: u64,
    pub surface_key: Vec<u8>,
    pub surface_name: String,
}

/// Generation-scoped Stitch create/edit draft sent from the Stitch dialog to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiStitchRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub surfaces: Vec<Vec<u8>>,
    pub maintain_as_surface: bool,
    /// Parsed tolerance in millimeters when valid, else `None`.
    pub maximum_tolerance_mm: Option<f64>,
    pub show_edge_conditions: bool,
    pub show_near_tangent: bool,
}

/// Owner evaluation outcome for a Stitch preview request.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiStitchPreview {
    Incomplete,
    Ready {
        stitched_edge_count: usize,
        free_edges: Vec<GuiStitchFreeEdge>,
        near_tangent_count: usize,
    },
    Failed(String),
}

/// Active Stitch preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiStitchPreviewState {
    pub request: GuiStitchRequest,
    pub result: GuiStitchPreview,
}

/// Shell actions for opening, previewing, applying, and closing a Part Surface Stitch feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiStitchAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiStitchRequest>),
    Apply(Box<GuiStitchRequest>),
    Done,
}

impl GuiStitchAction {
    /// Whether this Stitch action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.stitch_edits.contains_key(feature))
            }
            Self::Clear | Self::Done => true,
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Apply(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
                    && request.surfaces.len() >= 2
                    && request
                        .maximum_tolerance_mm
                        .is_some_and(|t| t.is_finite() && t > 0.0)
                    && state.stitch_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request
                            && matches!(preview.result, GuiStitchPreview::Ready { .. })
                    })
            }
        }
    }
}
