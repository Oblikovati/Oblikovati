// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn coincident_creation_without_an_edit_preference_splits_motion() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [8.0, 4.0]).unwrap();
    let mut history = owner(sketch);
    create(
        &mut history,
        &direct(PlanarConstraint::Coincident {
            points: [entity(1), entity(2)],
        }),
    );
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    for id in [entity(1), entity(2)] {
        let position = geometry.point(id).unwrap().position;
        assert!((position[0] - 4.0).abs() < 1e-6);
        assert!((position[1] - 2.0).abs() < 1e-6);
    }
}
