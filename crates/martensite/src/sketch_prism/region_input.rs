// SPDX-License-Identifier: GPL-2.0-only
//! Analytic source conversion in authored order; unsupported curves decline before construction.
use super::super::*;
use martensite_tkbo::profile_extrusion::{
    ArcSense, ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId,
    SketchProfile,
};
use oblikovati_model::sketch::{
    OwnedSketchGeometry, Sketch2D, SketchArc2D, SketchEntityId, SketchPlane,
};
use std::collections::BTreeMap;

pub(in crate::sketch_prism) fn profile(
    sketch: &Sketch2D,
    plane: SketchPlane,
) -> Result<(SketchProfile, BTreeMap<u64, Vec<u64>>), EvaluationFailure> {
    let coincidence = incidence::CoincidentPoints::new(sketch);
    let index = sketch.read_index();
    let mut vertices = BTreeMap::new();
    let mut curves = Vec::new();
    for &id in index.entity_ids() {
        let geometry = index
            .entity(id)
            .map_err(|cause| Phase::Input.sketch(Entity::SketchEntity(id.0), cause))?;
        if matches!(geometry, OwnedSketchGeometry::Point(_))
            || index
                .is_construction(id)
                .map_err(|cause| Phase::Input.sketch(Entity::SketchEntity(id.0), cause))?
        {
            continue;
        }
        curves.push(curve(sketch, geometry, &coincidence, &mut vertices)?);
    }
    let frame = plane.recipe();
    let sources = vertices
        .keys()
        .map(|id| (id.0, coincidence.sources(SketchEntityId(id.0))))
        .collect();
    Ok((
        SketchProfile {
            plane: ProfilePlane {
                origin: frame.origin,
                x_direction: frame.x_axis,
                normal: plane.normal(),
            },
            vertices: vertices.into_values().collect(),
            curves,
        },
        sources,
    ))
}
fn curve(
    sketch: &Sketch2D,
    geometry: OwnedSketchGeometry<'_>,
    coincidence: &incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    match geometry {
        OwnedSketchGeometry::Line(line) => certified::profile_line(
            sketch,
            SketchDirectedLine {
                line: line.id,
                reversed: false,
            },
            coincidence,
            vertices,
        ),
        OwnedSketchGeometry::Circle(circle) => Ok(ProfileCurve::Circle {
            id: ProfileCurveId(circle.id.0),
            center: point(sketch, circle.center)?,
            radius: circle.radius,
        }),
        OwnedSketchGeometry::Arc(arc) => arc_curve(sketch, arc, coincidence, vertices),
        OwnedSketchGeometry::Ellipse(ellipse) => {
            let major = ellipse.radii()[0];
            Ok(ProfileCurve::Ellipse {
                id: ProfileCurveId(ellipse.id().0),
                center: point(sketch, ellipse.center())?,
                major_axis: ellipse.major_axis().map(|value| value * major),
                minor_radius: ellipse.radii()[1],
            })
        }
        OwnedSketchGeometry::Spline(spline) => Err(unsupported(spline.id())),
        OwnedSketchGeometry::OffsetSpline(spline) => Err(unsupported(spline.id())),
        OwnedSketchGeometry::Point(point) => Err(unsupported(point.id)),
    }
}
fn arc_curve(
    sketch: &Sketch2D,
    arc: SketchArc2D,
    coincidence: &incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    Ok(ProfileCurve::Arc {
        id: ProfileCurveId(arc.id.0),
        start: certified::profile_vertex(coincidence, vertices, arc.start)?,
        end: certified::profile_vertex(coincidence, vertices, arc.end)?,
        center: point(sketch, arc.center)?,
        sense: if arc.counter_clockwise {
            ArcSense::CounterClockwise
        } else {
            ArcSense::Clockwise
        },
    })
}
fn point(sketch: &Sketch2D, id: SketchEntityId) -> Result<[f64; 2], EvaluationFailure> {
    sketch
        .point(id)
        .map(|point| point.position)
        .map_err(|cause| Phase::Input.sketch(Entity::SketchEntity(id.0), cause))
}
fn unsupported(id: SketchEntityId) -> EvaluationFailure {
    Phase::Input.refused(Kind::Unsupported, Entity::SketchEntity(id.0), format!("sketch curve {id:?}: expected a line, circle, arc, or ellipse for certified region input"))
}

#[cfg(test)]
#[path = "../../tests/unit/region_input.rs"]
mod tests;
