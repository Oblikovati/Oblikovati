// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn validate(
    args: &RedefineWorkPlaneArgs,
    constructor: &str,
    count: i64,
) -> Result<(), Error> {
    let scalars = args.scalars.as_deref().unwrap_or_default();
    if !scalars.is_empty() {
        return Err(Error(format!(
            "{constructor} scalar edits {scalars:?}: expected no scalar slots"
        )));
    }
    validate_repicks(args, constructor, count)
}
pub(in crate::datums) fn validate_repicks(
    args: &RedefineWorkPlaneArgs,
    constructor: &str,
    count: i64,
) -> Result<(), Error> {
    let edits = args.repick.as_deref().unwrap_or_default();
    let slots = edits
        .iter()
        .map(|edit| edit.slot)
        .collect::<std::collections::BTreeSet<_>>();
    if slots.len() != edits.len() || slots.iter().any(|slot| !(0..count).contains(slot)) {
        return Err(Error(format!(
            "{constructor} reference edits {edits:?}: expected unique slots in 0..{count}"
        )));
    }
    Ok(())
}

pub(super) fn slots(labels: &[(&str, &str)]) -> Vec<WorkPlaneRefSlot> {
    labels
        .iter()
        .enumerate()
        .map(|(index, (label, kind))| WorkPlaneRefSlot {
            index: index as i64,
            label: (*label).into(),
            kind: (*kind).into(),
        })
        .collect()
}

pub(super) fn references<const COUNT: usize>(
    request: &CreateWorkPlaneArgs,
) -> Result<&[String; COUNT], Error> {
    let references = request.refs.as_deref().unwrap_or_default();
    references.try_into().map_err(|_| Error(format!(
        "plane {:?} references {references:?}: expected exactly {COUNT} ordered constructor references",
        request.kind
    )))
}

/// Resolve one slot after the caller validates unique repicks; example: `repick(snapshot, args, (0, source.line), input::axis_reference)?`.
pub(in crate::datums) fn repick<Reference: Copy>(
    snapshot: &DocumentSnapshot,
    args: &RedefineWorkPlaneArgs,
    slot: (i64, Reference),
    parse: fn(&DocumentSnapshot, &str) -> Result<Reference, Error>,
) -> Result<Reference, Error> {
    args.repick
        .as_deref()
        .unwrap_or_default()
        .iter()
        .find(|edit| edit.slot == slot.0)
        .map_or(Ok(slot.1), |edit| parse(snapshot, &edit.ref_))
}
