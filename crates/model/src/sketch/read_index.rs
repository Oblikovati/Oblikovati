// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityId, SketchError, SketchPoint2D,
    entity_geometry::{OwnedSketchGeometry, catalogue},
};
use std::collections::{BTreeMap, BTreeSet};
/// One immutable index for a batch read; borrowing the sketch prevents stale lookups after an edit.
pub struct SketchReadIndex<'a> {
    sketch: &'a Sketch2D,
    entities: BTreeMap<SketchEntityId, OwnedSketchGeometry<'a>>,
}
impl Sketch2D {
    /// Index all geometry once for bulk conversion; example: `let index=sketch.read_index();` before drawing.
    pub fn read_index(&self) -> SketchReadIndex<'_> {
        SketchReadIndex {
            sketch: self,
            entities: catalogue(self),
        }
    }
}
impl<'a> SketchReadIndex<'a> {
    /// Read visible entity order, excluding hidden defining points; example: `index.entity_ids()` for presentation order.
    pub fn entity_ids(&self) -> &[SketchEntityId] {
        self.sketch.entity_ids()
    }
    /// Resolve one entity or hidden defining point; example: `index.entity(id)?` for typed conversion.
    pub fn entity(&self, id: SketchEntityId) -> Result<OwnedSketchGeometry<'a>, SketchError> {
        self.entities.get(&id).copied().ok_or_else(|| {
            SketchError::Definition(format!(
                "sketch entity {id:?}: expected an owned geometry identity"
            ))
        })
    }
    /// Resolve a defining point without another collection scan; example: `index.point(line.endpoints[0])?`.
    pub fn point(&self, id: SketchEntityId) -> Result<SketchPoint2D, SketchError> {
        match self.entity(id)? {
            OwnedSketchGeometry::Point(point) => Ok(point),
            actual => Err(SketchError::Definition(format!(
                "sketch point {id:?}, kind {:?}: expected point geometry",
                actual.kind()
            ))),
        }
    }
    /// Derive a circular arc radius through the common model rule; example: `index.arc_radius(id)?`.
    pub fn arc_radius(&self, id: SketchEntityId) -> Result<f64, SketchError> {
        let OwnedSketchGeometry::Arc(arc) = self.entity(id)? else {
            return Err(SketchError::Definition(format!(
                "sketch arc {id:?}: expected circular arc geometry"
            )));
        };
        super::arc::derived_radius(
            id,
            self.point(arc.center)?.position,
            self.point(arc.start)?.position,
        )
    }
    /// Read circle/arc centers that need a marker in addition to visible point entities.
    /// Example: `index.implicit_circular_centers()?` to retain Go's center targets (#2159).
    pub fn implicit_circular_centers(&self) -> Result<Vec<SketchPoint2D>, SketchError> {
        let mut marked: BTreeSet<_> = self.entity_ids().iter().copied().collect();
        self.sketch
            .circles
            .iter()
            .map(|circle| circle.center)
            .chain(self.sketch.arcs.iter().map(|arc| arc.center))
            .filter(|id| marked.insert(*id))
            .map(|id| self.point(id))
            .collect()
    }
    /// Read the geometric construction role; example: `index.is_construction(id)?` for line style.
    pub fn is_construction(&self, id: SketchEntityId) -> Result<bool, SketchError> {
        self.entity(id)?;
        Ok(self.sketch.construction_curves.contains(&id) || self.sketch.centerlines.contains(&id))
    }
    /// Read the axis role of any owned entity; non-lines return false. Example: `index.is_centerline(id)?` before choosing its pattern.
    pub fn is_centerline(&self, id: SketchEntityId) -> Result<bool, SketchError> {
        self.entity(id)?;
        Ok(self.sketch.centerlines.contains(&id))
    }
}
