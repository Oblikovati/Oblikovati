// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Revolution Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState, feature_properties::FeatureOutputMode};
use oblikovati_model::{
    feature::{FeatureId, sketch_revolve::SketchRevolveDirection},
    sketch::SketchEntityId,
};

/// Extent mode selectable in the Revolution Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiRevolveExtent {
    #[default]
    Full,
    Angle,
    ToReference,
}

/// Authored Revolve options shown when reopening an existing Revolution feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRevolveEditSeed {
    pub angle: String,
    pub second_angle: String,
    pub extent: GuiRevolveExtent,
    pub direction: SketchRevolveDirection,
    pub output: FeatureOutputMode,
    pub axis: Option<SketchEntityId>,
}

impl Default for GuiRevolveEditSeed {
    fn default() -> Self {
        Self {
            angle: "360 deg".into(),
            second_angle: "45,00 deg".into(),
            extent: GuiRevolveExtent::Full,
            direction: SketchRevolveDirection::Positive,
            output: FeatureOutputMode::NewSolid,
            axis: None,
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRevolveRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub profile_cleared: bool,
    pub axis: Option<SketchEntityId>,
    pub extent: GuiRevolveExtent,
    pub direction: SketchRevolveDirection,
    pub angle: String,
    pub second_angle: String,
    pub output: FeatureOutputMode,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for a Revolve draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRevolvePreview {
    Ready,
    MissingInputs,
    MissingProfile,
    MissingAxis,
    MissingToFace,
    UnchangedCut(String),
    Failed(String),
}

/// Last certified or declined Revolve draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRevolvePreviewState {
    pub request: GuiRevolveRequest,
    pub result: GuiRevolvePreview,
    pub profile_curves: usize,
}

/// Actions emitted by the Revolve ribbon, shortcut, and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRevolveAction {
    Open,
    OpenEdit(FeatureId),
    Preview(GuiRevolveRequest),
    Clear,
    Commit(GuiRevolveRequest),
}

impl GuiRevolveAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiRevolveRequest| {
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
                        .is_some_and(|browser| browser.revolve_edits.contains_key(feature))
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.revolve_preview.as_ref().is_some_and(|preview| {
                        preview.request == *request && preview.result == GuiRevolvePreview::Ready
                    })
            }
        }
    }
}
