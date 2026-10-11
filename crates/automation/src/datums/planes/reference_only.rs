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
