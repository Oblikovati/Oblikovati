// SPDX-License-Identifier: GPL-2.0-only
//! Transient interactive selection; model references and presentation remain caller-owned.
mod ambient;
mod filter;
mod set;
pub use ambient::{AmbientSelectionFilter, SelectionPriority};
pub use filter::SelectionFilter;
pub use set::{Selectable, SelectionSet};

/// Internal desktop categories, not the public API's wire entity kinds.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum SelectionKind {
    Face,
    Edge,
    Vertex,
    Body,
    Feature,
    Profile,
    SketchEntity,
    WorkPlane,
    Sketch,
    WorkPoint,
    WorkAxis,
    Path,
    Occurrence,
    Constraint,
    Joint,
    Representation,
    ModelState,
    DrawingView,
    PointCloud,
    PointCloudPoint,
    MeshFace,
    SketchDimension,
    SketchConstraint,
}
