// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{NamedPrismSnapshot, retain_named_sketch_prism};
use oblikovati_model::{
    feature::FeatureId,
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
};
fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn owner(snapshot: DocumentSnapshot) -> DocumentEditHistory {
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}
fn named(history: &DocumentEditHistory) -> NamedPrismSnapshot {
    retain_named_sketch_prism(history, document(), FeatureId(1)).unwrap()
}
fn key(role: &str, sources: &[u64]) -> Vec<u8> {
    let kind = if role == "body" { 4 } else { 1 };
    SemanticReferenceName::new(document(), 1, kind, "sketch-prism", role, sources)
        .unwrap()
        .encode()
        .unwrap()
}
fn assert_centroid(actual: [f64; 3], expected: [f64; 3]) {
    // Analytic integration can round differently across platforms at these centimetre scales.
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() <= expected.abs().max(1.) * 1e-11,
            "centroid coordinate {actual}, expected {expected}"
        );
    }
}
fn assert_faces(body: &NamedPrismSnapshot, history: &DocumentEditHistory, depth: f64) {
    for role in ["start", "end"] {
        let face = body
            .face_report(history, document(), &key(role, &[1]))
            .unwrap();
        assert_volume(face.area, 12.);
        assert_eq!(face.loops, 1);
        assert_centroid(
            face.centroid,
            [2., 1.5, if role == "start" { 0. } else { depth }],
        );
    }
    for (edge, length) in [(101, 4.), (102, 3.), (103, 4.), (104, 3.)] {
        let face = body
            .face_report(history, document(), &key("side", &[1, edge]))
            .unwrap();
        assert_volume(face.area, length * depth);
        assert_eq!(face.role, PrismFaceRole::ProfileEdge(edge));
        assert_eq!(face.loops, 1);
    }
}
#[test]
fn named_faces_follow_depth_edits_and_undo_without_reusing_old_handles() {
    let mut history = owner(selected_prism_document());
    let first = named(&history);
    let keys = first.reference_keys(&history, document()).unwrap();
    assert_eq!(keys.len(), 7);
    assert_eq!(
        first.body_reference_key(&history, document()).unwrap(),
        key("body", &[1])
    );
    assert_faces(&first, &history, 2.);
    let token = history.begin("Depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(first.reference_keys(&history, document()).is_err());
    assert!(first.body_reference_key(&history, document()).is_err());
    assert!(
        first
            .physical_properties(&history, document(), &key("body", &[1]))
            .is_err()
    );
    let changed = named(&history);
    assert_eq!(changed.reference_keys(&history, document()).unwrap(), keys);
    assert_faces(&changed, &history, 3.);
    history.undo().unwrap();
    assert!(
        changed
            .face_report(&history, document(), &key("start", &[1]))
            .is_err()
    );
    assert_faces(&named(&history), &history, 2.);
}
#[test]
fn exact_names_reject_wrong_kind_source_feature_document_and_history() {
    let history = owner(selected_prism_document());
    let body = named(&history);
    let body_key = key("body", &[1]);
    assert_volume(
        body.body_report(&history, document(), &body_key)
            .unwrap()
            .properties
            .volume,
        24.,
    );
    assert!(body.face_report(&history, document(), &body_key).is_err());
    assert!(
        body.body_report(&history, document(), &key("start", &[1]))
            .is_err()
    );
    assert!(
        body.face_report(&history, document(), &key("side", &[1, 999]))
            .is_err()
    );
    let other_doc = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap();
    assert!(body.body_report(&history, other_doc, &body_key).is_err());
    assert!(
        body.body_report(&owner(selected_prism_document()), document(), &body_key)
            .is_err()
    );
    assert!(retain_named_sketch_prism(&history, document(), FeatureId(99)).is_err());
}
#[test]
fn profile_parameter_changes_preserve_source_names_and_change_exact_face_area() {
    let mut history = owner(selected_prism_document());
    let before = named(&history)
        .reference_keys(&history, document())
        .unwrap();
    history
        .refresh(|snapshot| {
            snapshot
                .parameters
                .set_expression(1, "6 cm")
                .map_err(|error| error.to_string())?;
            solve_selected_sketches(snapshot);
            Ok::<(), String>(())
        })
        .unwrap();
    let body = named(&history);
    assert_eq!(body.reference_keys(&history, document()).unwrap(), before);
    assert_volume(
        body.face_report(&history, document(), &key("start", &[1]))
            .unwrap()
            .area,
        18.,
    );
    assert_volume(
        body.face_report(&history, document(), &key("side", &[1, 101]))
            .unwrap()
            .area,
        12.,
    );
}

#[path = "named/rebuild.rs"]
mod rebuild;
#[test]
fn part_properties_require_every_body_to_belong_to_the_current_history() {
    let mut history = owner(selected_prism_document());
    let first = named(&history);
    let other_history = owner(selected_prism_document());
    let foreign = named(&other_history);
    assert!(
        oblikovati_martensite::named_part_properties(&history, document(), [&first, &foreign])
            .is_err()
    );
    let key = first.body_reference_key(&history, document()).unwrap();
    assert_eq!(
        oblikovati_martensite::named_part_properties(&history, document(), [&first])
            .unwrap()
            .unwrap(),
        first
            .physical_properties(&history, document(), key)
            .unwrap()
    );
    let token = history.begin("Depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(oblikovati_martensite::named_part_properties(&history, document(), [&first]).is_err());
}
