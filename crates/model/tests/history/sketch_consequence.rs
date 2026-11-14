// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{PlanarConstraint, Sketch2D, SketchPlane};

fn unsolved_history() -> DocumentEditHistory {
    let mut owner = history();
    let token = owner.begin("Create sketch").unwrap();
    owner
        .edit(&token, |snapshot, ids| {
            let sketch = ids.allocate_sketch().unwrap();
            let point = ids.allocate_sketch_entity().unwrap();
            let constraint = ids.allocate_sketch_constraint(sketch).unwrap();
            let mut geometry = Sketch2D::default();
            geometry.add_point(point, [0., 0.]).unwrap();
            geometry
                .add_constraint(
                    constraint,
                    PlanarConstraint::Fix {
                        point,
                        position: [2., 0.],
                    },
                )
                .unwrap();
            snapshot
                .sketches
                .add(sketch, "Profile", SketchPlane::xy(), geometry)
                .unwrap();
            Ok::<_, ()>(())
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
    owner
}
#[test]
fn nested_derived_solve_does_not_create_an_undo_step() {
    let mut owner = unsolved_history();
    let before = owner.snapshot().sketches.clone();
    let solved = owner
        .snapshot()
        .solve_pending_document_sketches(Default::default(), || false)
        .candidate;
    assert!(!solved.same_recipe(&before));
    let count = owner.undo_labels().len();
    let outer = owner.begin("Outer").unwrap();
    let inner = owner.begin("Inner").unwrap();
    owner.edit_sketch_consequence::<()>(&inner, solved).unwrap();
    owner.commit(&inner, false).unwrap();
    assert!(owner.discard_unchanged(&outer).unwrap());
    assert_eq!(owner.undo_labels().len(), count);
    assert!(!owner.snapshot().sketches.same_recipe(&before));
}
#[test]
fn nested_derived_solve_does_not_hide_an_authored_edit() {
    let mut owner = unsolved_history();
    let before = owner.snapshot().sketches.clone();
    let outer = owner.begin("Change width").unwrap();
    set_width(&mut owner, &outer, "20 mm");
    let solved = owner
        .snapshot()
        .solve_pending_document_sketches(Default::default(), || false)
        .candidate;
    let inner = owner.begin("Solve").unwrap();
    owner.edit_sketch_consequence::<()>(&inner, solved).unwrap();
    assert!(owner.discard_unchanged(&inner).unwrap());
    assert!(!owner.discard_unchanged(&outer).unwrap());
    owner.commit(&outer, false).unwrap();
    owner.undo().unwrap();
    assert_eq!(width(&owner), 1.);
    assert!(owner.snapshot().sketches.same_recipe(&before));
}
