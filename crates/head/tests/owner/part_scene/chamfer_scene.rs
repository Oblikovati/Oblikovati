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
