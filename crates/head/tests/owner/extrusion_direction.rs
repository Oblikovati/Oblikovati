// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn asymmetric_extrusion_keeps_each_side_independently_driven() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.distance = "8 mm".into();
    input.second_distance = "4 mm".into();
    Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_asymmetric_body(&mut owner, 1_200., 2.);

    let invalid = EditFeatureArgs {
        id: 1,
        scalars: Some(vec![
            ScalarEdit {
                index: 0,
                value: "9 mm".into(),
            },
            ScalarEdit {
                index: 1,
                value: "0 mm".into(),
            },
        ]),
        ..Default::default()
    };
    assert!(
        Client::new(owner.document_host())
            .features()
            .edit(&invalid)
            .unwrap_err()
            .to_string()
            .contains("Distance B")
    );
    assert_asymmetric_body(&mut owner, 1_200., 2.);

    let client = Client::new(owner.document_host());
    client
        .features()
        .edit(&EditFeatureArgs {
            id: 1,
            scalars: Some(vec![ScalarEdit {
                index: 1,
                value: "6 mm".into(),
            }]),
            ..Default::default()
        })
        .unwrap();
    drop(client);
    assert_asymmetric_body(&mut owner, 1_400., 1.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_asymmetric_body(&mut owner, 1_200., 2.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_asymmetric_body(&mut owner, 1_400., 1.);
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(
        session.snapshot().work_geometry.features()[0]
            .owned_scalars()
            .len(),
        2
    );
    let client = Client::new(owner.document_host());
    let path = document_fixture_path!("/memory/asymmetric-extrusion.opd");
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    drop(client);
    assert_asymmetric_body(&mut owner, 1_400., 1.);
}

fn assert_asymmetric_body(owner: &mut TestOwner, volume_mm3: f64, centroid_z_mm: f64) {
    let body = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 0,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    assert!(
        (body.centroid_z_mm - centroid_z_mm).abs() < 1e-8,
        "{body:?}"
    );
    assert!(
        ((body.volume_mm3 - volume_mm3) / 1_000.).abs() < 1e-8,
        "{body:?}"
    );
}

#[test]
fn typed_symmetric_extrusion_remains_centered_when_its_driver_changes() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.direction = " Symmetric ".into();
    Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_centered_volume(&mut owner, 1_000.);
    set_size(&mut owner, "20 mm");
    assert_centered_volume(&mut owner, 8_000.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_centered_volume(&mut owner, 1_000.);
    let client = Client::new(owner.document_host());
    client.transactions().redo().unwrap();
    let path = document_fixture_path!("/memory/symmetric-extrusion.opd");
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    drop(client);
    assert_centered_volume(&mut owner, 8_000.);
}

fn assert_centered_volume(owner: &mut TestOwner, expected: f64) {
    let body = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 0,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    assert!(body.centroid_z_mm.abs() < 1e-8, "{body:?}");
    // Use the existing cube volume assertion's tolerance in model units.
    assert!(
        ((body.volume_mm3 - expected) / 1_000.).abs() < 1e-8,
        "{body:?}"
    );
}

#[test]
fn typed_flipped_extrusion_keeps_positive_distance_through_edit_and_reopen() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.direction = " Negative ".into();
    Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_flipped_depth(&mut owner, 10.);
    set_size(&mut owner, "20 mm");
    assert_flipped_depth(&mut owner, 20.);

    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    drop(client);
    assert_flipped_depth(&mut owner, 10.);
    let client = Client::new(owner.document_host());
    client.transactions().redo().unwrap();
    client
        .features()
        .edit(&EditFeatureArgs {
            id: 1,
            scalars: Some(vec![ScalarEdit {
                index: 0,
                value: "30 mm".into(),
            }]),
            ..Default::default()
        })
        .unwrap();
    let path = document_fixture_path!("/memory/flipped-extrusion.opd");
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    drop(client);
    assert_flipped_depth(&mut owner, 30.);
}

fn assert_flipped_depth(owner: &mut TestOwner, depth_mm: f64) {
    let client = Client::new(owner.document_host());
    let properties = client
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 0,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    assert!((properties.centroid_z_mm + depth_mm / 2.).abs() < 1e-8);
    let extrusion = client.features().get(&FeatureRefArgs { id: 1 }).unwrap();
    assert_eq!(extrusion.feature.scalars.unwrap()[0].value, depth_mm);
}

#[test]
fn desktop_extrude_checks_document_identity_and_uses_the_typed_command() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(
                document + 1,
                crate::scene_fixture::SKETCH,
                0,
                "width",
                Positive,
                None
            )
            .is_err()
    );
    assert!(
        owner
            .extrude_from_gui(
                document,
                oblikovati_model::document_sketch::DocumentSketchId(999),
                0,
                "width",
                Positive,
                None,
            )
            .is_err()
    );
    assert!(
        owner
            .extrude_from_gui(
                document,
                crate::scene_fixture::SKETCH,
                0,
                "width",
                Positive,
                None
            )
            .unwrap()
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, document, 1.);
}
