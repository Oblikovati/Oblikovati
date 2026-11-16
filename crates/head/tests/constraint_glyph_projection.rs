// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchPlane},
};
use std::cell::Cell;

#[derive(Default)]
struct FakeGlyphWindow {
    batches: Cell<usize>,
    points: Cell<usize>,
    reject_batch: bool,
}
impl HeadWindow for FakeGlyphWindow {
    fn sketch_pixels(&self, _: SketchPlane, points: &[[f64; 2]]) -> Result<Vec<[i32; 2]>, String> {
        self.batches.set(self.batches.get() + 1);
        if self.reject_batch {
            return Err("batch: expected valid points".into());
        }
        Ok(points.iter().map(|p| [p[0] as i32, p[1] as i32]).collect())
    }
    fn sketch_pixel(&self, _: SketchPlane, point: [f64; 2]) -> Result<[i32; 2], String> {
        self.points.set(self.points.get() + 1);
        if point[0] < 0. {
            return Err("point: expected nonnegative X".into());
        }
        Ok([point[0] as i32, point[1] as i32])
    }
    fn events(&mut self, _: usize) -> Result<Vec<oblikovati_viewport::ViewportEvent>, String> {
        Ok(vec![])
    }
    fn present(&mut self) -> Result<crate::HeadPresentation, String> {
        Ok(crate::HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
fn mark(id: u64, x: f64, plane: SketchPlane) -> crate::HeadConstraintMark {
    crate::HeadConstraintMark {
        sketch: DocumentSketchId(1),
        constraint: SketchConstraintId(id),
        plane,
        at: [x, 2.],
        text: id.to_string(),
        related: vec![],
    }
}
#[test]
fn matching_plane_projects_once_and_keeps_mark_order() {
    let window = FakeGlyphWindow::default();
    let marks = vec![
        mark(1, 3., SketchPlane::xy()),
        mark(2, 4., SketchPlane::xy()),
    ];
    let result = project_constraint_marks(&window, marks);
    assert_eq!(window.batches.get(), 1);
    assert_eq!(window.points.get(), 0);
    assert_eq!(
        result.iter().map(|g| g.pixel).collect::<Vec<_>>(),
        [[3, 2], [4, 2]]
    );
}
#[test]
fn failed_batch_preserves_other_valid_marks() {
    let window = FakeGlyphWindow {
        reject_batch: true,
        ..Default::default()
    };
    let marks = vec![
        mark(1, -1., SketchPlane::xy()),
        mark(2, 4., SketchPlane::xy()),
    ];
    let result = project_constraint_marks(&window, marks);
    assert_eq!(window.points.get(), 2);
    assert_eq!(result.len(), 1);
    assert_eq!(result[0].pixel, [4, 2]);
}
#[test]
fn mixed_planes_keep_individual_projection() {
    let window = FakeGlyphWindow::default();
    let marks = vec![
        mark(1, 3., SketchPlane::xy()),
        mark(2, 4., SketchPlane::xz()),
    ];
    assert_eq!(project_constraint_marks(&window, marks).len(), 2);
    assert_eq!(window.batches.get(), 0);
    assert_eq!(window.points.get(), 2);
}
