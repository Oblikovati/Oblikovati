// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn hidden_constraint_glyphs_survive_native_save_and_reopen() {
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
    let id = DocumentSketchId(3);
    let relation = SketchConstraintId(1);
    let mut document = sketches();
    let mut geometry = document.get(id).unwrap().geometry().clone();
    geometry
        .add_constraint(
            relation,
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(7),
            },
        )
        .unwrap();
    document.replace_geometry(id, geometry.clone()).unwrap();
    document.set_constraints_shown(id, true).unwrap();
    document
        .set_constraint_glyph_visible(id, relation, false)
        .unwrap();
    let saved = package().with_sketches(&document, 9, 15).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let restored = reopened.restore_sketches().unwrap().sketches;
    assert_eq!(
        restored
            .get(id)
            .unwrap()
            .hidden_constraint_glyphs()
            .iter()
            .copied()
            .collect::<Vec<_>>(),
        [relation]
    );
    assert_eq!(restored.get(id).unwrap().geometry(), &geometry);
    assert!(restored.get(id).unwrap().constraints_shown());
}
