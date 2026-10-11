// SPDX-License-Identifier: GPL-2.0-only
use super::planes::reference_only::repick;
use super::*;
use oblikovati_model::feature::work_geometry::{DatumAngle, WorkPlaneLineAngle};
pub(super) struct LineAngleRevision {
    pub(super) source: WorkPlaneLineAngle,
    pub(super) expression: Option<String>,
}
pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    mut source: WorkPlaneLineAngle,
    args: &RedefineWorkPlaneArgs,
) -> Result<PlaneGeometryRevision, Error> {
    let expression = request::scalar_source(
        snapshot,
        args.scalars.as_deref().unwrap_or_default(),
        OwnedDatumScalar::Angle,
    )?;
    planes::reference_only::validate_repicks(args, WORK_PLANE_LINE_PLANE_ANGLE.0.as_ref(), 2)?;
    source.line = repick(snapshot, args, (0, source.line), input::axis_reference)?;
    source.base = repick(snapshot, args, (1, source.base), planes::plane_reference)?;
    Ok(PlaneGeometryRevision::LineAngle(LineAngleRevision {
        source,
        expression,
    }))
}
pub(super) fn install(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    revision: &LineAngleRevision,
) -> Result<(), Error> {
    let (source, added) = revised_source(snapshot, ids, owner, revision)?;
    let definition = WorkGeometryDefinition::AtAngleAboutAxis(source);
    snapshot
        .work_geometry
        .replace(owner, definition, definition.reads())
        .map_err(api_error)?;
    if let Some(parameter) = added {
        assign_scalar_owner(snapshot, owner, OwnedDatumScalar::Angle, parameter)?;
    }
    Ok(())
}
fn revised_source(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: FeatureId,
    revision: &LineAngleRevision,
) -> Result<(WorkPlaneLineAngle, Option<u64>), Error> {
    let mut source = revision.source;
    let Some(expression) = &revision.expression else {
        return Ok((source, None));
    };
    let (parameter, added) = OwnedDatumScalar::Angle.revise(snapshot, ids, owner, expression)?;
    source.angle = DatumAngle::parameter(parameter);
    Ok((source, added.then_some(parameter)))
}
