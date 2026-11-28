// SPDX-License-Identifier: GPL-2.0-only
//! Public GUI types and actions for interactive Work Plane construction (#3918).

use oblikovati_model::feature::{FeatureId, work_axis::WorkAxisReference};

/// Construction mode chosen from the 3D Model Work Features Plane menu.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiWorkPlaneMode {
    General,
    MidplaneTwoPlanes,
    ThreePoints,
    ParallelThroughPoint,
    NormalToAxisThroughPoint,
    AngleToPlaneAroundEdge,
    TwoCoplanarEdges,
    LineThroughPoint,
    MidplaneOfTorus,
    TangentThroughEdge,
    TangentThroughPoint,
    TangentParallelToPlane,
}

impl GuiWorkPlaneMode {
    /// Human-readable title for the active command window.
    pub fn title(self) -> &'static str {
        match self {
            Self::General => "Plane",
            Self::MidplaneTwoPlanes => "Midplane between Two Planes",
            Self::ThreePoints => "Three Points",
            Self::ParallelThroughPoint => "Parallel to Plane through Point",
            Self::NormalToAxisThroughPoint => "Normal to Axis through Point",
            Self::AngleToPlaneAroundEdge => "Angle to Plane around Edge",
            Self::TwoCoplanarEdges => "Two Coplanar Edges",
            Self::LineThroughPoint => "Through Axis and Point",
            Self::MidplaneOfTorus => "Midplane of Torus",
            Self::TangentThroughEdge => "Tangent to Surface through Edge",
            Self::TangentThroughPoint => "Tangent to Surface through Point",
            Self::TangentParallelToPlane => "Tangent to Surface and Parallel to Plane",
        }
    }
}

/// One typed reference picked from the Model Browser or 3D viewport.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiWorkPlanePick {
    OriginPlane(crate::GuiOriginPlaneKind),
    OriginAxis(WorkAxisReference),
    OriginPoint,
    WorkPlane(FeatureId),
    WorkAxis(FeatureId),
    WorkPoint(FeatureId),
    TopologyFace(Vec<u8>),
    TopologyEdge(Vec<u8>),
    TopologyVertex(Vec<u8>),
}

impl GuiWorkPlanePick {
    pub fn is_plane(&self) -> bool {
        matches!(
            self,
            Self::OriginPlane(_) | Self::WorkPlane(_) | Self::TopologyFace(_)
        )
    }

    pub fn is_point(&self) -> bool {
        matches!(
            self,
            Self::OriginPoint | Self::WorkPoint(_) | Self::TopologyVertex(_)
        )
    }

    pub fn is_axis(&self) -> bool {
        matches!(
            self,
            Self::OriginAxis(_) | Self::WorkAxis(_) | Self::TopologyEdge(_)
        )
    }

    pub fn is_surface(&self) -> bool {
        matches!(self, Self::TopologyFace(_))
    }
}

/// Viewport pick pushed from DesktopOwner to the active Work Plane dialog.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiWorkPlaneViewportPick {
    pub sequence: u64,
    pub document: u64,
    pub candidates: Vec<GuiWorkPlanePick>,
}

/// Authoring request for previewing or committing an extended Work Plane.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiWorkPlaneRequest {
    pub document: u64,
    pub mode: GuiWorkPlaneMode,
    pub picks: Vec<GuiWorkPlanePick>,
    pub angle: String,
}

/// Actions emitted by Work Plane ribbon controls and interactive authoring dialogs.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiWorkPlaneAction {
    Start(GuiWorkPlaneMode),
    Cancel,
    Preview(GuiWorkPlaneRequest),
    Commit(GuiWorkPlaneRequest),
    PreviewOffset {
        document: u64,
        base: crate::GuiOffsetPlaneBase,
        offset: String,
    },
    ClearPreview,
}

impl GuiWorkPlaneAction {
    pub(crate) fn enabled(&self, state: &crate::GuiCommandState) -> bool {
        match self {
            Self::Cancel | Self::ClearPreview => true,
            Self::Start(_) => {
                state.authoring == crate::GuiAuthoringContext::Part
                    && !state.transaction_open
                    && state.document.is_some()
                    && state.browser.is_some()
            }
            Self::Preview(request) | Self::Commit(request) => {
                state.authoring == crate::GuiAuthoringContext::Part
                    && !state.transaction_open
                    && state.document == Some(request.document)
            }
            Self::PreviewOffset { document, .. } => {
                state.authoring == crate::GuiAuthoringContext::Part
                    && !state.transaction_open
                    && state.document == Some(*document)
            }
        }
    }
}
