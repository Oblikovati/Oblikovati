// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::CreateWorkPlaneArgs;

#[test]
fn to_face_tracks_the_named_face_through_edit_history_and_reopen() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let source = add_distance_source(&mut owner);
    let face = distal_face_key(&mut owner);
    let bounded = add_plane_extrusion(&mut owner, "to-face", "", &face);
    assert_reference_reads(&owner, document, bounded, source);
    assert_extent_body(&mut owner, 1, 1200., 6.);
    let client = Client::new(owner.document_host());
    client
        .features()
        .edit(&distance_edit(source, "16 mm"))
        .unwrap();
    drop(client);
    assert_extent_body(&mut owner, 1, 1600., 8.);
    verify_extent_history(&mut owner, document, bounded);
}

fn assert_reference_reads(owner: &TestOwner, document: u64, bounded: u64, source: u64) {
    use oblikovati_model::feature::{FeatureId, FeatureReadKey};
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let feature = snapshot.work_geometry.get(FeatureId(bounded)).unwrap();
    assert!(feature.owned_scalars().is_empty());
    assert!(
        feature
            .reads()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(source)))
    );
}

fn verify_extent_history(owner: &mut TestOwner, document: u64, bounded: u64) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_extent_body(owner, 1, 1200., 6.);
    let client = Client::new(owner.document_host());
    client.transactions().redo().unwrap();
    let result = client.features().edit(&distance_edit(bounded, "19 mm"));
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("no authored distance")
    );
    drop(client);
    reopen_extent_document(owner, document);
    assert_extent_body(owner, 1, 1600., 8.);
}

#[test]
fn from_to_uses_two_resolved_planes_and_refuses_invalid_bounds_atomically() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    add_distance_source(&mut owner);
    let to = distal_face_key(&mut owner);
    let from = make_start_plane(&mut owner);
    assert_invalid_bounds(&mut owner, document, &to);
    add_plane_extrusion(&mut owner, "from-to", &to, &from);
    assert_extent_body(&mut owner, 1, 800., 8.);
    reopen_extent_document(&mut owner, document);
    assert_extent_body(&mut owner, 1, 800., 8.);
}

fn assert_invalid_bounds(owner: &mut TestOwner, document: u64, to: &str) {
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for (from, target, expected) in [
        ("", to, "fromFace"),
        ("origin/plane/yz", to, "parallel to the profile"),
        (to, to, "distinct finite bounds"),
    ] {
        let result = Client::new(owner.document_host())
            .features()
            .add_feature(&plane_extrusion("from-to", from, target));
        assert!(
            result.unwrap_err().to_string().contains(expected),
            "{expected}"
        );
        let session = owner.documents().workspace().session(document).unwrap();
        assert_eq!(session.generation(), generation);
        assert_eq!(session.snapshot().work_geometry.features().len(), 2);
    }
}

fn make_start_plane(owner: &mut TestOwner) -> String {
    let plane = Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec!["origin/plane/xy".into()]),
            offset: "4 mm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(plane.healthy);
    format!("plane/{}", plane.index)
}

fn add_distance_source(owner: &mut TestOwner) -> u64 {
    let mut request = request();
    request.distance = "12 mm".into();
    add_extent_test_input(owner, &request)
}

fn distal_face_key(owner: &mut TestOwner) -> String {
    Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()
        .remove(0)
        .faces
        .unwrap()
        .into_iter()
        .find(|face| {
            face.kind == "plane"
                && face
                    .point
                    .as_ref()
                    .is_some_and(|p| (p[2] - 1.2).abs() < 1e-9)
        })
        .expect("distal plane of the forward extrusion")
        .key
}

fn add_plane_extrusion(owner: &mut TestOwner, extent: &str, from: &str, to: &str) -> u64 {
    add_extent_test_input(owner, &plane_extrusion(extent, from, to))
}

fn add_extent_test_input(owner: &mut TestOwner, input: &Extrude) -> u64 {
    Client::new(owner.document_host())
        .features()
        .add_feature(input)
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap()
}

fn plane_extrusion(extent: &str, from: &str, to: &str) -> Extrude {
    Extrude {
        operation: "new".into(),
        extent: extent.into(),
        from_face: from.into(),
        to_face: to.into(),
        ..Default::default()
    }
}

fn distance_edit(id: u64, value: &str) -> EditFeatureArgs {
    EditFeatureArgs {
        id,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: value.into(),
        }]),
        ..Default::default()
    }
}

fn assert_extent_body(owner: &mut TestOwner, index: i64, volume_mm3: f64, centroid_z_mm: f64) {
    let body = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: index,
            density_g_cm3: 1.,
            ..Default::default()
        })
        .unwrap();
    assert!((body.volume_mm3 - volume_mm3).abs() < 1e-6, "{body:?}");
    assert!(
        (body.centroid_z_mm - centroid_z_mm).abs() < 1e-8,
        "{body:?}"
    );
}

fn reopen_extent_document(owner: &mut TestOwner, document: u64) {
    let client = Client::new(owner.document_host());
    let path = "/memory/extrusion-reference-extent.opd";
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
}
