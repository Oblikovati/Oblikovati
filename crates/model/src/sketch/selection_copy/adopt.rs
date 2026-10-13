// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CopyIdentities, OwnedSketchGeometry, PreparedSelectionCopy, Sketch2D, SketchEntityId,
    SketchError, copy_selection_error, entity_geometry,
};
use crate::sketch::{SketchArc2D, SketchSpline2D};
impl PreparedSelectionCopy {
    pub(super) fn append(
        &self,
        target: &mut Sketch2D,
        identities: &CopyIdentities,
    ) -> Result<(), SketchError> {
        let catalogue = entity_geometry::catalogue(&self.source);
        for source in &self.sources {
            let shape = catalogue
                .get(source)
                .ok_or_else(|| copy_selection_error(*source))?;
            self.append_shape(target, *shape, identities)?;
        }
        Ok(())
    }
    fn append_shape(
        &self,
        target: &mut Sketch2D,
        shape: OwnedSketchGeometry<'_>,
        ids: &CopyIdentities,
    ) -> Result<(), SketchError> {
        let id = ids.get(shape.id())?;
        if let OwnedSketchGeometry::Point(point) = shape {
            if self.selected.contains(&point.id) {
                return target.add_point(id, point.position);
            }
            return target.add_defining_point(id, point.position);
        }
        append_curve(target, shape, ids, id)?;
        self.append_roles(target, shape.id(), id)
    }
    fn append_roles(
        &self,
        target: &mut Sketch2D,
        source: SketchEntityId,
        copy: SketchEntityId,
    ) -> Result<(), SketchError> {
        if self.source.construction_curves.contains(&source) {
            target.set_curve_construction(&[copy], true)?;
        }
        if self.source.centerlines.contains(&source) {
            target.set_line_centerline(copy, true)?;
        }
        Ok(())
    }
}
fn append_curve(
    target: &mut Sketch2D,
    shape: OwnedSketchGeometry<'_>,
    ids: &CopyIdentities,
    id: SketchEntityId,
) -> Result<(), SketchError> {
    match shape {
        OwnedSketchGeometry::Line(line) => target.add_line(id, ids.pair(line.endpoints)?),
        OwnedSketchGeometry::Circle(circle) => {
            target.add_circle(id, ids.get(circle.center)?, circle.radius)
        }
        OwnedSketchGeometry::Arc(arc) => append_arc(target, arc, ids, id),
        OwnedSketchGeometry::Ellipse(ellipse) => {
            target.add_ellipse(ellipse.copy_identity(id, ids.get(ellipse.center())?)?)
        }
        OwnedSketchGeometry::Spline(spline) => append_spline(target, spline, ids, id),
        _ => Err(copy_selection_error(shape.id())),
    }
}
fn append_spline(
    target: &mut Sketch2D,
    spline: &SketchSpline2D,
    ids: &CopyIdentities,
    id: SketchEntityId,
) -> Result<(), SketchError> {
    let points = spline
        .points()
        .iter()
        .map(|point| ids.get(*point))
        .collect::<Result<_, _>>()?;
    target.add_spline(SketchSpline2D::new(
        id,
        points,
        spline.closed(),
        spline.kind(),
    )?)
}

fn append_arc(
    target: &mut Sketch2D,
    arc: SketchArc2D,
    ids: &CopyIdentities,
    id: SketchEntityId,
) -> Result<(), SketchError> {
    let center = ids.get(arc.center)?;
    let endpoints = ids.pair([arc.start, arc.end])?;
    target.add_arc(id, center, endpoints, arc.counter_clockwise)
}
