// SPDX-License-Identifier: GPL-2.0-only
use super::planes::axis_point::{self, AxisPointPlaneConstructor as Constructor};
use super::*;

pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    args: RedefineWorkPlaneArgs,
) -> Result<PlaneRevision, Error> {
    let feature = placed_plane(snapshot, args.index)?;
    Ok(PlaneRevision {
        owner: feature.id(),
        index: args.index,
        geometry: geometry_revision(snapshot, feature, &args)?,
        display: display::prepare(feature.plane_display_settings(), &args)?,
    })
}
fn offset_definition(
    definition: WorkGeometryDefinition,
) -> Result<(WorkPlaneReference, DatumLength), Error> {
    let WorkGeometryDefinition::Offset { base, distance } = definition else {
        return Err(Error(format!(
            "plane definition {definition:?}: expected a supported offset constructor for scalar/reference edits"
        )));
    };
    Ok((base, distance))
}

fn geometry_revision(
    snapshot: &DocumentSnapshot,
    feature: &AuthoredFeature<WorkGeometryDefinition>,
    args: &RedefineWorkPlaneArgs,
) -> Result<Option<PlaneGeometryRevision>, Error> {
    let scalars = args.scalars.as_deref().unwrap_or_default();
    let repicks = args.repick.as_deref().unwrap_or_default();
    if scalars.is_empty() && repicks.is_empty() {
        return Ok(None);
    }
    revise_geometry(snapshot, *feature.definition(), args).map(Some)
}
fn revise_geometry(
    snapshot: &DocumentSnapshot,
    definition: WorkGeometryDefinition,
    args: &RedefineWorkPlaneArgs,
) -> Result<PlaneGeometryRevision, Error> {
    use WorkGeometryDefinition::*;
    let scalars = args.scalars.as_deref().unwrap_or_default();
    let repicks = args.repick.as_deref().unwrap_or_default();
    let revised = match definition {
        NormalToAxis(source) => {
            axis_point::revise(snapshot, source, args, Constructor::NormalToAxis)
        }
        LineThroughPoint(source) => {
            axis_point::revise(snapshot, source, args, Constructor::ContainsLine)
        }
        ThroughThreePoints(points) => planes::three_points::revise(snapshot, points, args),
        ParallelThroughPoint(source) => planes::parallel::revise(snapshot, source, args),
        _ => {
            return prepare_offset(snapshot, definition, scalars, repicks);
        }
    }?;
    Ok(PlaneGeometryRevision::Definition(revised))
}

fn prepare_offset(
    snapshot: &DocumentSnapshot,
    definition: WorkGeometryDefinition,
    scalars: &[ScalarEdit],
    repicks: &[SlotRepick],
) -> Result<PlaneGeometryRevision, Error> {
    let (base, distance) = offset_definition(definition)?;
    let expression = scalar_source(snapshot, scalars)?;
    let base = revised_base(snapshot, base, repicks)?;
    Ok(PlaneGeometryRevision::Offset(OffsetRevision {
        base,
        distance,
        expression,
    }))
}

pub(in crate::datums) fn placed_plane(
    snapshot: &DocumentSnapshot,
    index: i64,
) -> Result<&AuthoredFeature<WorkGeometryDefinition>, Error> {
    let position = index
        .checked_sub(3)
        .and_then(|value| usize::try_from(value).ok());
    position.and_then(|position| rows::entries(snapshot, DatumCollection::Planes).nth(position))
        .ok_or_else(|| Error(format!("plane index {index}: expected a current placed plane index >= 3; origin planes are immutable")))
}

fn scalar_source(
    snapshot: &DocumentSnapshot,
    edits: &[ScalarEdit],
) -> Result<Option<String>, Error> {
    let Some(edit) = single_slot_edit(edits, "offset scalar slot 0")? else {
        return Ok(None);
    };
    if edit.index != 0 {
        return Err(Error(format!(
            "scalar slot {}: expected offset scalar slot 0",
            edit.index
        )));
    }
    scalars::validate_offset(snapshot, &edit.value)?;
    Ok(Some(edit.value.clone()))
}

fn revised_base(
    snapshot: &DocumentSnapshot,
    original: WorkPlaneReference,
    edits: &[SlotRepick],
) -> Result<WorkPlaneReference, Error> {
    let Some(edit) = single_slot_edit(edits, "base plane slot 0")? else {
        return Ok(original);
    };
    if edit.slot != 0 {
        return Err(Error(format!(
            "reference slot {}: expected base plane slot 0",
            edit.slot
        )));
    }
    planes::plane_reference(snapshot, &edit.ref_)
}
fn single_slot_edit<'a, T: std::fmt::Debug>(
    edits: &'a [T],
    expected: &str,
) -> Result<Option<&'a T>, Error> {
    match edits {
        [] => Ok(None),
        [edit] => Ok(Some(edit)),
        _ => Err(Error(format!(
            "edits {edits:?}: expected at most one unique {expected}"
        ))),
    }
}
