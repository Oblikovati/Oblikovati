// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI contract types for the Part Modify Mark Properties panel (#3918).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        mark::{MarkDirection, MarkMethod, MarkType},
    },
    sketch::SketchEntityId,
};

/// Mark command eligibility reported by the desktop owner from exact solid counts and volumes.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiMarkEligibility {
    #[default]
    NoVolume,
    Eligible,
    MultipleSolidBodies,
}

/// One geometry-set row inside the Mark Properties panel.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiMarkGeometrySet {
    pub sketch: Option<DocumentSketchId>,
    pub curves: Vec<SketchEntityId>,
    pub mark_type: MarkType,
}

/// Saved authored settings of an existing Mark feature for browser Edit Feature hydration.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiMarkEditSeed {
    pub body: Vec<u8>,
    pub geometry_sets: Vec<GuiMarkGeometrySet>,
    pub method: MarkMethod,
    pub faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
    pub direction: MarkDirection,
}

/// Generation-scoped viewport face pick or chain expansion resolved by the desktop owner for Mark.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiMarkFacePick {
    pub document: u64,
    pub generation: u64,
    pub body: Vec<u8>,
    pub seed_face: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
}

/// Generation-scoped sketch curve pick resolved by the desktop owner for Mark.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiMarkSketchPick {
    pub document: u64,
    pub generation: u64,
    pub sketch: DocumentSketchId,
    pub curves: Vec<SketchEntityId>,
}

/// Generation-scoped Mark create/edit draft sent from the Properties panel to the owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiMarkRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub body: Option<Vec<u8>>,
    pub geometry_sets: Vec<GuiMarkGeometrySet>,
    pub method: MarkMethod,
    pub faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
    pub direction: MarkDirection,
    pub preview_hidden: bool,
}

/// Owner evaluation outcome for a Mark preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiMarkPreview {
    Incomplete,
    Ready,
    Failed(String),
}

/// Active Mark preview state synchronized from the desktop owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiMarkPreviewState {
    pub request: GuiMarkRequest,
    pub result: GuiMarkPreview,
}

/// Shell actions for opening, chaining, previewing, and committing a Part Modify Mark feature.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiMarkAction {
    Open,
    OpenEdit(FeatureId),
    Clear,
    ExpandFaceChain {
        document: u64,
        generation: u64,
        seed_face: Vec<u8>,
        automatic_face_chain: bool,
    },
    Preview(Box<GuiMarkRequest>),
    Commit(Box<GuiMarkRequest>),
}

impl GuiMarkAction {
    /// Whether this Mark action is enabled in the current desktop command state.
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
                        .is_some_and(|browser| browser.mark_edits.contains_key(feature))
            }
            Self::Clear => true,
            Self::ExpandFaceChain {
                document,
                generation,
                ..
            } => {
                part_open
                    && state.document == Some(*document)
                    && state.document_generation == Some(*generation)
            }
            Self::Preview(request) => {
                part_open
                    && state.document == Some(request.document)
                    && state.document_generation == Some(request.generation)
            }
            Self::Commit(request) => {
                part_open
                    && state.fillet_available
                    && state.document == Some(request.document)
                    && state.mark_preview.as_ref().is_some_and(|preview| {
                        preview.request == **request && preview.result == GuiMarkPreview::Ready
                    })
            }
        }
    }
}
