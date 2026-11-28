// SPDX-License-Identifier: GPL-2.0-only
//! Interactive User Coordinate System (UCS) placement state, actions, and viewport triad (#3918).
mod draw;
pub(crate) use draw::draw_ucs_triads;

use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::FeatureId;

/// Active interactive placement stage for User Coordinate System authoring.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiUcsStage {
    PickOrigin,
    PickXDirection,
    PickYDirection,
}

impl GuiUcsStage {
    /// Human-readable status prompt for this UCS placement stage.
    /// Example: `GuiUcsStage::PickXDirection.prompt()`.
    pub fn prompt(self, redefining: bool) -> &'static str {
        match (self, redefining) {
            (Self::PickOrigin, false) => "Select origin vertex for UCS (Escape to cancel)",
            (Self::PickOrigin, true) => {
                "Select new origin vertex to redefine UCS (Escape to cancel)"
            }
            (Self::PickXDirection, _) => "Specify X-axis direction for UCS (Escape to cancel)",
            (Self::PickYDirection, _) => "Specify Y-axis direction for UCS (Escape to cancel)",
        }
    }
}

/// Commands dispatched from ribbon, browser context menu, or keyboard for UCS.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiUcsAction {
    Start,
    Redefine(FeatureId),
    Cancel,
    SetVisibility { feature: FeatureId, visible: bool },
}

impl GuiUcsAction {
    /// Whether this UCS action can be invoked in the current command state.
    /// Example: `GuiUcsAction::Start.enabled(&state)`.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        let in_part = state.document.is_some()
            && state.document_type == Some(oblikovati_api::types::DOCUMENT_PART)
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        match self {
            Self::Start => in_part,
            Self::Redefine(feature) | Self::SetVisibility { feature, .. } => {
                in_part
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.ucs_features.contains_key(feature))
            }
            Self::Cancel => state.ucs.active.is_some(),
        }
    }
}

/// World-space and projected screen-space geometry for a displayed 3D UCS triad.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiUcsTriad {
    pub feature: Option<FeatureId>,
    pub name: String,
    pub origin_world: [f64; 3],
    pub x_tip_world: [f64; 3],
    pub y_tip_world: [f64; 3],
    pub z_tip_world: [f64; 3],
    pub origin_pixel: Option<[i32; 2]>,
    pub x_pixel: Option<[i32; 2]>,
    pub y_pixel: Option<[i32; 2]>,
    pub z_pixel: Option<[i32; 2]>,
    pub stage: Option<GuiUcsStage>,
}

/// Active interactive UCS placement view exposed to the GUI frame.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiUcsPlacementView {
    pub document: u64,
    pub redefining: Option<FeatureId>,
    pub stage: GuiUcsStage,
    pub prompt: String,
    pub preview_triad: Option<GuiUcsTriad>,
}

/// Complete per-frame UCS presentation state on `GuiCommandState`.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiUcsState {
    pub active: Option<GuiUcsPlacementView>,
    pub triads: Vec<GuiUcsTriad>,
}

impl GuiUcsState {
    /// Current UCS placement status prompt if active; example: `state.ucs.status_prompt()`.
    pub fn status_prompt(&self) -> Option<&str> {
        self.active.as_ref().map(|view| view.prompt.as_str())
    }
}
