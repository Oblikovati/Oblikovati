// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::SemanticReferenceName;

#[test]
fn ambiguous_direct_history_is_refused() {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let shape = martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox::from_sizes(1., 1., 1.)
        .unwrap()
        .shape()
        .unwrap()
        .clone();
    let face = shapes(&shape, ShapeEnum::Face).unwrap().remove(0);
    let record = |id| BlendHistoryRecord {
        name: SemanticReferenceName::new(document, id, 1, "fixture", "face", &[1])
            .unwrap()
            .encode()
            .unwrap(),
        source: face.clone(),
        generated: Vec::new(),
        modified: Vec::new(),
        deleted: false,
        transient: false,
    };
    let target = record(1);
    let expected = target.name.clone();
    let tool = BlendHistoryRecord {
        transient: true,
        ..record(2)
    };
    assert_eq!(
        direct_name(&[target, tool], &face, 1, document, FeatureId(3)).unwrap(),
        Some(expected),
    );
    assert!(
        direct_name(&[record(1), record(2)], &face, 1, document, FeatureId(3))
            .unwrap_err()
            .contains("unambiguous")
    );
}
