// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn view_display_history_refuses_navigation_without_a_display_owner() {
    let mut session = view_session(false, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    assert_eq!(
        session.undo(),
        Err(crate::SessionError::PartViewDisplayOwner)
    );
    assert!(!session.body_visible("aa"));
}

#[test]
fn tagged_visibility_undo_and_redo_restore_live_and_stored_state() {
    use crate::PartViewNavigationDirection::{Redo, Undo};
    let mut session = view_session(false, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    assert!(
        session
            .prepare_part_view_navigation(Undo, &BTreeSet::new())
            .is_err()
    );
    let undo = session
        .prepare_part_view_navigation(Undo, &available)
        .unwrap();
    assert!(undo.camera().is_none());
    session.commit_part_view_navigation(undo, None).unwrap();
    assert!(session.body_visible("aa"));
    assert!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .get(1)
            .unwrap()
            .hidden_body_keys()
            .is_empty()
    );
    let redo = session
        .prepare_part_view_navigation(Redo, &available)
        .unwrap();
    session.commit_part_view_navigation(redo, None).unwrap();
    assert!(!session.body_visible("aa"));
}

#[test]
fn ordinary_history_does_not_erase_locked_transient_visibility() {
    let mut session = view_session(true, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    session
        .run("Ordinary metadata", |snapshot, _| {
            snapshot.body_names.insert("aa".into(), "Body".into());
            Ok::<_, String>(())
        })
        .unwrap();
    session.undo().unwrap();
    assert!(!session.body_visible("aa"));
    session.redo().unwrap();
    assert!(!session.body_visible("aa"));
}

#[test]
fn view_switch_undo_and_redo_return_the_exact_cameras() {
    use crate::PartViewNavigationDirection::{Redo, Undo};
    let mut session = view_session(false, false);
    let available = BTreeSet::new();
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    let mut target = camera();
    target.scale = 22.;
    session
        .commit_part_view_activation(prepared, target.clone())
        .unwrap();
    let undo = session
        .prepare_part_view_navigation(Undo, &available)
        .unwrap();
    assert_eq!(undo.camera(), Some(&camera()));
    session
        .commit_part_view_navigation(undo, Some(&camera()))
        .unwrap();
    assert_eq!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .id(),
        1
    );
    let redo = session
        .prepare_part_view_navigation(Redo, &available)
        .unwrap();
    assert_eq!(redo.camera(), Some(&target));
    session
        .commit_part_view_navigation(redo, Some(&target))
        .unwrap();
    assert_eq!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .id(),
        6
    );
}

#[test]
fn tagged_navigation_rejects_missing_keys_and_veto_without_partial_changes() {
    use crate::PartViewNavigationDirection::Undo;
    let mut session = view_session(false, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    let undo = session
        .prepare_part_view_navigation(Undo, &available)
        .unwrap();
    session.observer.reject = true;
    assert!(session.commit_part_view_navigation(undo, None).is_err());
    assert!(!session.body_visible("aa"));
    assert!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .get(1)
            .unwrap()
            .hidden_body_keys()
            .contains("aa")
    );
}
