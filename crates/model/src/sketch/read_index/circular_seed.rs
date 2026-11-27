// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{SketchArc2D, SolvedCircularSeed};
impl SketchReadIndex<'_> {
    /// Read current circle/arc coordinates for a post-solve check; example: `index.circular_seed(entity)?`.
    pub fn circular_seed(&self, id: SketchEntityId) -> Result<SolvedCircularSeed, SketchError> {
        match self.entity(id)? {
            OwnedSketchGeometry::Circle(circle) => Ok(SolvedCircularSeed {
                center: self.point(circle.center)?.position,
                radius: circle.radius,
                arc: None,
            }),
            OwnedSketchGeometry::Arc(arc) => self.arc_seed(arc),
            actual => Err(SketchError::Definition(format!(
                "circular seed {id:?}, kind {:?}: expected circle or circular arc geometry",
                actual.kind()
            ))),
        }
    }
    fn arc_seed(&self, arc: SketchArc2D) -> Result<SolvedCircularSeed, SketchError> {
        let endpoints = [
            self.point(arc.start)?.position,
            self.point(arc.end)?.position,
        ];
        Ok(SolvedCircularSeed {
            center: self.point(arc.center)?.position,
            radius: self.arc_radius(arc.id)?,
            arc: Some((endpoints, arc.counter_clockwise)),
        })
    }
}
