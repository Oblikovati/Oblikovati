// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchCornerCreation, SketchCornerCreationReport, SketchDimensionCreation},
    sketch::{
        MeasuredDimensionOperands, SketchCornerConstruction, SketchCornerSize,
        SketchDimensionalConstraintCreation, SketchEntityId, SketchLineEnd,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;

type CornerAccess<'a> = (
    &'a dyn SketchCornerConstruction,
    oblikovati_model::sketch::SketchConstructionPolicy,
);
struct CornerRequest {
    input: SketchCornerCreation,
    bindings: Vec<(usize, String)>,
    bound: f64,
}
pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    request: AddSketchEntityArgs,
    construction: Option<CornerAccess<'_>>,
) -> Result<Vec<u8>, Error> {
    let (constructor, policy) = construction.ok_or_else(|| {
        Error("sketch corner creation: expected an injected corner construction service".into())
    })?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let snapshot = session.snapshot();
    let prepared = prepare(snapshot, sketch, &request, policy)?;
    let options = session.sketch_solve_options();
    files
        .commands(document)
        .map_err(api_error)?
        .run("Create sketch corner", |snapshot, ids| {
            let report = snapshot
                .create_sketch_corner(sketch, &prepared.input, constructor, prepared.bound, ids)
                .map_err(api_error)?;
            bind_dimensions(snapshot, sketch, &report, &prepared, options, ids)?;
            encode(&AddSketchEntityResult {
                entity_id: report.entity.0,
                kind: request.kind,
                point_i_ds: Some(report.points.iter().map(|id| id.0).collect()),
                ..Default::default()
            })
        })
        .map_err(api_error)
}
fn prepare(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    request: &AddSketchEntityArgs,
    policy: oblikovati_model::sketch::SketchConstructionPolicy,
) -> Result<CornerRequest, Error> {
    let refs = request.entity_refs.as_deref().unwrap_or_default();
    let [first, second] = refs else {
        return Err(Error(format!(
            "corner references {refs:?}: expected exactly two line identities"
        )));
    };
    let lines = [SketchEntityId(*first), SketchEntityId(*second)];
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let ends = matching_ends(geometry, lines)?;
    let first_value = inputs::length_expression(snapshot, &request.radius)?;
    let second_source = if request.distance2.is_empty() {
        &request.radius
    } else {
        &request.distance2
    };
    let size = if request.kind == "fillet" {
        SketchCornerSize::Fillet(first_value)
    } else {
        SketchCornerSize::Chamfer([
            first_value,
            inputs::length_expression(snapshot, second_source)?,
        ])
    };
    let mut bindings = Vec::new();
    if associative::named(snapshot, &request.radius)? {
        bindings.push((0, request.radius.clone()));
    }
    if request.kind == "chamfer" && associative::named(snapshot, second_source)? {
        bindings.push((1, second_source.clone()));
    }
    let bound = policy
        .in_working_units(snapshot.parameters.units().working_scale())
        .map_err(api_error)?
        .maximum_point_residual;
    Ok(CornerRequest {
        input: SketchCornerCreation {
            lines,
            ends,
            size,
            construction: request.construction,
        },
        bindings,
        bound,
    })
}
fn matching_ends(
    geometry: &oblikovati_model::sketch::Sketch2D,
    lines: [SketchEntityId; 2],
) -> Result<[SketchLineEnd; 2], Error> {
    if lines[0] == lines[1] {
        return Err(Error(format!(
            "corner lines {lines:?}: expected distinct identities"
        )));
    }
    let a = geometry.line(lines[0]).map_err(api_error)?.endpoints;
    let b = geometry.line(lines[1]).map_err(api_error)?.endpoints;
    let mut matches = Vec::new();
    for (i, first) in a.into_iter().enumerate() {
        for (j, second) in b.into_iter().enumerate() {
            if geometry
                .points_coincident(first, second)
                .map_err(api_error)?
            {
                matches.push([i, j]);
            }
        }
    }
    let [indices] = matches.as_slice() else {
        return Err(Error(format!(
            "corner lines {lines:?}: expected one shared endpoint, found {}",
            matches.len()
        )));
    };
    Ok(indices.map(|i| {
        if i == 0 {
            SketchLineEnd::First
        } else {
            SketchLineEnd::Last
        }
    }))
}
fn bind_dimensions(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    report: &SketchCornerCreationReport,
    prepared: &CornerRequest,
    options: SolveOptions,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<(), Error> {
    for (index, expression) in &prepared.bindings {
        let operands = match prepared.input.size {
            SketchCornerSize::Fillet(_) => MeasuredDimensionOperands::Radius {
                curve: report.entity,
            },
            SketchCornerSize::Chamfer(_) => MeasuredDimensionOperands::CornerSetback {
                points: [
                    report.virtual_corners[*index],
                    report.trimmed[*index],
                    report.far[*index],
                ],
            },
        };
        let created = snapshot
            .create_sketch_dimension(
                sketch,
                &SketchDimensionCreation {
                    operands: SketchDimensionalConstraintCreation::Measured(operands),
                    expression: expression.clone(),
                    reference: false,
                    text_point: None,
                },
                options,
                ids,
            )
            .map_err(api_error)?;
        snapshot
            .own_corner_relation(sketch, report.entity, created.constraint)
            .map_err(api_error)?;
    }
    Ok(())
}
