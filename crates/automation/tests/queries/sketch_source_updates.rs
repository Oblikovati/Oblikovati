// SPDX-License-Identifier: GPL-2.0-only
use super::super::sketch_updates::{circle_radius, create_circle};
use super::*;
#[test]
fn source_refresh_recomputes_derived_sketch_geometry_without_an_authored_target_step() {
    let (mut files, source, target, _) = linked_files();
    let sketch = files
        .commands(target)
        .unwrap()
        .run("Sketch", create_circle)
        .unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    assert!((circle_radius(files.workspace(), target, sketch) - 2.0).abs() < 1e-8);
    let before = files
        .workspace()
        .session(target)
        .unwrap()
        .undo_labels()
        .len();
    files
        .commands(source)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    assert!((circle_radius(files.workspace(), target, sketch) - 5.0).abs() < 1e-8);
    assert_eq!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .len(),
        before
    );
    assert!(!files.refresh_derived_table(target, 7).unwrap());
}
