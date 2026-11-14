// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeSet;
// These indices identify fixture inputs only. They are not persistent application keys.
/// Recover source identities through the released operation history.
/// Example: `let parents = original_region_parents(&edges, &arrangement)?;`.
pub(super) fn original_region_parents(
    edges: &IndexedMapOfShape,
    arrangement: &Arrangement,
) -> Result<Vec<usize>, Failure> {
    let mut found = BTreeSet::new();
    for index in 1..=edges.extent() {
        found.extend(source_split_parents(edges.find_key(index)?, arrangement)?);
    }
    Ok(found.into_iter().collect())
}
fn source_split_parents(split: &Shape, arrangement: &Arrangement) -> Result<Vec<usize>, Failure> {
    let mut found = Vec::new();
    for index in 1..=arrangement.sources.extent() {
        let source = arrangement.sources.find_key(index)?;
        let modified = arrangement.fuse.modified(source)?;
        if split.is_same(source) || modified.iter().any(|shape| shape.is_same(split)) {
            found.push(index);
        }
    }
    assert!(
        !found.is_empty(),
        "split edge parents={found:?}, source count={}: expected recorded original input parents",
        arrangement.sources.extent()
    );
    Ok(found)
}
/// Refuse an arrangement that loses a fixture input from its edge history.
/// Example: `require_region_history_coverage(&arrangement)?;`.
pub(super) fn require_region_history_coverage(arrangement: &Arrangement) -> Result<(), Failure> {
    let mut found = BTreeSet::new();
    for index in 1..=arrangement.split.extent() {
        found.extend(source_split_parents(
            arrangement.split.find_key(index)?,
            arrangement,
        )?);
    }
    assert_eq!(found.len(), arrangement.sources.extent());
    Ok(())
}
