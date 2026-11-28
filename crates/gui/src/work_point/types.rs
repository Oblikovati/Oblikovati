// SPDX-License-Identifier: GPL-2.0-only
//! Typed actions, modes, and frame state for Work Point commands (#3918).
use crate::{GuiAuthoringContext, GuiCommandState, GuiOriginPlaneKind};
use oblikovati_model::feature::{FeatureId, work_axis::WorkAxisReference};

/// Construction mode selected from the 3D Model > Work Features > Point menu.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiWorkPointMode {
    General,
    OnVertex,
    LoopCenter,
    Grounded,
    TwoLines,
    PlaneAndLine,
    ThreePlanes,
    TorusCenter,
    SphereCenter,
}

impl GuiWorkPointMode {
    /// Menu caption for this work-point constructor.
    pub fn label(self) -> &'static str {
        match self {
            Self::General => "Point",
            Self::OnVertex => "On Vertex, Sketch point, or Midpoint",
            Self::LoopCenter => "Center Point of Loop of Edges",
            Self::Grounded => "Grounded Point",
            Self::TwoLines => "Intersection of Two Lines",
            Self::PlaneAndLine => "Intersection of Plane/Surface and Line",
            Self::ThreePlanes => "Intersection of Three Planes",
            Self::TorusCenter => "Center Point of Torus",
            Self::SphereCenter => "Center Point of Sphere",
        }
    }

    /// Initial status prompt when starting this constructor.
    pub fn initial_prompt(self) -> &'static str {
        match self {
            Self::General => "Define work point by selecting geometry",
            Self::OnVertex => "Select a vertex, sketch point, or midpoint",
            Self::LoopCenter => "Select a closed loop of edges",
            Self::Grounded => "Select a vertex, sketch point, or work point",
            Self::TwoLines => "Select two intersecting lines or axes",
            Self::PlaneAndLine => "Select a plane/surface and a line",
            Self::ThreePlanes => "Select three planes",
            Self::TorusCenter => "Select a torus",
            Self::SphereCenter => "Select a sphere",
        }
    }

    /// Whether browser Origin plane rows are valid references for this mode.
    pub fn accepts_plane(self) -> bool {
        matches!(self, Self::General | Self::PlaneAndLine | Self::ThreePlanes)
    }

    /// Whether browser Origin axis rows are valid references for this mode.
    pub fn accepts_axis(self) -> bool {
        matches!(self, Self::General | Self::TwoLines | Self::PlaneAndLine)
    }

    /// Whether browser Origin Center Point is a valid reference for this mode.
    pub fn accepts_point(self) -> bool {
        matches!(self, Self::General | Self::OnVertex | Self::Grounded)
    }
}

/// Actions emitted by the Point menu, browser picks, and Grounded Point HUD.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiWorkPointAction {
    Start(GuiWorkPointMode),
    PickOriginPlane(GuiOriginPlaneKind),
    PickOriginAxis(WorkAxisReference),
    PickOriginCenter,
    UpdateGroundedCoords([f64; 3]),
    AcceptGrounded { coordinates: [f64; 3] },
    Cancel,
}

impl GuiWorkPointAction {
    /// True when the active document context permits this work-point action.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        if state.transaction_open {
            return false;
        }
        match self {
            Self::Start(_) => state.authoring == GuiAuthoringContext::Part,
            Self::PickOriginPlane(_)
            | Self::PickOriginAxis(_)
            | Self::PickOriginCenter
            | Self::UpdateGroundedCoords(_)
            | Self::AcceptGrounded { .. }
            | Self::Cancel => state.work_point.is_some(),
        }
    }
}

/// Active Grounded Point placement coordinates and projected 3D move triad.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiGroundedPointState {
    pub coordinates: [f64; 3],
    pub screen_origin: Option<[i32; 2]>,
    pub screen_axes: Option<[[i32; 2]; 3]>,
}

/// Snapshot of an active Work Point command in the current part document.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiWorkPointState {
    pub document: u64,
    pub mode: GuiWorkPointMode,
    pub prompt: String,
    pub picked_count: usize,
    pub grounded: Option<GuiGroundedPointState>,
}

/// Evaluated visible Work Point projected into framebuffer pixels for viewport display.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiWorkPointMarker {
    pub id: FeatureId,
    pub name: String,
    pub position: [f64; 3],
    pub pixel: Option<[i32; 2]>,
    pub selected: bool,
}
