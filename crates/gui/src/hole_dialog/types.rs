// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Hole Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    hole::{HoleFastenerSpec, HoleKind, HoleTerminationSurface, HoleThreadSpec},
};

/// Termination mode selected in the Hole Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiHoleTermination {
    Distance,
    #[default]
    ThroughAll,
    ToSurface,
}

/// Seat geometry selected in the Hole Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiHoleSeatKind {
    #[default]
    None,
    Counterbore,
    Spotface,
    Countersink,
}

/// Saved authored settings of an existing Hole feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiHoleEditSeed {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
    pub kind: HoleKind,
    pub seat: GuiHoleSeatKind,
    pub diameter: String,
    pub depth: String,
    pub seat_diameter: String,
    pub seat_depth: String,
    pub seat_angle: String,
    pub termination: GuiHoleTermination,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub drill_point_angled: bool,
    pub drill_point_angle: String,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub fastener: Option<HoleFastenerSpec>,
    pub thread: Option<HoleThreadSpec>,
}

/// A generation-scoped viewport placement pick resolved by the desktop owner for Hole.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiHolePick {
    pub document: u64,
    pub generation: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
}

/// Generation-scoped Hole create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiHoleRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub face: Option<Vec<u8>>,
    pub position: Option<[f64; 3]>,
    pub kind: HoleKind,
    pub seat: GuiHoleSeatKind,
    pub diameter: String,
    pub depth: String,
    pub seat_diameter: String,
    pub seat_depth: String,
    pub seat_angle: String,
    pub termination: GuiHoleTermination,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub drill_point_angled: bool,
    pub drill_point_angle: String,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub fastener: Option<HoleFastenerSpec>,
    pub thread: Option<HoleThreadSpec>,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Hole preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiHolePreview {
    Incomplete,
    Ready,
    UnchangedCut(String),
    Failed(String),
}

/// Active Hole preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiHolePreviewState {
    pub request: GuiHoleRequest,
    pub result: GuiHolePreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Hole feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiHoleAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiHoleRequest>),
    Commit(Box<GuiHoleRequest>),
}

impl GuiHoleAction {
    /// Whether this Hole action is enabled in the current desktop command state.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let base = state.fillet_available
            && state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Open => base,
            Self::OpenEdit(feature) => {
                base && state
                    .browser
                    .as_ref()
                    .is_some_and(|browser| browser.hole_edits.contains_key(feature))
            }
            Self::Clear => true,
            Self::Preview(request) => {
                base && state.document == Some(request.document)
                    && state
                        .fillet_selection
                        .as_ref()
                        .is_some_and(|(generation, _)| *generation == request.generation)
            }
            Self::Commit(request) => {
                base && state.document == Some(request.document)
                    && state.hole_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiHolePreview::Ready
                    })
            }
        }
    }
}
