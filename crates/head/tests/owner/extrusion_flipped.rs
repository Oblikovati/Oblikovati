// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
    let path = "/memory/flipped-extrusion.opd";
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
