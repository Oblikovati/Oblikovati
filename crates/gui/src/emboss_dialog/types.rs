// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for the dockable Emboss Properties panel.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{FeatureId, sketch_emboss::EmbossType};

/// Authored Emboss options shown when reopening an existing Emboss feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiEmbossEditSeed {
    pub depth: String,
    pub emboss_type: EmbossType,
    pub direction_reversed: bool,
    pub taper: String,
    pub wrap_to_face: bool,
    pub wrap_face: Option<Vec<u8>>,
}

impl Default for GuiEmbossEditSeed {
    fn default() -> Self {
        Self {
            depth: "1 mm".into(),
            emboss_type: EmbossType::EmbossFromFace,
            direction_reversed: false,
            taper: "0,0".into(),
            wrap_to_face: false,
            wrap_face: None,
        }
    }
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiEmbossRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub depth: String,
    pub emboss_type: EmbossType,
    pub direction_reversed: bool,
    pub taper: String,
    pub wrap_to_face: bool,
    pub wrap_face: Option<Vec<u8>>,
    pub preview_hidden: bool,
}

/// Geometry owner's certification result for an Emboss draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiEmbossPreview {
    Ready,
    OpenLoop(String),
    MissingProfile,
    MissingWrapFace,
    RangeError(String),
    Failed(String),
}

/// Last certified or declined Emboss draft state.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiEmbossPreviewState {
    pub request: GuiEmbossRequest,
    pub result: GuiEmbossPreview,
}

/// Actions emitted by the Emboss ribbon button and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiEmbossAction {
    Open,
    OpenEdit(FeatureId),
    ClearSelectedFace,
    Preview(Box<GuiEmbossRequest>),
    Clear,
    Commit(Box<GuiEmbossRequest>),
}

impl GuiEmbossAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiEmbossRequest| {
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
                        .is_some_and(|browser| browser.emboss_edits.contains_key(feature))
            }
            Self::ClearSelectedFace => state.document.is_some(),
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.emboss_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiEmbossPreview::Ready
                    })
            }
        }
    }
}
