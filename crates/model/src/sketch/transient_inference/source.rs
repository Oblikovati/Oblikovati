// SPDX-License-Identifier: GPL-2.0-only
//! Preserve the authored topology that a placement may reference.
use super::{Sketch2D, SketchEntityId, SketchTransientSolveError};
use crate::sketch::{SketchArc2D, SketchLine2D, SketchSpline2D, SketchSplineHandle2D};

#[derive(Clone, Debug)]
pub(super) struct FrozenTopology {
    lines: Vec<SketchLine2D>,
    arcs: Vec<SketchArc2D>,
    circles: Vec<(SketchEntityId, SketchEntityId)>,
    ellipses: Vec<(SketchEntityId, SketchEntityId, [u64; 2])>,
    splines: Vec<SketchSpline2D>,
    offsets: Vec<(SketchEntityId, SketchEntityId)>,
    handles: Vec<SketchSplineHandle2D>,
}
impl FrozenTopology {
    pub(super) fn capture(sketch: &Sketch2D) -> Self {
        Self {
            lines: sketch.lines().to_vec(),
            arcs: sketch.arcs().to_vec(),
            circles: sketch
                .circles()
                .iter()
                .map(|circle| (circle.id, circle.center))
                .collect(),
            ellipses: sketch
                .ellipses()
                .iter()
                .map(|ellipse| {
                    (
                        ellipse.id(),
                        ellipse.center(),
                        ellipse.major_axis().map(f64::to_bits),
                    )
                })
                .collect(),
            splines: sketch.splines().to_vec(),
            offsets: sketch
                .offset_splines()
                .iter()
                .map(|offset| (offset.id(), offset.parent()))
                .collect(),
            handles: sketch.spline_handles().to_vec(),
        }
    }
    pub(super) fn validate(&self, sketch: &Sketch2D) -> Result<(), SketchTransientSolveError> {
        if !sketch.lines().starts_with(&self.lines) || !sketch.arcs().starts_with(&self.arcs) {
            return changed("source line or arc topology changed");
        }
        if !self.circles_match(sketch) || !self.ellipses_match(sketch) {
            return changed("source circle or ellipse topology changed");
        }
        if !sketch.splines().starts_with(&self.splines)
            || !self.offsets_match(sketch)
            || !sketch.spline_handles().starts_with(&self.handles)
        {
            return changed("source spline topology or handles changed");
        }
        Ok(())
    }
    fn circles_match(&self, sketch: &Sketch2D) -> bool {
        self.circles.len() <= sketch.circles().len()
            && self
                .circles
                .iter()
                .zip(sketch.circles())
                .all(|(expected, actual)| *expected == (actual.id, actual.center))
    }
    fn ellipses_match(&self, sketch: &Sketch2D) -> bool {
        self.ellipses.len() <= sketch.ellipses().len()
            && self
                .ellipses
                .iter()
                .zip(sketch.ellipses())
                .all(|(expected, actual)| {
                    *expected
                        == (
                            actual.id(),
                            actual.center(),
                            actual.major_axis().map(f64::to_bits),
                        )
                })
    }
    fn offsets_match(&self, sketch: &Sketch2D) -> bool {
        self.offsets.len() <= sketch.offset_splines().len()
            && self
                .offsets
                .iter()
                .zip(sketch.offset_splines())
                .all(|(expected, actual)| *expected == (actual.id(), actual.parent()))
    }
}
fn changed(detail: &str) -> Result<(), SketchTransientSolveError> {
    Err(SketchTransientSolveError::SourceStructureChanged(
        detail.into(),
    ))
}
