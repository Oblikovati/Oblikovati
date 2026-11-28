// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Slot Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState, GuiHoleSeatKind, GuiHoleTermination};
use oblikovati_model::feature::{
    FeatureId,
    hole::{HoleFastenerSpec, HoleKind, HoleTerminationSurface},
    slot::{SlotCenterMarks, SlotShape},
};

/// Saved authored settings of an existing Slot feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSlotEditSeed {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
    pub shape: SlotShape,
    pub kind: HoleKind,
    pub seat: GuiHoleSeatKind,
    pub length: String,
    pub width: String,
    pub rotation: String,
    pub depth: String,
    pub seat_width: String,
    pub seat_depth: String,
    pub seat_angle: String,
    pub termination: GuiHoleTermination,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub end_mill_pointed: bool,
    pub end_mill_angle: String,
    pub center_marks: SlotCenterMarks,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub imate: bool,
    pub fastener: Option<HoleFastenerSpec>,
}

/// Generation-scoped viewport placement pick resolved by the desktop owner for Slot.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSlotPick {
    pub document: u64,
    pub generation: u64,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
}

/// Generation-scoped Slot create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSlotRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub face: Option<Vec<u8>>,
    pub position: Option<[f64; 3]>,
    pub shape: SlotShape,
    pub kind: HoleKind,
    pub seat: GuiHoleSeatKind,
    pub length: String,
    pub width: String,
    pub rotation: String,
    pub depth: String,
    pub seat_width: String,
    pub seat_depth: String,
    pub seat_angle: String,
    pub termination: GuiHoleTermination,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub end_mill_pointed: bool,
    pub end_mill_angle: String,
    pub center_marks: SlotCenterMarks,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub imate: bool,
    pub fastener: Option<HoleFastenerSpec>,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Slot preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSlotPreview {
    Incomplete,
    Ready,
    UnchangedCut(String),
    Failed(String),
}

/// Active Slot preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSlotPreviewState {
    pub request: GuiSlotRequest,
    pub result: GuiSlotPreview,
}

/// Shell actions for opening, previewing, and committing a Part Modify Slot feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiSlotAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    Preview(Box<GuiSlotRequest>),
    Commit(Box<GuiSlotRequest>),
}

impl GuiSlotAction {
    /// Whether this Slot action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.slot_edits.contains_key(feature))
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
                    && state.slot_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiSlotPreview::Ready
                    })
            }
        }
    }
}
