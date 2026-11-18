// SPDX-License-Identifier: GPL-2.0-only
//! Typed Hole API against the real desktop replay owner (#4093).
use super::*;
use oblikovati_api::wire::{MassPropertiesArgs, MeasureArgs, featureargs::Hole};
use oblikovati_model::identity::SemanticReferenceName;

fn face_key(owner: &TestOwner, document: u64) -> String {
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let key = SemanticReferenceName::new(guid, 1, 1, "sketch-prism", "end", &[1])
        .unwrap()
        .encode()
        .unwrap();
    key.iter().map(|byte| format!("{byte:02x}")).collect()
}
fn face_area(owner: &mut TestOwner, face: &str) -> f64 {
    let result = Client::new(owner.document_host())
        .analysis()
        .measure(&MeasureArgs {
            body_index: 0,
            type_: "area".into(),
            key_a: face.into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(result.unit, "mm²");
    result.value
}
fn expect_area(owner: &mut TestOwner, face: &str, diameter_mm: f64) {
    let expected = 1200. - std::f64::consts::PI * (diameter_mm / 2.).powi(2);
    let area = face_area(owner, face);
    assert!(
        (area - expected).abs() < 1e-5,
        "face area {area}: expected {expected}"
    );
}
fn add_diameter(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "hole_diameter".into(),
            expression: "2 mm".into(),
            ..Default::default()
        })
        .unwrap();
}
fn hole(face: &str) -> Hole {
    Hole {
        face_ref: face.into(),
        diameter: "hole_diameter".into(),
        ..Default::default()
    }
}

#[test]
fn typed_hole_recomputes_undoes_redoes_and_reopens_its_named_face() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    add_diameter(&mut owner);
    let face = face_key(&owner, document);
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&hole(&face))
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "hole");
    expect_area(&mut owner, &face, 2.);
    set_parameter(&mut owner, "hole_diameter", "3 mm");
    expect_area(&mut owner, &face, 3.);
    owner.files.commands(document).unwrap().undo().unwrap();
    expect_area(&mut owner, &face, 2.);
    owner.files.commands(document).unwrap().redo().unwrap();
    expect_area(&mut owner, &face, 3.);
    super::cutoff_checkpoint::save_close_reopen(&mut owner, document);
    expect_area(&mut owner, &face, 3.);
}

#[test]
fn rejected_hole_requests_leave_document_generation_and_geometry_unchanged() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    add_diameter(&mut owner);
    let face = face_key(&owner, document);
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let mut options = hole(&face);
    options.type_ = "counterbore".into();
    let mut outside = hole(&face);
    outside.center = Some(vec![99., 99., 2.]);
    let mut diameter = hole(&face);
    diameter.diameter = "0 mm".into();
    let mut bad_key = hole(&face);
    bad_key.face_ref = "not-hex".into();
    for input in [options, outside, diameter, bad_key] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .add_feature(&input)
                .is_err(),
            "unexpectedly accepted {input:?}"
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
    assert!((face_area(&mut owner, &face) - 1200.).abs() < 1e-5);
}

#[test]
fn typed_blind_hole_removes_only_the_requested_depth() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    add_diameter(&mut owner);
    let face = face_key(&owner, document);
    let mut input = hole(&face);
    input.termination = "distance".into();
    input.depth = "4 mm".into();
    Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    let volume = Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs::default())
        .unwrap()
        .volume_mm3;
    let expected = 24000. - 4. * std::f64::consts::PI;
    assert!(
        (volume - expected).abs() < 1e-5,
        "volume {volume}: expected {expected}"
    );
}

#[test]
fn selected_face_hole_preview_cancel_and_commit_use_current_generation() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    owner.ambient_selection.disable_all();
    owner
        .ambient_selection
        .set_enabled(oblikovati_session::selection::SelectionKind::Face, true);
    owner
        .click_model_with_window(document, [[2., 1.5, 10.], [0., 0., -1.]], Some(0.01))
        .unwrap();
    let face = owner.chosen_model_face(document).expect("selected face");
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let request = oblikovati_gui::GuiHoleRequest {
        document,
        generation,
        face: face.key.iter().map(|byte| format!("{byte:02x}")).collect(),
        diameter: "2 mm".into(),
        depth: String::new(),
    };
    let mut stale = request.clone();
    stale.generation += 1;
    assert!(crate::HeadWork::preview_hole_from_gui(&mut owner, Some(&stale)).is_err());
    assert!(crate::HeadWork::commit_hole_from_gui(&mut owner, &stale).is_err());
    assert!(crate::HeadWork::preview_hole_from_gui(&mut owner, Some(&request)).unwrap());
    assert!(owner.extrusion_preview.is_some());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(crate::HeadWork::preview_hole_from_gui(&mut owner, None).unwrap());
    assert!(owner.extrusion_preview.is_none());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(crate::HeadWork::commit_hole_from_gui(&mut owner, &request).unwrap());
    expect_area(&mut owner, &request.face, 2.);
}
