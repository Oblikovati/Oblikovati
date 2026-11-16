// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{EditFeatureArgs, FeatureRefArgs, ScalarEdit, featureargs::Fillet};
use oblikovati_model::identity::SemanticReferenceName;

fn edit(id: u64, value: &str) -> EditFeatureArgs {
    EditFeatureArgs {
        id,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: value.into(),
        }]),
        ..Default::default()
    }
}
fn radius(owner: &mut TestOwner, id: u64) -> f64 {
    let scalar = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()
        .remove(0);
    assert_eq!(scalar.label, "Radius");
    assert_eq!(scalar.unit, "mm");
    scalar.value
}
#[test]
fn typed_radius_edit_validates_before_commit_and_survives_history() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    let request = Fillet {
        edge_refs: Some(vec![edge.iter().map(|b| format!("{b:02x}")).collect()]),
        radius: "1 mm".into(),
        ..Default::default()
    };
    let id = Client::new(owner.document_host())
        .features()
        .add_feature(&request)
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap();
    let browser = owner.gui_command_state().unwrap().browser.unwrap();
    assert_eq!(
        browser
            .features
            .iter()
            .find(|feature| feature.id.0 == id)
            .unwrap()
            .kind,
        "Fillet"
    );
    let before_scene: Vec<_> = owner
        .take_scene()
        .unwrap()
        .unwrap()
        .scene
        .triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect();
    assert_eq!(radius(&mut owner, id), 1.);
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let draft = oblikovati_gui::GuiFilletEditRequest {
        document,
        generation,
        feature: oblikovati_model::feature::FeatureId(id),
        radius: "2 mm".into(),
    };
    assert!(owner.preview_fillet_edit(&draft).unwrap());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    let preview: Vec<_> = owner
        .take_scene()
        .unwrap()
        .unwrap()
        .scene
        .triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect();
    assert_ne!(before_scene, preview);
    assert!(owner.preview_fillet_from_gui(None).unwrap());
    assert_eq!(radius(&mut owner, id), 1.);
    assert!(owner.commit_fillet_edit(&draft).unwrap());
    assert_eq!(radius(&mut owner, id), 2.);
    assert!(owner.commit_fillet_edit(&draft).is_err());
    let after_scene: Vec<_> = owner
        .take_scene()
        .unwrap()
        .unwrap()
        .scene
        .triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect();
    assert_ne!(before_scene, after_scene);
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for invalid in ["0 mm", "100 mm", "1 deg"] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .edit(&edit(id, invalid))
                .is_err()
        );
        assert_eq!(
            owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .generation(),
            generation
        );
        assert_eq!(radius(&mut owner, id), 2.);
    }
    let mut invalid_slot = edit(id, "1 mm");
    invalid_slot.scalars.as_mut().unwrap()[0].index = 1;
    assert!(
        Client::new(owner.document_host())
            .features()
            .edit(&invalid_slot)
            .is_err()
    );
    let mut duplicate = edit(id, "1 mm");
    duplicate.scalars.as_mut().unwrap().push(ScalarEdit {
        index: 0,
        value: "2 mm".into(),
    });
    assert!(
        Client::new(owner.document_host())
            .features()
            .edit(&duplicate)
            .is_err()
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(radius(&mut owner, id), 1.);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(radius(&mut owner, id), 2.);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "round_radius".into(),
            expression: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(owner.document_host())
        .features()
        .edit(&edit(id, "round_radius"))
        .unwrap();
    assert_eq!(radius(&mut owner, id), 1.);
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "round_radius".into(),
            expression: "2 mm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(radius(&mut owner, id), 2.);
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/fillet-edit.opd".into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/memory/fillet-edit.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(radius(&mut owner, id), 2.);
}
