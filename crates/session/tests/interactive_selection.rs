// SPDX-License-Identifier: GPL-2.0-only
use SelectionKind::*;
#[path = "interactive_selection/click.rs"]
mod click;
use oblikovati_session::selection::{
    AmbientSelectionFilter, Selectable, SelectionFilter, SelectionKind, SelectionPriority,
    SelectionSet,
};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct AuthoredPick {
    owner: u64,
    entity: u64,
    kind: SelectionKind,
}
impl Selectable for AuthoredPick {
    fn selection_kind(&self) -> SelectionKind {
        self.kind
    }
}
fn pick(owner: u64, entity: u64, kind: SelectionKind) -> AuthoredPick {
    AuthoredPick {
        owner,
        entity,
        kind,
    }
}
const ALL_KINDS: [SelectionKind; 23] = [
    Face,
    Edge,
    Vertex,
    Body,
    Feature,
    Profile,
    SketchEntity,
    WorkPlane,
    Sketch,
    WorkPoint,
    WorkAxis,
    Path,
    Occurrence,
    Constraint,
    Joint,
    Representation,
    ModelState,
    DrawingView,
    PointCloud,
    PointCloudPoint,
    MeshFace,
    SketchDimension,
    SketchConstraint,
];
const EXPECTED_PICK_ORDER: [SelectionKind; 12] = [
    WorkPoint,
    WorkAxis,
    PointCloudPoint,
    Vertex,
    Edge,
    Occurrence,
    Body,
    Face,
    MeshFace,
    WorkPlane,
    Profile,
    SketchEntity,
];
#[test]
fn filters_distinguish_default_subset_and_blocking_for_every_kind() {
    let subset = SelectionFilter::only([Edge, SketchDimension, Edge]);
    assert_eq!(SelectionFilter::only([]), SelectionFilter::default());
    assert!(!SelectionFilter::All.is_restricted());
    assert!(subset.is_restricted());
    assert!(SelectionFilter::None.is_restricted());
    for kind in ALL_KINDS {
        assert!(SelectionFilter::All.accepts(kind));
        assert!(!SelectionFilter::None.accepts(kind));
        assert_eq!(
            subset.accepts(kind),
            [Edge, SketchDimension].contains(&kind)
        );
    }
}
#[test]
fn ordered_identity_includes_owner_and_category_and_deduplicates_repeated_picks() {
    let mut selected = SelectionSet::default();
    assert!(selected.first().is_none());
    let picks = [pick(1, 7, Edge), pick(2, 7, Edge), pick(1, 7, Face)];
    for item in picks {
        assert!(selected.add(item));
    }
    assert!(!selected.add(picks[0]));
    assert!(selected.contains(&picks[1]));
    assert_eq!(selected.items(), &picks);
    assert_eq!(selected.first(), Some(&picks[0]));
    assert!(selected.remove(&picks[1]));
    assert!(!selected.remove(&picks[1]));
    assert_eq!(selected.items(), &[picks[0], picks[2]]);
}
#[test]
fn filters_only_affect_additions_and_toggle_can_remove_a_now_blocked_pick() {
    let mut selected = SelectionSet::default();
    let edge = pick(1, 2, Edge);
    assert!(selected.toggle(edge));
    selected.set_filter(SelectionFilter::only([Face]));
    assert_eq!(selected.items(), &[edge]);
    assert!(selected.toggle(edge));
    assert!(!selected.toggle(edge));
    assert!(selected.add(pick(1, 3, Face)));
    selected.clear();
    assert!(selected.items().is_empty());
    assert!(selected.first().is_none());
    assert_eq!(selected.filter(), &SelectionFilter::only([Face]));
}
#[test]
fn selection_clone_has_independent_membership_and_filter() {
    let mut original = SelectionSet::default();
    original.add(pick(1, 2, SketchEntity));
    let mut copy = original.clone();
    copy.set_filter(SelectionFilter::None);
    copy.clear();
    assert_eq!(original.items(), &[pick(1, 2, SketchEntity)]);
    assert_eq!(original.filter(), &SelectionFilter::All);
    assert!(!copy.add(pick(1, 2, SketchEntity)));
}
#[test]
fn ambient_default_order_preserves_go_snap_priority_and_unlisted_categories_sort_last() {
    let ambient = AmbientSelectionFilter::default();
    assert_eq!(ambient.order(), EXPECTED_PICK_ORDER);
    for (index, kind) in ambient.order().iter().enumerate() {
        assert!(ambient.enabled(*kind));
        assert_eq!(ambient.rank(*kind), index);
    }
    assert_eq!(ambient.rank(SketchDimension), ambient.order().len());
    assert!(!ambient.enabled(SketchDimension));
    assert_eq!(ambient.filter(), SelectionFilter::All);
}
#[test]
fn ambient_enable_disable_reorder_and_presets_share_one_state() {
    let mut ambient = AmbientSelectionFilter::default();
    ambient.move_kind(ambient.rank(Face), 0);
    assert_eq!(ambient.order()[0], Face);
    ambient.set_enabled(Face, false);
    assert!(!ambient.filter().accepts(Face));
    assert!(ambient.filter().accepts(Edge));
    ambient.disable_all();
    assert_eq!(ambient.filter(), SelectionFilter::None);
    ambient.enable_all();
    assert_eq!(ambient.filter(), SelectionFilter::All);
    assert_eq!(ambient.rank(Face), 0);
}
#[test]
fn invalid_reorders_and_unlisted_enablement_leave_managed_window_policy_intact() {
    let mut ambient = AmbientSelectionFilter::default();
    let before = ambient.order().to_vec();
    ambient.move_kind(usize::MAX, 0);
    ambient.move_kind(0, before.len());
    ambient.move_kind(2, 2);
    assert_eq!(ambient.order(), before);
    ambient.set_enabled(SketchDimension, true);
    assert!(ambient.enabled(SketchDimension));
    ambient.disable_all();
    assert!(ambient.enabled(SketchDimension));
    assert_eq!(ambient.filter(), SelectionFilter::None);
}
#[test]
fn every_priority_preset_preserves_rank_and_matches_go_category_matrix() {
    let mut ambient = AmbientSelectionFilter::default();
    ambient.move_kind(7, 0);
    let order = ambient.order().to_vec();
    for (priority, allowed) in priority_cases() {
        ambient.apply_priority(priority);
        assert_eq!(ambient.order(), order);
        assert_priority_matrix(&ambient, priority, &allowed);
    }
}
#[test]
fn running_tool_default_and_explicit_restrictions_override_disabled_ambient() {
    let mut ambient = AmbientSelectionFilter::default();
    ambient.disable_all();
    assert_eq!(
        ambient.pick_filter(false, &SelectionFilter::All),
        SelectionFilter::None
    );
    assert_eq!(
        ambient.pick_filter(true, &SelectionFilter::All),
        SelectionFilter::All
    );
}
#[test]
fn explicit_restrictions_override_ambient_with_and_without_a_tool() {
    let mut ambient = AmbientSelectionFilter::default();
    ambient.disable_all();
    let tool = SelectionFilter::only([SketchEntity, SketchDimension, SketchConstraint]);
    assert_eq!(ambient.pick_filter(false, &tool), tool);
    assert_eq!(ambient.pick_filter(true, &tool), tool);
    assert_eq!(
        ambient.pick_filter(false, &SelectionFilter::None),
        SelectionFilter::None
    );
}

fn priority_cases() -> [(SelectionPriority, Vec<SelectionKind>); 4] {
    [
        (SelectionPriority::General, vec![]),
        (SelectionPriority::Part, vec![Body, Occurrence]),
        (SelectionPriority::Face, vec![Face]),
        (SelectionPriority::Edge, vec![Edge]),
    ]
}
fn assert_priority_matrix(
    ambient: &AmbientSelectionFilter,
    priority: SelectionPriority,
    allowed: &[SelectionKind],
) {
    for kind in ALL_KINDS {
        assert_eq!(
            ambient.filter().accepts(kind),
            priority == SelectionPriority::General || allowed.contains(&kind)
        );
    }
}
