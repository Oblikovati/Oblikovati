// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchBlockInstance},
    sketch::{Sketch2D, SketchEntityId},
};

fn block_history() -> DocumentEditHistory {
    let mut initial = snapshot();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(2), [2.0, 0.0]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let sketch = DocumentSketchId(1);
    initial
        .sketches
        .add(sketch, "Sketch", SketchPlane::xy(), geometry)
        .unwrap();
    initial
        .sketches
        .capture_block_definition("Rib", sketch, &[SketchEntityId(3)])
        .unwrap();
    for id in [10, 11] {
        let instance = SketchBlockInstance::new(
            SketchEntityId(id),
            "Rib",
            [10.0, 20.0],
            std::f64::consts::FRAC_PI_2,
            2.0,
        )
        .unwrap();
        initial
            .sketches
            .insert_block_instance(sketch, instance)
            .unwrap();
    }
    DocumentEditHistory::new(initial, RecipeIdAllocator::default()).unwrap()
}

#[test]
fn explode_is_one_undoable_edit_and_preserves_other_instances_and_definition() {
    let mut history = block_history();
    let sketch = DocumentSketchId(1);
    let token = history.begin("Explode").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.explode_sketch_block(sketch, SketchEntityId(10), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let placed = history.snapshot().sketches.get(sketch).unwrap();
    assert_eq!(placed.block_instances().len(), 1);
    assert_eq!(placed.block_instances()[0].id(), SketchEntityId(11));
    assert_eq!(history.snapshot().sketches.block_definitions().len(), 1);
    let line = placed.geometry().line(report.copies[0]).unwrap();
    let end = placed.geometry().point(line.endpoints[1]).unwrap().position;
    assert!((end[0] - 10.0).abs() < 1e-12 && (end[1] - 24.0).abs() < 1e-12);
    history.undo().unwrap();
    assert_eq!(
        history
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .block_instances()
            .len(),
        2
    );
    history.redo().unwrap();
    let placed = history.snapshot().sketches.get(sketch).unwrap();
    assert_eq!(placed.block_instances().len(), 1);
    assert!(placed.geometry().line(report.copies[0]).is_ok());
}

#[test]
fn missing_instance_leaves_document_unchanged() {
    let mut history = block_history();
    let before = history.snapshot().sketches.clone();
    let token = history.begin("Explode").unwrap();
    assert!(
        history
            .edit(&token, |snapshot, ids| snapshot.explode_sketch_block(
                DocumentSketchId(1),
                SketchEntityId(99),
                ids
            ))
            .is_err()
    );
    assert!(history.snapshot().sketches.same_recipe(&before));
    history.abort(&token).unwrap();
}
