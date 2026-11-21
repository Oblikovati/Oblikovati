// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::{
        DocumentSketchId, DocumentSketches, PlacedSketchHealth, SketchFrameEditDefinition,
        SketchFrameOrigin,
    },
    parameter::ParameterCollection,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
    units::UnitsOfMeasure,
};

#[test]
fn changed_frame_invalidates_the_solve_but_an_identical_edit_does_not() {
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let sketch = DocumentSketchId(1);
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [1., 0.]).unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(sketch, "Sketch", SketchPlane::xy(), geometry)
        .unwrap();
    sketches = sketches
        .solve_pending(&parameters, Default::default(), || false)
        .candidate;
    assert!(matches!(
        sketches.get(sketch).unwrap().health(&parameters),
        PlacedSketchHealth::Current(_)
    ));
    let edit = Some(SketchFrameEditDefinition {
        origin: Some(SketchFrameOrigin::Position([1., 2., 0.])),
        alignment: None,
    });
    sketches.set_frame_edit(sketch, edit.clone()).unwrap();
    assert!(matches!(
        sketches.get(sketch).unwrap().health(&parameters),
        PlacedSketchHealth::Unevaluated
    ));
    sketches = sketches
        .solve_pending(&parameters, Default::default(), || false)
        .candidate;
    sketches.set_frame_edit(sketch, edit).unwrap();
    assert!(matches!(
        sketches.get(sketch).unwrap().health(&parameters),
        PlacedSketchHealth::Current(_)
    ));
}
