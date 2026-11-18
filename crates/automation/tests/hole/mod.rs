// SPDX-License-Identifier: GPL-2.0-only
//! Hole API request classification and driving inputs (#4093).
use super::*;
use oblikovati_model::{
    feature::{
        hole::{HoleExtent, HolePosition},
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
};

fn key(kind: u32) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 1, kind, "fixture", "source", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn request() -> Hole {
    Hole {
        face_ref: "00".into(),
        diameter: "2 mm".into(),
        ..Default::default()
    }
}

#[test]
fn straight_hole_inputs_preserve_named_parameters_and_signed_coordinates() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    let mut input = request();
    input.diameter = "width".into();
    input.depth = "2 mm".into();
    input.center_expr = Some(vec!["width".into(), "-2 mm".into(), "0".into()]);
    input.center = Some(vec![9., 9., 9.]);
    input::require_supported(&input).unwrap();
    let definition = prepare(&snapshot, key(4), key(1), &input)
        .unwrap()
        .definition;
    let recipe = definition.recipe();
    assert_eq!(recipe.diameter, DatumLength::Parameter(1));
    assert_eq!(
        recipe.extent,
        HoleExtent::Distance(DatumLength::Constant(0.2))
    );
    assert_eq!(
        recipe.position,
        HolePosition::ModelPoint([
            DatumLength::Parameter(1),
            DatumLength::Constant(-0.2),
            DatumLength::Constant(0.)
        ])
    );
}

#[test]
fn unsupported_contract_options_are_rejected_instead_of_ignored() {
    let mut variants = Vec::new();
    let mut input = request();
    input.type_ = "counterbore".into();
    variants.push(input);
    let mut input = request();
    input.tap = "tapped".into();
    variants.push(input);
    let mut input = request();
    input.drill_point = "angled".into();
    variants.push(input);
    let mut input = request();
    input.to_face = "face".into();
    variants.push(input);
    let mut input = request();
    input.placement_flipped = true;
    variants.push(input);
    for input in variants {
        assert!(
            input::require_supported(&input).is_err(),
            "unexpectedly accepted {input:?}"
        );
    }
    let mut defaults = request();
    defaults.type_ = "drilled".into();
    defaults.tap = "none".into();
    defaults.drill_point = "flat".into();
    assert!(input::require_supported(&defaults).is_ok());
}

#[test]
fn extent_and_position_reject_malformed_or_ambiguous_inputs() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    for (termination, depth) in [
        ("distance", ""),
        ("through-all", "2 mm"),
        ("to-face", ""),
        ("distance", "0 mm"),
    ] {
        let mut input = request();
        input.termination = termination.into();
        input.depth = depth.into();
        assert!(input::recipe(&snapshot, key(4), key(1), &input).is_err());
    }
    let mut input = request();
    input.center = Some(vec![1., 2.]);
    assert!(input::recipe(&snapshot, key(4), key(1), &input).is_err());
    input.center_expr = Some(vec!["1 mm".into()]);
    assert!(input::recipe(&snapshot, key(4), key(1), &input).is_err());
    input.center_expr = Some(vec!["1 deg".into(), "0".into(), "0".into()]);
    assert!(input::recipe(&snapshot, key(4), key(1), &input).is_err());
}
