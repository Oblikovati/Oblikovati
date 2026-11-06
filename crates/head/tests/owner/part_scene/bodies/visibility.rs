// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn set_visible(owner: &mut TestOwner, index: i64, visible: bool) -> BodyInfo {
    Client::new(owner.document_host())
        .body()
        .set_visible(&BodySetVisibleArgs {
            body_index: index,
            visible,
        })
        .unwrap()
        .body
}
#[test]
fn body_visibility_replaces_the_scene_without_changing_model_or_dirty_state() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let first = owner.take_scene().unwrap().unwrap();
    let mut expected = listed(&mut owner)[0].clone();
    expected.visible = false;
    assert_eq!(set_visible(&mut owner, 0, false), expected);
    assert_eq!(listed(&mut owner), [expected]);
    let hidden = owner.take_scene().unwrap().unwrap();
    assert_eq!(hidden.status.source, first.status.source);
    assert_eq!(hidden.status.visible_bodies, 0);
    assert!(hidden.scene.triangles.is_empty());
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    assert!(set_visible(&mut owner, 0, true).visible);
    let shown = owner.take_scene().unwrap().unwrap();
    assert_scene_triangles(&shown.scene, &first.scene);
    set_visible(&mut owner, 0, true);
    assert!(owner.take_scene().unwrap().is_none());
}
fn assert_scene_triangles(
    actual: &oblikovati_viewport::ViewportScene,
    expected: &oblikovati_viewport::ViewportScene,
) {
    assert_eq!(actual.background, expected.background);
    assert_eq!(actual.triangles.len(), expected.triangles.len());
    for (actual, expected) in actual.triangles.iter().zip(&expected.triangles) {
        assert_eq!(actual.positions, expected.positions);
        assert_eq!(actual.color, expected.color);
    }
}
#[test]
fn visibility_follows_keys_across_recompute_failed_features_and_undo() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let hidden = set_visible(&mut owner, 1, false);
    set_parameter(&mut owner, "depth", "0 cm");
    let remaining = listed(&mut owner);
    assert_eq!(remaining.len(), 1);
    assert_eq!(remaining[0].key, hidden.key);
    assert!(!remaining[0].visible);
    let scene = owner.take_scene().unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 0);
    assert_eq!(scene.status.health.len(), 2);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!(!listed(&mut owner)[1].visible);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
#[test]
fn visibility_is_document_local_and_dropped_after_save_close_reopen() {
    let (mut owner, _) = owner();
    let first = open_prism(&mut owner, false);
    let hidden = set_visible(&mut owner, 0, false);
    let second = open_other_part(&mut owner);
    assert!(listed(&mut owner)[0].visible);
    activate_visibility_part(&mut owner, first);
    assert!(!listed(&mut owner)[0].visible);
    rename::save_reopen_body(&mut owner, first);
    assert!(listed(&mut owner)[0].visible);
    assert_eq!(listed(&mut owner)[0].key, hidden.key);
    activate_visibility_part(&mut owner, second);
    assert!(listed(&mut owner)[0].visible);
}
fn activate_visibility_part(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: document })
        .unwrap();
}
fn open_other_part(owner: &mut TestOwner) -> u64 {
    let path = std::env::temp_dir().join("head-visibility-other.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), stored_bytes(false));
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
#[test]
fn rejected_visibility_indices_preserve_pending_scene_and_body_flags() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let original = listed(&mut owner);
    for body_index in [-1, 1, i64::MAX] {
        let error = Client::new(owner.document_host())
            .body()
            .set_visible(&BodySetVisibleArgs {
                body_index,
                visible: false,
            })
            .unwrap_err()
            .to_string();
        assert!(error.contains(&body_index.to_string()));
    }
    assert_eq!(listed(&mut owner), original);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn hidden_bodies_remain_in_analytic_queries_and_visibility_is_not_transactional() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let request = MassPropertiesArgs {
        density_g_cm3: 1.,
        accuracy: String::new(),
    };
    let before = Client::new(owner.document_host())
        .analysis()
        .mass_properties(&request)
        .unwrap();
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Metadata group")
        .unwrap();
    set_visible(&mut owner, 0, false);
    owner.files.commands(document).unwrap().abort().unwrap();
    assert!(!listed(&mut owner)[0].visible);
    let after = Client::new(owner.document_host())
        .analysis()
        .mass_properties(&request)
        .unwrap();
    assert_eq!(after, before);
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
}
