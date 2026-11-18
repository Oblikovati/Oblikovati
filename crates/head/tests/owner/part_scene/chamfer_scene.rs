// SPDX-License-Identifier: GPL-2.0-only
//! Typed named-edge Chamfer through the desktop owner (#4046).
use super::*;
use oblikovati_api::wire::featureargs::Chamfer;
use oblikovati_model::identity::SemanticReferenceName;

#[test]
fn typed_chamfer_rejects_invalid_inputs_and_commits_an_undoable_solid() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
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
    let input = Chamfer {
        edge_refs: Some(vec![
            edge.iter().map(|byte| format!("{byte:02x}")).collect(),
        ]),
        distance: "1 mm".into(),
        ..Default::default()
    };
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for invalid in [
        Chamfer {
            chamfer_type: "twoDistances".into(),
            ..input.clone()
        },
        Chamfer {
            partial_length: "1 mm".into(),
            ..input.clone()
        },
        Chamfer {
            distance: "0 mm".into(),
            ..input.clone()
        },
        Chamfer {
            distance: "1000 mm".into(),
            ..input.clone()
        },
        Chamfer {
            edge_refs: Some(vec!["invalid".into()]),
            ..input.clone()
        },
    ] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .add_feature(&invalid)
                .is_err()
        );
    }
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
    );
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "chamfer");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .features()
            .len(),
        1
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .features()
            .len(),
        2
    );
}

#[test]
fn typed_chamfer_distance_edit_validates_and_replays() {
    use oblikovati_api::wire::{EditFeatureArgs, FeatureRefArgs, ScalarEdit};
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
    let input = Chamfer {
        edge_refs: Some(vec![
            edge.iter().map(|byte| format!("{byte:02x}")).collect(),
        ]),
        distance: "1 mm".into(),
        ..Default::default()
    };
    let id = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap();
    let edit = |value: &str| EditFeatureArgs {
        id,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: value.into(),
        }]),
        ..Default::default()
    };
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let original: Vec<_> = owner
        .take_scene()
        .unwrap()
        .unwrap()
        .scene
        .triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect();
    let draft = oblikovati_gui::GuiFilletEditRequest {
        document,
        generation: before,
        feature: oblikovati_model::feature::FeatureId(id),
        radius: "2 mm".into(),
    };
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .browser
            .unwrap()
            .chamfer_edits
            .contains_key(&draft.feature)
    );
    let before_metrics = owner.part_scenes.checkpoint_preview_metrics();
    assert!(owner.preview_chamfer_edit(&draft).unwrap());
    let first_metrics = owner.part_scenes.checkpoint_preview_metrics();
    assert_eq!(first_metrics, (before_metrics.0 + 1, before_metrics.1 + 1));
    assert!(owner.preview_chamfer_edit(&draft).unwrap());
    assert_eq!(
        owner.part_scenes.checkpoint_preview_metrics(),
        (first_metrics.0, first_metrics.1 + 1)
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
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
    assert_ne!(original, preview);
    assert!(owner.preview_chamfer_from_gui(None).unwrap());
    for value in ["0 mm", "1000 mm", "1 deg"] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .edit(&edit(value))
                .is_err()
        );
    }
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
    );
    let before_commit = owner.part_scenes.checkpoint_preview_metrics();
    let backend_before = owner.fillet_access.as_ref().unwrap().replay_counts();
    assert!(owner.commit_chamfer_edit(&draft).unwrap());
    assert_eq!(
        owner.part_scenes.checkpoint_preview_metrics(),
        before_commit
    );
    assert_eq!(
        owner.fillet_access.as_ref().unwrap().replay_counts(),
        backend_before
    );
    assert!(owner.commit_chamfer_edit(&draft).is_err());
    let scalar = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()
        .remove(0);
    assert_eq!(
        (scalar.label.as_str(), scalar.unit.as_str(), scalar.value),
        ("Distance", "mm", 2.)
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    let scalar = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()
        .remove(0);
    assert_eq!(scalar.value, 1.);
    owner.files.commands(document).unwrap().redo().unwrap();
    let scalar = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()
        .remove(0);
    assert_eq!(scalar.value, 2.);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "bevel_size".into(),
            expression: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(owner.document_host())
        .features()
        .edit(&edit("bevel_size"))
        .unwrap();
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "bevel_size".into(),
            expression: "2 mm".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/chamfer-edit.opd".into(),
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
            full_document_name: "/memory/chamfer-edit.opd".into(),
            ..Default::default()
        })
        .unwrap();
    let scalar = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()
        .remove(0);
    assert_eq!(scalar.value, 2.);
}
