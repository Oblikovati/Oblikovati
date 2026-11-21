// SPDX-License-Identifier: GPL-2.0-only
//! Hole API request classification and driving inputs (#4093).
use super::*;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    feature::{
        hole::{HoleExtent, HolePosition},
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
};
use oblikovati_model::{history::DocumentEditHistory, identity::RecipeIdAllocator};
use oblikovati_session::{DocumentSession, TransactionObserver};

struct SilentHoleObserver;
impl TransactionObserver for SilentHoleObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}

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
    let definition = prepare_hole(&snapshot, key(4), key(1), &input)
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
        let snapshot = crate::prism_test_fixture::selected_prism_document();
        assert!(prepare_hole(&snapshot, key(4), key(1), &input).is_err());
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

#[test]
fn prepared_hole_preview_leaves_the_document_unchanged() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let owner = DocumentSession::new(1, history, SilentHoleObserver);
    let before = owner.generation();
    let prepared = prepare_hole(owner.snapshot(), key(4), key(1), &request()).unwrap();
    let kind = owner
        .preview_projected(
            |candidate, ids| prepared.create(candidate, ids),
            |candidate, id| {
                Ok::<_, Error>(
                    candidate
                        .work_geometry
                        .get(id)
                        .unwrap()
                        .definition()
                        .kind()
                        .to_owned(),
                )
            },
        )
        .unwrap();
    assert_eq!(kind, "hole");
    assert_eq!(owner.generation(), before);
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
}
