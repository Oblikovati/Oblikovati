// SPDX-License-Identifier: GPL-2.0-only
//! A typed Shell refusal must not change the live document (#4106, Martensite #554).
use super::*;
use oblikovati_api::wire::featureargs::Shell;
use oblikovati_automation::{FileHost, PartReplayService, PartUpdateAccess};

#[test]
fn typed_shell_refusal_preserves_recipe_history_and_dirty_state() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.update_part(&mut owner.files, document).unwrap();
    let face = named_face(&owner, &replay, document);
    let before = document_state(&owner, document);
    assert!(!before.4);
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_part_update(&mut replay);
    let error = Client::new(host)
        .features()
        .add_feature(&Shell {
            face_refs: Some(vec![face]),
            thickness: "1 mm".into(),
            ..Default::default()
        })
        .unwrap_err()
        .to_string();
    assert!(
        error.contains("tolerance"),
        "unexpected Shell refusal: {error}"
    );
    let after = document_state(&owner, document);
    assert_eq!(after.1, before.1, "generation changed");
    assert_eq!(after.2, before.2, "undo history changed");
    assert_eq!(after.3, before.3, "redo history changed");
    assert!(
        before
            .0
            .work_geometry
            .same_work_plane_recipe(&after.0.work_geometry)
    );
    assert!(before.0.sketches.same_recipe(&after.0.sketches));
    assert!(before.0.parameters.same_recipe(&after.0.parameters));
    assert!(before.0.same_metadata_recipe(&after.0));
    assert_eq!(after.4, before.4, "dirty state changed");
}

fn named_face(owner: &TestOwner, replay: &PartReplayService, document: u64) -> String {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let evaluation = replay.current_part(&owner.files, document).unwrap();
    let body = evaluation.body(FeatureId(1)).unwrap();
    let keys = body.face_reference_keys(session.history(), guid).unwrap();
    keys.first()
        .unwrap()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}

fn document_state(
    owner: &TestOwner,
    document: u64,
) -> (
    oblikovati_model::history::DocumentSnapshot,
    u64,
    Vec<String>,
    Vec<String>,
    bool,
) {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let history = session.history();
    (
        session.snapshot().clone(),
        session.generation(),
        history
            .undo_labels()
            .into_iter()
            .map(str::to_owned)
            .collect(),
        history
            .redo_labels()
            .into_iter()
            .map(str::to_owned)
            .collect(),
        workspace.needs_save(document).unwrap(),
    )
}
