// SPDX-License-Identifier: GPL-2.0-only
//! Saved model-state identities must remain valid during blend candidate replay (#4080).
use super::*;
use oblikovati_model::identity::SemanticReferenceName;

fn edited_prism() -> (TestOwner, u64) {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    set_parameter(&mut owner, "width", "50 mm");
    for index in 0..4 {
        Client::new(owner.document_host())
            .model_states()
            .create(&CreateModelStateArgs {
                name: format!("Variant {index}"),
                ..Default::default()
            })
            .unwrap();
    }
    let reopened = save_and_reopen(&mut owner, document);
    assert_later_model_state(&owner, reopened);
    (owner, reopened)
}

fn save_and_reopen(owner: &mut TestOwner, document: u64) -> u64 {
    let path = "/memory/blend-model-states.opd";
    save_part(owner, document, path);
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

fn save_part(owner: &mut TestOwner, document: u64, path: &str) {
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
}

fn assert_later_model_state(owner: &TestOwner, document: u64) {
    let session = owner.files.workspace().session(document).unwrap();
    let snapshot = session.snapshot();
    let last = snapshot
        .model_states
        .as_ref()
        .unwrap()
        .list()
        .iter()
        .map(|state| state.id)
        .max()
        .unwrap();
    assert!(snapshot.parameters.ids().iter().all(|id| *id < last));
}

fn prism_edge(owner: &TestOwner, document: u64) -> String {
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
        .unwrap()
        .encode()
        .unwrap()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}

#[test]
fn reopened_edited_prism_accepts_fillet_after_model_states() {
    let (mut owner, document) = edited_prism();
    let input = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: Some(vec![prism_edge(&owner, document)]),
        radius: "1 mm".into(),
        ..Default::default()
    };
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "fillet");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[test]
fn reopened_edited_prism_accepts_chamfer_after_model_states() {
    let (mut owner, document) = edited_prism();
    let input = oblikovati_api::wire::featureargs::Chamfer {
        edge_refs: Some(vec![prism_edge(&owner, document)]),
        distance: "1 mm".into(),
        ..Default::default()
    };
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "chamfer");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
