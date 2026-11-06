// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::selection::SelectionFilter;
use oblikovati_model::sketch::SketchEntityId;
#[test]
fn restamping_keeps_membership_under_a_narrowed_filter() {
    let sketch = DocumentSketchId(1);
    let target = SketchPickTarget::Entity(SketchEntityId(2));
    let mut picks = SelectionSet::default();
    picks.add(SketchPick::checked(17, 0, sketch, target));
    picks.set_filter(SelectionFilter::None);
    let restored = restamped_editor_picks(&picks, 17, 9);
    assert_eq!(restored.filter(), &SelectionFilter::None);
    assert_eq!(restored.items().len(), 1);
    assert_eq!(restored.items()[0].target(), target);
    assert!(restored.validate_scope(17, 9, sketch).is_ok());
    assert!(restored.validate_scope(17, 0, sketch).is_err());
}
