// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    types::{ORTHOGRAPHIC_PROJECTION, Point, Vector},
    wire::TransactionEventPayload,
};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
};

struct ViewObserver {
    reject: bool,
}
impl TransactionObserver for ViewObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        if self.reject {
            return Err("view edit refused".into());
        }
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
fn camera() -> PartDesignViewCamera {
    PartDesignViewCamera {
        eye: Point {
            x: 0.,
            y: 0.,
            z: 8.,
        },
        target: Point::default(),
        up: Vector {
            x: 0.,
            y: 1.,
            z: 0.,
        },
        scale: 12.,
        near: 0.5,
        far: 250.,
        projection: ORTHOGRAPHIC_PROJECTION,
    }
}
fn view_session(locked: bool, reject: bool) -> DocumentSession<ViewObserver> {
    let mut views = oblikovati_model::part_design_view::PartDesignViews::default();
    views.copy(1, 6, "Review").unwrap();
    views
        .set_options(&oblikovati_api::wire::SetPartDesignViewOptionsArgs {
            id: 1,
            locked: Some(Box::new(locked)),
            ..Default::default()
        })
        .unwrap();
    let snapshot = DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        assets: Default::default(),
        model_states: None,
        assembly: None,
        drawing: None,
        body_names: Default::default(),
        design_views: Default::default(),
        part_design_views: Some(views),
        sketch_settings: None,
        parameters: oblikovati_model::parameter::ParameterCollection::new(Default::default()),
        properties: Default::default(),
    };
    let ids = RecipeIdAllocator::restore(Some(6), &[6]).unwrap();
    DocumentSession::new(
        1,
        DocumentEditHistory::new(snapshot, ids).unwrap(),
        ViewObserver { reject },
    )
}
#[test]
fn unlocked_view_retains_visibility_and_camera_when_switching() {
    let mut session = view_session(false, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    session
        .commit_part_view_activation(prepared, camera())
        .unwrap();
    let views = session.snapshot().part_design_views.as_ref().unwrap();
    assert_eq!(views.active().unwrap().id(), 6);
    assert_eq!(views.get(1).unwrap().hidden_body_keys(), &available);
    assert_eq!(views.get(1).unwrap().camera(), Some(&camera()));
    assert!(session.body_visible("aa"));
}
#[test]
fn locked_view_discards_transient_visibility_and_camera_when_switching() {
    let mut session = view_session(true, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    session
        .commit_part_view_activation(prepared, camera())
        .unwrap();
    let views = session.snapshot().part_design_views.as_ref().unwrap();
    assert!(views.get(1).unwrap().hidden_body_keys().is_empty());
    assert!(views.get(1).unwrap().camera().is_none());
    assert!(session.body_visible("aa"));
}
#[test]
fn unlocking_does_not_promote_locked_visibility_to_stored_state() {
    let mut session = view_session(true, false);
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    session
        .run("Unlock Primary", |snapshot, _| {
            snapshot.part_design_views.as_mut().unwrap().set_options(
                &oblikovati_api::wire::SetPartDesignViewOptionsArgs {
                    id: 1,
                    locked: Some(Box::new(false)),
                    ..Default::default()
                },
            )
        })
        .unwrap();
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    session
        .commit_part_view_activation(prepared, camera())
        .unwrap();
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
    assert!(session.body_visible("aa"));
}
#[test]
fn switching_back_to_locked_primary_restores_its_saved_camera() {
    let mut session = view_session(false, false);
    session
        .run("Capture and lock Primary", |snapshot, _| {
            let views = snapshot.part_design_views.as_mut().unwrap();
            views.capture_camera(1, camera())?;
            views.set_options(&oblikovati_api::wire::SetPartDesignViewOptionsArgs {
                id: 1,
                locked: Some(Box::new(true)),
                ..Default::default()
            })
        })
        .unwrap();
    let available = BTreeSet::from(["aa".into()]);
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    let to_copy = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    session
        .commit_part_view_activation(to_copy, camera())
        .unwrap();
    let to_primary = session
        .prepare_part_view_activation(1, camera(), &available)
        .unwrap();
    assert_eq!(to_primary.camera(), Some(&camera()));
    session
        .commit_part_view_activation(to_primary, camera())
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
    assert!(session.body_visible("aa"));
}
#[test]
fn rejected_or_stale_switch_keeps_document_and_visibility() {
    let mut session = view_session(false, true);
    session.body_visibility.set("aa", false).unwrap();
    let available = BTreeSet::from(["aa".into()]);
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    assert!(
        session
            .commit_part_view_activation(prepared, camera())
            .is_err()
    );
    assert!(!session.body_visible("aa"));
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
    let prepared = session
        .prepare_part_view_activation(6, camera(), &available)
        .unwrap();
    session.body_visibility.set("aa", true).unwrap();
    assert!(
        session
            .commit_part_view_activation(prepared, camera())
            .is_err()
    );
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
}
#[test]
fn unresolved_body_keys_do_not_change_view_or_display() {
    let mut session = view_session(false, false);
    let available = BTreeSet::from(["aa".into()]);
    assert!(
        session
            .set_part_view_body_visibility(&["bb".into()], &available, false)
            .is_err()
    );
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
    session
        .set_part_view_body_visibility(&["aa".into()], &available, false)
        .unwrap();
    assert!(
        session
            .prepare_part_view_activation(6, camera(), &BTreeSet::new())
            .is_err()
    );
    assert!(!session.body_visible("aa"));
}

#[test]
fn vetoed_unlocked_visibility_edit_restores_the_live_body_set() {
    let mut session = view_session(false, true);
    let available = BTreeSet::from(["aa".into()]);
    assert!(
        session
            .set_part_view_body_visibility(&["aa".into()], &available, false)
            .is_err()
    );
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
}

#[test]
fn bulk_visibility_in_an_empty_part_does_not_change_the_document() {
    let mut session = view_session(false, false);
    let generation = session.generation();
    let empty = BTreeSet::new();
    assert!(
        !session
            .set_part_view_body_visibility(&[], &empty, false)
            .unwrap()
    );
    assert!(
        !session
            .set_part_view_body_visibility(&[], &empty, true)
            .unwrap()
    );
    assert_eq!(session.generation(), generation);
}

#[test]
fn open_transaction_refuses_view_display_changes() {
    let mut session = view_session(false, false);
    session.begin("Pending edit").unwrap();
    let available = BTreeSet::from(["aa".into()]);
    assert!(
        session
            .prepare_part_view_activation(6, camera(), &available)
            .is_err()
    );
    assert!(
        session
            .set_part_view_body_visibility(&["aa".into()], &available, false)
            .is_err()
    );
    assert!(session.body_visible("aa"));
}

#[test]
fn switch_prepared_by_another_session_is_refused() {
    let first = view_session(false, false);
    let mut second = view_session(false, false);
    let prepared = first
        .prepare_part_view_activation(6, camera(), &BTreeSet::new())
        .unwrap();
    assert!(
        second
            .commit_part_view_activation(prepared, camera())
            .is_err()
    );
    assert_eq!(
        second
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .id(),
        1
    );
}

#[path = "part_view_navigation_tests.rs"]
mod navigation;
