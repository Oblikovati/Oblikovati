// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::json::EmptyObject;
fn listed(owner: &mut TestOwner) -> Vec<BodyInfo> {
    Client::new(owner.document_host())
        .body()
        .list(&EmptyObject {})
        .unwrap()
        .bodies
        .unwrap_or_default()
}
#[test]
fn body_query_reports_exact_topology_without_consuming_scene_or_editing_history() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let bodies = listed(&mut owner);
    assert_eq!(bodies.len(), 2);
    for (index, body) in bodies.iter().enumerate() {
        assert_eq!(body.index, i64::try_from(index).unwrap());
        assert_eq!(body.name, format!("Solid{}", index + 1));
        assert!(body.solid && body.visible);
        assert_eq!(
            (
                body.faces,
                body.edges,
                body.vertices,
                body.shells,
                body.wires
            ),
            (6, 12, 8, 1, 6)
        );
        assert!(!body.key.is_empty());
        assert!(body.material_id.is_empty() && body.style.is_empty());
    }
    assert_ne!(bodies[0].key, bodies[1].key);
    assert_eq!(bodies, listed(&mut owner));
    assert_eq!(
        generation,
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation()
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        2
    );
    assert!(owner.take_scene().unwrap().is_none());
}
#[test]
fn listed_keys_accept_assignments_and_survive_edits_undo_and_failure_of_another_body() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let original = listed(&mut owner);
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: original[1].key.clone(),
            material_id: "steel".into(),
        })
        .unwrap();
    assert_eq!(listed(&mut owner)[1].material_id, "steel");
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!(listed(&mut owner)[1].material_id.is_empty());
    owner.files.commands(document).unwrap().redo().unwrap();
    set_parameter(&mut owner, "width", "6 cm");
    assert_eq!(listed(&mut owner)[1].key, original[1].key);
    set_parameter(&mut owner, "depth", "0 cm");
    let current = listed(&mut owner);
    assert_eq!(current.len(), 1);
    assert_eq!(current[0].key, original[1].key);
    assert_eq!(current[0].index, 0);
    assert_eq!(current[0].material_id, "steel");
    assert_eq!(owner.take_scene().unwrap().unwrap().status.health.len(), 2);
}
#[test]
fn body_query_rejects_missing_document_and_missing_native_owner() {
    let (mut owner, _) = owner();
    assert!(
        Client::new(owner.document_host())
            .body()
            .list(&EmptyObject {})
            .is_err()
    );
    let document = open_prism(&mut owner, false);
    assert!(
        owner
            .files
            .body_display_metadata(document + 1, "absent")
            .is_err()
    );
    let mut guids = OwnerThreadGuids::default();
    let host = oblikovati_automation::FileHost::new(&mut owner.files, &mut guids, "test");
    assert!(
        Client::new(host)
            .body()
            .list(&EmptyObject {})
            .unwrap_err()
            .to_string()
            .contains("injected exact topology owner")
    );
}
#[test]
fn saved_names_and_legacy_styles_are_returned_after_reopen() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let key = listed(&mut owner)[0].key.clone();
    Client::new(owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document })
        .unwrap();
    let path = owner.files.path(document).unwrap().to_path_buf();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    let mut bytes = owner
        .documents()
        .files()
        .saved
        .borrow()
        .get(&path)
        .unwrap()
        .clone();
    bytes.extend_from_slice(
        format!("\nbodyNames:\n  {key}: Bracket\nbodyColorStyles:\n  {key}: Blue\n").as_bytes(),
    );
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), bytes);
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap();
    let bodies = listed(&mut owner);
    assert_eq!(bodies[0].key, key);
    assert_eq!(bodies[0].name, "Bracket");
    assert_eq!(bodies[0].style, "Blue");
}
#[test]
fn empty_part_lists_no_bodies_and_part_material_is_inherited() {
    let (mut owner, awake) = owner();
    let empty = create(&mut owner, &awake);
    assert!(listed(&mut owner).is_empty());
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: empty,
            force: true,
        })
        .unwrap();
    open_prism(&mut owner, true);
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: "steel".into(),
        })
        .unwrap();
    assert!(
        listed(&mut owner)
            .iter()
            .all(|body| body.material_id == "steel")
    );
}

#[path = "bodies/rename.rs"]
mod rename;

#[path = "bodies/visibility.rs"]
mod visibility;
