// SPDX-License-Identifier: GPL-2.0-only
//! Isolated conversion of legacy sketch recipes. This does not migrate or open a package.
mod source;
use super::{DocumentPackage, PackageCodecError, RestoredParameters};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    quantity::Dimension,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
};
use source::{LegacyConstraint, LegacyDimension, LegacyEntity, LegacyPoint, LegacySketch};

/// An isolated candidate; feature conversion and atomic package migration are still required.
#[derive(Debug)]
pub struct LegacySketchCandidate {
    pub sketches: DocumentSketches,
    pub parameters: RestoredParameters,
    pub creation_sequences: Vec<(DocumentSketchId, u64)>,
}

impl DocumentPackage {
    /// Convert supported legacy sketches without changing or adopting the source package.
    /// Example: `let candidate = package.prepare_legacy_sketch_candidate()?;`.
    pub fn prepare_legacy_sketch_candidate(
        &self,
    ) -> Result<LegacySketchCandidate, PackageCodecError> {
        let model = self.root.get("model");
        if model.and_then(|m| m.get("planarSketches")).is_some() {
            return Err(failure(
                "model.planarSketches is present; expected an unmigrated legacy model",
            ));
        }
        let source = model.and_then(|m| m.get("sketches"));
        let rows: Vec<LegacySketch> = match source {
            None | Some(serde_yaml_ng::Value::Null) => Vec::new(),
            Some(value) => serde_yaml_ng::from_value(value.clone()).map_err(failure)?,
        };
        let mut candidate = LegacySketchCandidate {
            sketches: DocumentSketches::default(),
            parameters: self.restore_document_parameters()?,
            creation_sequences: Vec::new(),
        };
        for row in rows {
            append_sketch(&mut candidate, row)?;
        }
        Ok(candidate)
    }
}
fn failure(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "legacy model.sketches: expected a supported, lossless sketch recipe: {error}"
    ))
}
fn append_sketch(
    candidate: &mut LegacySketchCandidate,
    row: LegacySketch,
) -> Result<(), PackageCodecError> {
    let plane = SketchPlane::new(
        row.plane.origin,
        row.plane.x_axis,
        row.plane.y_axis,
        Default::default(),
    )
    .map_err(failure)?;
    let mut geometry = legacy_points(row.points)?;
    for entity in row.entities {
        append_entity(&mut geometry, entity)?;
    }
    append_constraints(candidate, &mut geometry, row.constraints, row.dimensions)?;
    let id = DocumentSketchId(row.id);
    candidate
        .sketches
        .add(id, &row.name, plane, geometry)
        .map_err(failure)?;
    candidate.creation_sequences.push((id, row.seq));
    Ok(())
}
fn legacy_points(points: Vec<LegacyPoint>) -> Result<Sketch2D, PackageCodecError> {
    let mut geometry = Sketch2D::default();
    for point in points {
        let id = SketchEntityId(point.id);
        if point.standalone {
            geometry.add_point(id, [point.x, point.y])
        } else {
            geometry.add_defining_point(id, [point.x, point.y])
        }
        .map_err(failure)?;
    }
    Ok(geometry)
}
fn append_constraints(
    candidate: &mut LegacySketchCandidate,
    geometry: &mut Sketch2D,
    relations: Vec<LegacyConstraint>,
    dimensions: Vec<LegacyDimension>,
) -> Result<(), PackageCodecError> {
    let mut number = 0_u64;
    for relation in relations {
        number += 1;
        append_relation(geometry, number, relation)?;
    }
    for dimension in dimensions {
        number += 1;
        append_dimension(candidate, geometry, number, dimension)?;
    }
    Ok(())
}
fn pair(kind: &str, ids: Vec<u64>) -> Result<[SketchEntityId; 2], PackageCodecError> {
    let ids: [u64; 2] = ids
        .try_into()
        .map_err(|ids| failure(format!("{kind} operands {ids:?}: expected two point IDs")))?;
    Ok(ids.map(SketchEntityId))
}
fn append_entity(geometry: &mut Sketch2D, entity: LegacyEntity) -> Result<(), PackageCodecError> {
    if entity.kind != "line" {
        return Err(failure(format!(
            "entity kind {:?}: expected a supported line",
            entity.kind
        )));
    }
    geometry
        .add_line(SketchEntityId(entity.id), pair("line", entity.points)?)
        .map_err(failure)
}
fn append_relation(
    geometry: &mut Sketch2D,
    number: u64,
    item: LegacyConstraint,
) -> Result<(), PackageCodecError> {
    let relation = match item.kind.as_str() {
        "fix" => fixed_point(geometry, item.points)?,
        "horizontal" => PlanarConstraint::Horizontal {
            points: pair("horizontal", item.points)?,
        },
        "vertical" => PlanarConstraint::Vertical {
            points: pair("vertical", item.points)?,
        },
        "coincident" => PlanarConstraint::Coincident {
            points: pair("coincident", item.points)?,
        },
        other => {
            return Err(failure(format!(
                "constraint kind {other:?}: expected a supported point relation"
            )));
        }
    };
    geometry
        .add_constraint(SketchConstraintId(number), relation)
        .map_err(failure)
}
fn fixed_point(geometry: &Sketch2D, ids: Vec<u64>) -> Result<PlanarConstraint, PackageCodecError> {
    let [id]: [u64; 1] = ids
        .try_into()
        .map_err(|ids| failure(format!("fix operands {ids:?}: expected one point ID")))?;
    let point = geometry
        .read_index()
        .point(SketchEntityId(id))
        .map_err(failure)?;
    Ok(PlanarConstraint::Fix {
        point: point.id,
        position: point.position,
    })
}
fn append_dimension(
    candidate: &mut LegacySketchCandidate,
    geometry: &mut Sketch2D,
    number: u64,
    dimension: LegacyDimension,
) -> Result<(), PackageCodecError> {
    if dimension.kind != "distance" {
        return Err(failure(format!(
            "dimension kind {:?}: expected distance",
            dimension.kind
        )));
    }
    let measure = distance_measure(&dimension.orientation)?;
    let points = pair("distance", dimension.points)?;
    let parameter = dimension_parameter(candidate, &dimension.expression)?;
    geometry
        .add_constraint(
            SketchConstraintId(number),
            PlanarConstraint::Distance {
                points,
                measure,
                parameter,
            },
        )
        .map_err(failure)
}
fn distance_measure(orientation: &str) -> Result<DistanceMeasure, PackageCodecError> {
    match orientation {
        "" | "aligned" => Ok(DistanceMeasure::Aligned),
        "horizontal" => Ok(DistanceMeasure::Horizontal),
        "vertical" => Ok(DistanceMeasure::Vertical),
        other => Err(failure(format!(
            "distance orientation {other:?}: expected aligned, horizontal, or vertical"
        ))),
    }
}
fn dimension_parameter(
    candidate: &mut LegacySketchCandidate,
    expression: &str,
) -> Result<u64, PackageCodecError> {
    let parameter = candidate.parameters.ids.allocate().map_err(failure)?;
    candidate
        .parameters
        .parameters
        .insert_auto_model(parameter, expression)
        .map_err(failure)?;
    let state = candidate
        .parameters
        .parameters
        .get(parameter)
        .expect("new dimension parameter");
    if !state
        .current_value()
        .is_some_and(|value| value.dimension == Dimension::Length)
    {
        return Err(failure(format!(
            "dimension expression {:?} has {:?}: expected a resolved length",
            expression, state.health
        )));
    }
    Ok(parameter)
}
