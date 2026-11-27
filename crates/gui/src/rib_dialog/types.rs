// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Rib Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_rib::{RibDirection, RibThickenSide, RibThicknessPlane},
    },
    sketch::SketchEntityId,
};

/// Visible unconsumed sketch offering open line curves for a Rib profile.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRibSketchCandidate {
    pub sketch: DocumentSketchId,
    pub name: String,
    pub curves: Vec<SketchEntityId>,
}

/// Authored Rib options shown when reopening an existing Rib feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRibEditSeed {
    pub sketch: Option<DocumentSketchId>,
    pub profile_curves: Vec<SketchEntityId>,
    pub thickness: String,
    pub depth: String,
    pub to_next: bool,
    pub direction: RibDirection,
    pub flip_direction: bool,
    pub thicken_side: RibThickenSide,
    pub extend_profile: bool,
    pub draft: String,
    pub thickness_plane: RibThicknessPlane,
    pub target_body: Option<String>,
}

impl Default for GuiRibEditSeed {
    fn default() -> Self {
        Self {
            sketch: None,
            profile_curves: Vec::new(),
            thickness: "1 mm".into(),
            depth: "5 mm".into(),
            to_next: true,
            direction: RibDirection::ParallelToSketchPlane,
            flip_direction: false,
            thicken_side: RibThickenSide::Symmetric,
            extend_profile: true,
            draft: "0 deg".into(),
            thickness_plane: RibThicknessPlane::AtTop,
            target_body: None,
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRibRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub sketch: Option<DocumentSketchId>,
    pub profile_curves: Vec<SketchEntityId>,
    pub thickness: String,
    pub depth: String,
    pub to_next: bool,
    pub direction: RibDirection,
    pub flip_direction: bool,
    pub thicken_side: RibThickenSide,
    pub extend_profile: bool,
    pub draft: String,
    pub thickness_plane: RibThicknessPlane,
    pub target_body: Option<String>,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for a Rib draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRibPreview {
    Ready,
    MissingProfile,
    MissingSolid,
    RangeError(String),
    Failed(String),
}

/// Last certified or declined Rib draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRibPreviewState {
    pub request: GuiRibRequest,
    pub result: GuiRibPreview,
}

/// Actions emitted by the Rib ribbon button and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRibAction {
    Open,
    OpenEdit(FeatureId),
    Preview(Box<GuiRibRequest>),
    Clear,
    Commit(Box<GuiRibRequest>),
}

impl GuiRibAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiRibRequest| {
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
                        .is_some_and(|browser| browser.rib_edits.contains_key(feature))
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.rib_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiRibPreview::Ready
                    })
            }
        }
    }
}
