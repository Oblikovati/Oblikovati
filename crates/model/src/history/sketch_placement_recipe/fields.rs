// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    history::{SketchDimensionCreation, SketchDimensionCreationError},
    sketch::SketchDimensionalConstraintCreation,
};
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
    let created = snapshot.create_placement_dimension(sketch, &input, options, ids)?;
    // A locked placement input must be evaluable even when redundant dimensions remain driving.
    if let Some(cause) = created.unavailable_targets.first() {
        return Err(cause.clone().into());
    }
    Ok(created)
}
