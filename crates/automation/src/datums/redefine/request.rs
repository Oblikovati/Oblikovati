// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    args: RedefineWorkPlaneArgs,
) -> Result<PlaneRevision, Error> {
    refuse_display_edits(&args)?;
    let feature = placed_plane(snapshot, args.index)?;
    let (base, distance) = offset_definition(*feature.definition())?;
    let expression = scalar_source(snapshot, args.scalars.as_deref().unwrap_or_default())?;
    let base = revised_base(snapshot, base, args.repick.as_deref().unwrap_or_default())?;
    Ok(PlaneRevision {
        owner: feature.id(),
        index: args.index,
        base,
        distance,
        expression,
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

fn refuse_display_edits(args: &RedefineWorkPlaneArgs) -> Result<(), Error> {
    if args.auto_resize.is_some() || args.grounded.is_some() || args.size.is_some() {
        return Err(Error(format!(
            "plane edit {args:?}: expected scalar/reference edits; display rectangle and associativity flags are not yet supported"
        )));
    }
    Ok(())
}

fn placed_plane(
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
