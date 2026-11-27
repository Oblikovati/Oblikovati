// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI action, seed, source summary, and preview types for Derived Part.
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    derived_part::{DeriveMirrorPlane, DeriveStyle},
};

/// Inspected contents of a selected native source `.opd` component.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeriveSourceSummary {
    pub path: String,
    pub name: String,
    pub solid_bodies: Vec<String>,
    pub sketches: Vec<String>,
    pub work_geometry: Vec<String>,
    pub parameters: Vec<String>,
    pub model_states: Vec<String>,
    pub design_views: Vec<String>,
}

/// Authored fields for reopening an existing Derived Part feature for editing.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeriveEditSeed {
    pub source_path: String,
    pub source_name: String,
    pub style: DeriveStyle,
    pub include_bodies: bool,
    pub include_sketches: bool,
    pub include_work_geometry: bool,
    pub include_parameters: bool,
    pub scale: String,
    pub mirror: bool,
    pub mirror_plane: DeriveMirrorPlane,
    pub link_face_color: bool,
    pub link_sketch_format: bool,
    pub show_all_objects: bool,
    pub linked: bool,
    pub model_state: String,
    pub design_view: String,
}

impl Default for GuiDeriveEditSeed {
    fn default() -> Self {
        Self {
            source_path: String::new(),
            source_name: String::new(),
            style: DeriveStyle::SingleSolid,
            include_bodies: true,
            include_sketches: false,
            include_work_geometry: false,
            include_parameters: false,
            scale: "1".into(),
            mirror: false,
            mirror_plane: DeriveMirrorPlane::XY,
            link_face_color: true,
            link_sketch_format: true,
            show_all_objects: false,
            linked: true,
            model_state: "Primary".into(),
            design_view: "Primary".into(),
        }
    }
}

/// Exact Derived Part draft submitted for live preview or commit at one document generation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDeriveRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub source_path: String,
    pub source_name: String,
    pub style: DeriveStyle,
    pub include_bodies: bool,
    pub include_sketches: bool,
    pub include_work_geometry: bool,
    pub include_parameters: bool,
    pub scale: String,
    pub mirror: bool,
    pub mirror_plane: DeriveMirrorPlane,
    pub link_face_color: bool,
    pub link_sketch_format: bool,
    pub show_all_objects: bool,
    pub linked: bool,
    pub model_state: String,
    pub design_view: String,
    pub preview_hidden: bool,
}

/// Certified preview outcome reported by the desktop owner for one Derived Part request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDerivePreview {
    Ready,
    Failed(String),
}

/// Current Derived Part preview synchronized with its triggering request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDerivePreviewState {
    pub request: GuiDeriveRequest,
    pub result: GuiDerivePreview,
}

/// Shell actions dispatched by the Derived Part picker, dialog, and Model Browser rows.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDeriveAction {
    Open,
    OpenEdit(FeatureId),
    OpenSource(FeatureId),
    InspectSource { document: u64, path: String },
    Preview(Box<GuiDeriveRequest>),
    Commit(Box<GuiDeriveRequest>),
    Clear,
}

impl GuiDeriveAction {
    /// True when the active part document can open or evaluate a Derived Part operation.
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        if state.transaction_open {
            return false;
        }
        match self {
            Self::Clear => true,
            Self::Open
            | Self::OpenEdit(_)
            | Self::OpenSource(_)
            | Self::InspectSource { .. }
            | Self::Preview(_)
            | Self::Commit(_) => {
                state.document.is_some() && state.authoring == GuiAuthoringContext::Part
            }
        }
    }
}
