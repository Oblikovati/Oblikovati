// SPDX-License-Identifier: GPL-2.0-only
use std::error::Error;

/// Exact orientation of three finite planar points.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchCornerTurn {
    Clockwise,
    Collinear,
    CounterClockwise,
}
/// Injected kernel predicate for a sketch corner's topological branch.
pub trait SketchCornerPredicate {
    /// Classify a corner without a geometric tolerance.
    /// Example: `predicate.turn([corner, first, second])?`.
    fn turn(&self, points: [[f64; 2]; 3]) -> Result<SketchCornerTurn, Box<dyn Error>>;
}

/// Two setback lengths or a tangent arc radius in working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchCornerSize {
    Chamfer([f64; 2]),
    Fillet(f64),
}
/// Constructed trim points and an optional arc center and direction.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchCornerSeed {
    pub endpoints: [[f64; 2]; 2],
    pub arc: Option<([f64; 2], bool)>,
}
/// Injected geometry constructor for the selected corner branch.
pub trait SketchCornerConstruction: SketchCornerPredicate {
    /// Construct within both finite source segments and certify the metric residuals.
    /// Example: `backend.construct_corner(points, size, bound)?`.
    fn construct_corner(
        &self,
        points: [[f64; 2]; 3],
        size: SketchCornerSize,
        maximum_residual: f64,
    ) -> Result<SketchCornerSeed, Box<dyn Error>>;
}
