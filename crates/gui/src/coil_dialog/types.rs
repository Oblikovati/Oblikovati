// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Coil Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState, feature_properties::FeatureOutputMode};
use oblikovati_model::feature::{
    FeatureId,
    sketch_coil::{CoilAxisSource, CoilHandedness, CoilMethod},
};

/// Authored Coil options shown when reopening an existing Coil feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCoilEditSeed {
    pub axis: Option<CoilAxisSource>,
    pub axis_reversed: bool,
    pub method: CoilMethod,
    pub pitch: String,
    pub revolutions: String,
    pub height: String,
    pub taper: String,
    pub handedness: CoilHandedness,
    pub close_start: bool,
    pub start_flat: String,
    pub start_transition: String,
    pub close_end: bool,
    pub end_flat: String,
    pub end_transition: String,
    pub output: FeatureOutputMode,
}

impl Default for GuiCoilEditSeed {
    fn default() -> Self {
        Self {
            axis: None,
            axis_reversed: false,
            method: CoilMethod::PitchAndRevolution,
            pitch: "15 mm".into(),
            revolutions: "2.5".into(),
            height: "37.5 mm".into(),
            taper: "0 deg".into(),
            handedness: CoilHandedness::Left,
            close_start: false,
            start_flat: "90 deg".into(),
            start_transition: "90 deg".into(),
            close_end: false,
            end_flat: "90 deg".into(),
            end_transition: "90 deg".into(),
            output: FeatureOutputMode::NewSolid,
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCoilRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub profile_cleared: bool,
    pub axis: Option<CoilAxisSource>,
    pub axis_reversed: bool,
    pub method: CoilMethod,
    pub pitch: String,
    pub revolutions: String,
    pub height: String,
    pub taper: String,
    pub handedness: CoilHandedness,
    pub close_start: bool,
    pub start_flat: String,
    pub start_transition: String,
    pub close_end: bool,
    pub end_flat: String,
    pub end_transition: String,
    pub output: FeatureOutputMode,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for a Coil draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiCoilPreview {
    Ready,
    MissingProfile,
    MissingAxis,
    UnchangedCut(String),
    Failed(String),
}

/// Last certified or declined Coil draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiCoilPreviewState {
    pub request: GuiCoilRequest,
    pub result: GuiCoilPreview,
}

/// Actions emitted by the Coil ribbon button and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiCoilAction {
    Open,
    OpenEdit(FeatureId),
    Preview(Box<GuiCoilRequest>),
    Clear,
    Commit(Box<GuiCoilRequest>),
}

impl GuiCoilAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiCoilRequest| {
            can_start
                && state.document == Some(request.document)
                && state.document_generation == Some(request.generation)
        };
        match self {
            Self::Open => can_start,
            Self::OpenEdit(feature) => {
                can_start
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.coil_edits.contains_key(feature))
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.coil_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiCoilPreview::Ready
                    })
            }
        }
    }
}
