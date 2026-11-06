// SPDX-License-Identifier: GPL-2.0-only
use super::three_point::{CardinalCircleConstruction, call};
use super::*;
fn slot_request(variant: &str) -> AddSketchEntityArgs {
    let points = match variant {
        "arc" => vec![[0.0; 2], [4.0, 0.0], [0.0, 4.0]],
        "arcThreePoint" => vec![[4.0, 0.0], [0.0, 4.0], [-4.0, 0.0]],
        _ => vec![[0.0; 2], [4.0, 0.0]],
    };
    AddSketchEntityArgs {
        variant: variant.into(),
        width: "2 cm".into(),
        end_angle: "90 deg".into(),
        ccw: true,
        construction: true,
        ..literal_input("slot", &points)
    }
}
#[test]
fn all_slot_variants_keep_go_reply_ownership_and_survive_undo_reload() {
    for variant in [
        "",
        "unknown",
        "overall",
        "slotCenter",
        "arc",
        "arcCenterPoint",
        "arcThreePoint",
    ] {
        let (mut files, document, mut source) = fresh();
        let backend = CardinalCircleConstruction::default();
        let reply = call(&mut files, &mut source, &slot_request(variant), &backend).unwrap();
        let curved = variant.starts_with("arc");
        let first = if curved { 10 } else { 8 };
        assert_eq!(reply.entity_id, first);
        assert_eq!(reply.entity_i_ds, Some((first..first + 4).collect()));
        assert_eq!(
            reply.point_i_ds,
            if curved { None } else { Some(vec![2, 3, 4, 5]) }
        );
        let created = rows(&files);
        assert_eq!(created.len(), 5);
        assert!(created[1..].iter().all(|row| row.construction));
        verify_keys(&files, &created);
        files.commands(document).unwrap().undo().unwrap();
        assert_eq!(rows(&files).len(), 1);
        files.commands(document).unwrap().redo().unwrap();
        assert_eq!(rows(&files), created);
        reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_eq!(rows(&files), created);
    }
}
#[test]
fn three_point_slot_verification_refusal_discards_the_complete_composite() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    let mut backend = CardinalCircleConstruction::default();
    backend.fail_verification = true;
    expect_api_decline(
        call(
            &mut files,
            &mut source,
            &slot_request("arcThreePoint"),
            &backend,
        ),
        "final circular seed rejected",
    );
    assert_eq!(rows(&files), before);
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_next_point(&mut files, &mut source, 14);
}
#[test]
fn malformed_slots_and_missing_kernel_decline_before_reservation() {
    let (mut files, document, mut source) = fresh();
    assert_refused(
        &mut files,
        document,
        &mut source,
        &slot_request("arcThreePoint"),
        "injected",
    );
    for variant in ["", "arc", "arcCenterPoint"] {
        let mut input = slot_request(variant);
        input.points.as_mut().unwrap().pop();
        assert_refused(&mut files, document, &mut source, &input, "exactly");
    }
    let mut input = slot_request("");
    input.width = "-2 cm".into();
    assert_refused(&mut files, document, &mut source, &input, "positive finite");
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn slot_expressions_use_working_units_and_preserve_physical_radii() {
    for variant in ["arc", "arcThreePoint"] {
        let (mut files, _, mut source) = configured_scaled(0.001);
        create_empty_sketch(&mut files, &mut source);
        let backend = CardinalCircleConstruction::default();
        let mut input = slot_request(variant);
        let coordinates = if variant == "arc" {
            [["0 cm", "0 cm"], ["40 mm", "0 cm"], ["0 cm", "40 mm"]]
        } else {
            [["40 mm", "0 cm"], ["0 cm", "40 mm"], ["-40 mm", "0 cm"]]
        };
        input.point_exprs = Some(
            coordinates
                .map(|pair| Some(pair.map(String::from).to_vec()))
                .to_vec(),
        );
        input.points = Some(vec![None]);
        let reply = call(&mut files, &mut source, &input, &backend).unwrap();
        for (id, radius) in reply
            .entity_i_ds
            .unwrap()
            .into_iter()
            .zip([5.0, 1.0, 3.0, 1.0])
        {
            assert!((row(&files, id).radius - radius).abs() < 1e-9);
        }
    }
}
#[test]
fn invalid_slot_quantity_expressions_keep_the_next_identity_available() {
    let (mut files, document, mut source) = fresh();
    let mut input = slot_request("");
    input.width = "2 deg".into();
    assert_refused(&mut files, document, &mut source, &input, "length");
    input = slot_request("arcCenterPoint");
    input.end_angle = "missing_sweep".into();
    assert!(add_call(&mut files, &mut source, &input).is_err());
    assert_next_point(&mut files, &mut source, 2);
}
