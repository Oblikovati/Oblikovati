// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
fn constrained_collection(vertical: bool) -> DocumentSketches {
    let mut collection = sketches();
    let sketch = DocumentSketchId(3);
    let mut geometry = collection.get(sketch).unwrap().geometry().clone();
    let line = SketchEntityId(7);
    let relation = if vertical {
        PlanarConstraint::LineVertical { line }
    } else {
        PlanarConstraint::LineHorizontal { line }
    };
    geometry
        .add_constraint(SketchConstraintId(1), relation)
        .unwrap();
    collection.replace_geometry(sketch, geometry).unwrap();
    collection
}
#[test]
fn both_line_alignment_kinds_round_trip_with_their_owned_operand() {
    for vertical in [false, true] {
        let expected = constrained_collection(vertical);
        let saved = package().with_sketches(&expected, 9, 15).unwrap();
        let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
        let restored = reopened.restore_sketches().unwrap();
        assert!(restored.sketches.same_recipe(&expected));
        let placed = restored.sketches.get(DocumentSketchId(3)).unwrap();
        let geometry = placed.geometry();
        assert_eq!(
            geometry
                .geometric_constraint_index(SketchConstraintId(1))
                .unwrap(),
            0
        );
        assert_eq!(restored.constraint_ids.marks()[&3], 1);
    }
}
#[test]
fn native_line_relations_with_invalid_operands_are_refused_before_restore() {
    for vertical in [false, true] {
        let saved = package()
            .with_sketches(&constrained_collection(vertical), 9, 15)
            .unwrap();
        let yaml = String::from_utf8(saved.encode().unwrap()).unwrap();
        assert!(yaml.contains("line: 7"));
        for line in [0, 2, 99] {
            let modified = yaml.replace("line: 7", &format!("line: {line}"));
            let malformed = DocumentPackage::decode(modified.as_bytes()).unwrap();
            let cause = malformed.restore_sketches().unwrap_err().to_string();
            assert!(
                cause.contains("model.planarSketches") && cause.contains(&line.to_string()),
                "{cause}"
            );
        }
    }
}
