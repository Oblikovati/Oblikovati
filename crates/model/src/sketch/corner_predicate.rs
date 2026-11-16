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
