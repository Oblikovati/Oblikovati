// SPDX-License-Identifier: GPL-2.0-only
//! Convert current document-owned sketch identities to the released profile operation.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId, Resolution,
    SketchProfile,
};
use oblikovati_model::feature::sketch_prism::ResolvedSketchPrism;
use std::collections::BTreeMap;

/// Evaluate a current, fully constrained saved prism with an explicit modelling resolution.
/// Example: `retain_certified_sketch_prism(&document, &definition, resolution)?`.
pub fn retain_certified_sketch_prism(
    document: &DocumentSnapshot,
    definition: &SketchPrismDefinition,
    resolution: Resolution,
) -> Result<crate::CertifiedPrismSnapshot, EvaluationFailure> {
    let resolved = definition.resolve(document).map_err(|cause| {
        Phase::Input
            .refused(Kind::InvalidInput, Entity::Body, cause.to_string())
            .with_sketch_identity(definition.sketch())
    })?;
    let sketch = resolved.sketch.sketch.id();
    evaluate(resolved, resolution)
        .map_err(|cause| cause.for_sketch(sketch, format!("document sketch {sketch:?}")))
}

fn evaluate(
    input: ResolvedSketchPrism<'_>,
    resolution: Resolution,
) -> Result<crate::CertifiedPrismSnapshot, EvaluationFailure> {
    let plane = input.sketch.plane;
    let profile = selected_profile(input.sketch.sketch.geometry(), input.selection, plane)?;
    let vector = crate::prism::placement::model_displacement(plane, input.displacement)?;
    crate::retain_certified_profile_prism(
        &profile,
        input.selection,
        [vector.x(), vector.y(), vector.z()],
        resolution,
    )
}

fn selected_profile(
    sketch: &oblikovati_model::sketch::Sketch2D,
    selection: &SketchPrismSelection,
    plane: oblikovati_model::sketch::SketchPlane,
) -> Result<SketchProfile, EvaluationFailure> {
    let mut vertices = BTreeMap::new();
    let mut curves = Vec::new();
    for line in std::iter::once(&selection.outer)
        .chain(&selection.holes)
        .flatten()
    {
        curves.push(profile_line(sketch, *line, &mut vertices)?);
    }
    let recipe = plane.recipe();
    Ok(SketchProfile {
        plane: ProfilePlane {
            origin: recipe.origin,
            x_direction: recipe.x_axis,
            normal: plane.normal(),
        },
        vertices: vertices.into_values().collect(),
        curves,
    })
}

fn profile_line(
    sketch: &oblikovati_model::sketch::Sketch2D,
    directed: SketchDirectedLine,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    let line = boundary::require_boundary_line(sketch, directed.line)?;
    for id in line.endpoints {
        let point = sketch
            .point(id)
            .map_err(|cause| Phase::Input.sketch(Entity::SketchEntity(id.0), cause))?;
        vertices.insert(
            ProfileVertexId(id.0),
            ProfileVertex {
                id: ProfileVertexId(id.0),
                point: point.position,
            },
        );
    }
    Ok(ProfileCurve::Line {
        id: ProfileCurveId(line.id.0),
        start: ProfileVertexId(line.endpoints[0].0),
        end: ProfileVertexId(line.endpoints[1].0),
    })
}
