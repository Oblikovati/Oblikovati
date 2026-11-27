// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::GuiExtrudeDirection::Positive;
#[test]
fn desktop_resolution_supports_preview_commit_and_parameter_replay() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: None,
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        additional_profiles: Vec::new(),
        distance: "width".into(),
        second_distance: None,
        operation: None,
        reference_extent: None,
        direction: Default::default(),
    };
    owner.extrusion_preview_from_gui(Some(&preview)).unwrap();
    assert!(
        !owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
    owner
        .extrude_from_gui(
            document,
            crate::scene_fixture::SKETCH,
            0,
            "width",
            Positive,
            None,
        )
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    set_size(&mut owner, "20 mm");
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 2.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 2.);
    let keys = cube_keys(&owner, document);
    let path = document_fixture_path!("/memory/default-policy-cube.opd");
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, reopened, 2.);
    assert_eq!(cube_keys(&owner, reopened), keys);
}

#[test]
fn stale_extrusion_request_does_not_replace_the_current_preview() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let mut preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        additional_profiles: Vec::new(),
        distance: "width".into(),
        second_distance: None,
        operation: None,
        reference_extent: None,
        direction: Default::default(),
    };
    set_size(&mut owner, "20 mm");
    preview.generation = Some(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
    );
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    preview.generation = Some(generation);
    assert!(!owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
}

#[test]
fn api_rectangle_and_profiles_share_certified_extrusion_indices() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let client = Client::new(owner.document_host());
    let sketch_index = client
        .sketch()
        .create(&CreateSketchArgs::default())
        .unwrap()
        .sketch_index;
    let args = SketchArgs { sketch_index };
    assert_eq!(
        client.sketch().profiles(&args).unwrap().profiles,
        Some(Vec::new())
    );
    let rectangle = SketchRectangleArgs {
        sketch_index,
        width: "width * 3".into(),
        height: "20 mm".into(),
    };
    let reply = client.sketch().rectangle(&rectangle).unwrap();
    assert_eq!(reply.sketch_index, sketch_index);
    assert_eq!(reply.profiles, 1);
    for (center, radius) in [([1.0, 1.0], "5 mm"), ([8.0, 8.0], "1 cm")] {
        client
            .sketch()
            .add_entity(&AddSketchEntityArgs {
                sketch_index,
                kind: "circle".into(),
                points: Some(vec![Some(center.to_vec())]),
                radius: radius.into(),
                ..Default::default()
            })
            .unwrap();
    }
    let profiles = client.sketch().profiles(&args).unwrap().profiles.unwrap();
    assert_eq!(profiles.len(), 3, "{profiles:?}");
    for (profile, index, holes, area) in [
        (&profiles[0], 0, 1, 6.0 - std::f64::consts::PI / 4.0),
        (&profiles[1], 1, 0, std::f64::consts::PI / 4.0),
        (&profiles[2], 2, 0, std::f64::consts::PI),
    ] {
        assert_eq!(profile.index, index);
        assert_eq!(profile.holes, holes);
        assert!(profile.closed);
        assert!((profile.area - area).abs() < 1e-8);
    }
    assert!(client.sketch().rectangle(&rectangle).is_err());
    assert_eq!(
        client.sketch().profiles(&args).unwrap().profiles.unwrap(),
        profiles
    );
    let reply = client
        .features()
        .add_feature(&Extrude {
            sketch_index,
            profile_index: profiles[2].index,
            ..request()
        })
        .unwrap()
        .0
        .unwrap();
    assert_eq!(reply["bodies"], 1);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert!(owner.documents().workspace().session(document).is_ok());
}
