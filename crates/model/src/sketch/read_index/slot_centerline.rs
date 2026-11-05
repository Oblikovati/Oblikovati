// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{SketchArc2D, SolvedCircularSeed};
impl SketchReadIndex<'_> {
    /// Read a solved slot centreline without adding a redundant curve; example: `index.arc_slot_centerline(outer,[start_cap,end_cap])?`.
    pub fn arc_slot_centerline(
        &self,
        outer: SketchEntityId,
        caps: [SketchEntityId; 2],
    ) -> Result<SolvedCircularSeed, SketchError> {
        let arc = self.slot_arc(outer)?;
        let center = self.point(arc.center)?.position;
        let endpoints = [
            self.point(self.slot_arc(caps[0])?.center)?.position,
            self.point(self.slot_arc(caps[1])?.center)?.position,
        ];
        Ok(SolvedCircularSeed {
            center,
            radius: crate::sketch::arc::derived_radius(outer, center, endpoints[0])?,
            arc: Some((endpoints, arc.counter_clockwise)),
        })
    }
    fn slot_arc(&self, id: SketchEntityId) -> Result<SketchArc2D, SketchError> {
        let OwnedSketchGeometry::Arc(arc) = self.entity(id)? else {
            return Err(SketchError::Definition(format!(
                "slot centreline member {id:?}: expected circular arc geometry"
            )));
        };
        Ok(arc)
    }
}
