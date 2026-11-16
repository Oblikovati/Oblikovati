// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, sketch_primitive::append_primitive};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlanarConstraint, Sketch2D, SketchCornerConstruction, SketchCornerSeed, SketchCornerSize,
        SketchEntityId, SketchError, SketchLineEnd, SketchPointInput, SketchPrimitiveCreation,
    },
};

/// A certified corner seed and explicit source endpoints to replace.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchCornerCreation {
    pub lines: [SketchEntityId; 2],
    pub ends: [SketchLineEnd; 2],
    pub size: SketchCornerSize,
    pub construction: bool,
}
/// New blend entity and the persistent virtual corner and trimmed endpoints.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchCornerCreationReport {
    pub entity: SketchEntityId,
    pub points: Vec<SketchEntityId>,
    pub virtual_corners: [SketchEntityId; 2],
    pub trimmed: [SketchEntityId; 2],
}
impl DocumentSnapshot {
    /// Apply one certified corner to an isolated snapshot; example: `snapshot.create_sketch_corner(sketch, &input, ids)?`.
    pub fn create_sketch_corner(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchCornerCreation,
        constructor: &dyn SketchCornerConstruction,
        residual_bound: f64,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchCornerCreationReport, SketchError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        let points = validate_corner_input(&geometry, input)?;
        let seed = constructor
            .construct_corner(points, input.size, residual_bound)
            .map_err(|e| SketchError::Definition(e.to_string()))?;
        let report = append_corner(&mut geometry, sketch, input, seed, ids)?;
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok(report)
    }
}
fn validate_corner_input(
    geometry: &Sketch2D,
    input: &SketchCornerCreation,
) -> Result<[[f64; 2]; 3], SketchError> {
    if input.lines[0] == input.lines[1] {
        return Err(SketchError::Definition(format!(
            "corner lines {:?}: expected distinct line identities",
            input.lines
        )));
    }
    let points = std::array::from_fn::<_, 2, _>(|i| {
        corner_line_points(geometry, input.lines[i], input.ends[i])
    });
    let [first, second] = points;
    let (first, second) = (first?, second?);
    if first[0] != second[0] {
        return Err(SketchError::Definition(format!(
            "corner endpoints {:?} and {:?}: expected coincident seed positions",
            first[0], second[0]
        )));
    }
    Ok([first[0], first[1], second[1]])
}
fn corner_line_points(
    geometry: &Sketch2D,
    line: SketchEntityId,
    end: SketchLineEnd,
) -> Result<[[f64; 2]; 2], SketchError> {
    let line = geometry.line(line)?;
    let index = if end == SketchLineEnd::First { 0 } else { 1 };
    Ok([
        geometry.point(line.endpoints[index])?.position,
        geometry.point(line.endpoints[1 - index])?.position,
    ])
}
fn append_corner(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    input: &SketchCornerCreation,
    seed: SketchCornerSeed,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchCornerCreationReport, SketchError> {
    let [first, second] = detach_ends(geometry, input, seed, ids)?;
    let corners = [first.previous, second.previous];
    let trimmed = [first.created, second.created];
    let primitive = corner_primitive(input, seed, trimmed);
    let report = append_primitive(geometry, sketch, &primitive, ids)
        .map_err(|e| SketchError::Definition(e.to_string()))?;
    retain_corner_incidence(geometry, sketch, input.lines, corners, report.entity, ids)?;
    if seed.arc.is_some() {
        add_fillet_tangencies(geometry, sketch, input.lines, report.entity, ids)?;
    }
    Ok(SketchCornerCreationReport {
        entity: report.entity,
        points: report.points,
        virtual_corners: corners,
        trimmed,
    })
}
fn detach_ends(
    geometry: &mut Sketch2D,
    input: &SketchCornerCreation,
    seed: SketchCornerSeed,
    ids: &mut RecipeReservations<'_>,
) -> Result<[crate::sketch::SketchLineEndpointReport; 2], SketchError> {
    let mut result = Vec::new();
    for i in 0..2 {
        let report = geometry
            .replace_line_endpoint(
                input.lines[i],
                input.ends[i],
                seed.endpoints[i],
                &mut || ids.allocate_sketch_entity(),
            )
            .map_err(|e| SketchError::Definition(e.to_string()))?;
        result.push(report);
    }
    result
        .try_into()
        .map_err(|_| SketchError::Definition("corner endpoint count: expected two".into()))
}
fn corner_primitive(
    input: &SketchCornerCreation,
    seed: SketchCornerSeed,
    trimmed: [SketchEntityId; 2],
) -> SketchPrimitiveCreation {
    let endpoints = trimmed.map(SketchPointInput::Owned);
    if let Some((center, counter_clockwise)) = seed.arc {
        SketchPrimitiveCreation::Arc {
            center: SketchPointInput::New(center),
            endpoints,
            counter_clockwise,
            construction: input.construction,
        }
    } else {
        SketchPrimitiveCreation::Line {
            endpoints,
            construction: input.construction,
            centerline: false,
        }
    }
}
fn retain_corner_incidence(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    lines: [SketchEntityId; 2],
    corners: [SketchEntityId; 2],
    owner: SketchEntityId,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), SketchError> {
    for i in 0..2 {
        let id = ids
            .allocate_sketch_constraint(sketch)
            .map_err(|e| SketchError::Definition(e.to_string()))?;
        geometry.add_constraint(
            id,
            PlanarConstraint::PointOnLine {
                point: corners[i],
                line: lines[i],
            },
        )?;
        geometry.own_relation(owner, id)?;
    }
    if corners[0] != corners[1] {
        let id = ids
            .allocate_sketch_constraint(sketch)
            .map_err(|e| SketchError::Definition(e.to_string()))?;
        geometry.add_constraint(id, PlanarConstraint::Coincident { points: corners })?;
        geometry.own_relation(owner, id)?;
    }
    Ok(())
}
fn add_fillet_tangencies(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    lines: [SketchEntityId; 2],
    arc: SketchEntityId,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), SketchError> {
    for line in lines {
        let id = ids
            .allocate_sketch_constraint(sketch)
            .map_err(|e| SketchError::Definition(e.to_string()))?;
        geometry.add_line_tangent(id, line, arc)?;
        geometry.own_relation(arc, id)?;
    }
    Ok(())
}
