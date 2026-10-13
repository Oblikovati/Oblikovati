// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    history::{SketchDimensionCreation, SketchDimensionCreationError},
    sketch::{SketchDimensionalConstraintCreation, SketchError},
};
use oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;
pub(super) fn append(
    snapshot: &mut DocumentSnapshot,
    recipe: &SketchPlacementRecipe,
    locks: &[Option<SketchPlacementFieldLock>],
    report: &mut SketchPlacementCreationReport,
    options: SolveOptions,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), SketchPlacementCreationError> {
    for (index, field) in recipe.fields.iter().enumerate() {
        let failure = |cause| SketchPlacementCreationError::Field {
            label: field.label.clone(),
            cause,
        };
        let operands = field
            .binding
            .resolve(&report.points, &report.members)
            .map_err(|cause| failure(cause.into()))?;
        let Some(lock) = locks
            .get(index)
            .and_then(Option::as_ref)
            .filter(|lock| !lock.expression.is_empty())
        else {
            continue;
        };
        let Some(operands) = operands else {
            continue;
        };
        let created =
            append_field(snapshot, operands, lock, report.sketch, options, ids).map_err(failure)?;
        report.dimensions.push((index, created));
    }
    Ok(())
}
fn append_field(
    snapshot: &mut DocumentSnapshot,
    operands: SketchDimensionalConstraintCreation,
    lock: &SketchPlacementFieldLock,
    sketch: DocumentSketchId,
    options: SolveOptions,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchDimensionCreationReport, SketchDimensionCreationError> {
    let input = SketchDimensionCreation {
        operands,
        expression: lock.expression.clone(),
        reference: false,
        text_point: lock.text_point,
    };
    let before = snapshot
        .analyze_document_sketch(sketch, options.rank_tolerance)?
        .redundant_equations;
    let mut created = snapshot.create_sketch_dimension(sketch, &input, options, ids)?;
    // A locked placement input must be evaluable even when redundant dimensions remain driving.
    if let Some(cause) = created.unavailable_targets.first() {
        return Err(cause.clone().into());
    }
    resolve_redundancy(snapshot, sketch, before, &mut created, options)?;
    Ok(created)
}
fn resolve_redundancy(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    before: usize,
    created: &mut SketchDimensionCreationReport,
    options: SolveOptions,
) -> Result<(), SketchError> {
    if snapshot
        .effective_sketch_settings()
        .over_constrained_behavior
        == OVER_CONSTRAINED_APPLY_DRIVING
    {
        return Ok(());
    }
    let after = snapshot
        .analyze_document_sketch(sketch, options.rank_tolerance)?
        .redundant_equations;
    if after <= before {
        return Ok(());
    }
    snapshot
        .sketches
        .set_dimension_reference(sketch, created.constraint, true)?;
    let solved = snapshot.solve_one_document_sketch(sketch, options, || false)?;
    snapshot.sketches = solved.candidate;
    created.degrees_of_freedom = snapshot
        .analyze_document_sketch(sketch, options.rank_tolerance)?
        .degrees_of_freedom;
    created.measured = snapshot
        .sketches
        .get(sketch)?
        .geometry()
        .measure_dimension(created.constraint)?;
    Ok(())
}
