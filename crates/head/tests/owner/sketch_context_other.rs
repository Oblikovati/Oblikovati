// SPDX-License-Identifier: GPL-2.0-only
//! Context picking at coincident geometry (#4041).
use super::*;
use crate::{HeadPresentation, HeadWindow};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};
use std::time::Duration;

struct PlanarContextWindow;
impl HeadWindow for PlanarContextWindow {
    fn sketch_point(&self, pixel: [i32; 2], _: SketchPlane) -> Result<Option<[f64; 2]>, String> {
        Ok(Some(pixel.map(f64::from)))
    }
    fn sketch_pixel(&self, _: SketchPlane, point: [f64; 2]) -> Result<[i32; 2], String> {
        Ok(point.map(|value| value.round() as i32))
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
#[test]
fn context_lists_curve_with_overlapping_point() {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    geometry
        .add_defining_point(SketchEntityId(2), [-10., 0.])
        .unwrap();
    geometry
        .add_defining_point(SketchEntityId(3), [10., 0.])
        .unwrap();
    geometry
        .add_line(SketchEntityId(4), [SketchEntityId(2), SketchEntityId(3)])
        .unwrap();
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        look_at_plane: false,
    };
    let target = context_target(&geometry, edit, 1, [0, 0], &PlanarContextWindow).unwrap();
    let offered: Vec<_> = target
        .candidates
        .iter()
        .map(|item| (item.entity, item.label.as_str()))
        .collect();
    assert_eq!(
        offered,
        [(SketchEntityId(1), "Point"), (SketchEntityId(4), "Line")]
    );
}

#[test]
fn context_candidates_keep_point_priority_reach_and_stable_curve_order() {
    let mut sketch = Sketch2D::default();
    for (id, at) in [(1, [0., 0.]), (2, [5., 0.]), (3, [7., 0.])] {
        sketch.add_point(SketchEntityId(id), at).unwrap();
    }
    for (line, a, b, y) in [(20, 21, 22, 10.), (10, 11, 12, 10.), (30, 31, 32, 13.)] {
        sketch
            .add_defining_point(SketchEntityId(a), [-30., y])
            .unwrap();
        sketch
            .add_defining_point(SketchEntityId(b), [30., y])
            .unwrap();
        sketch
            .add_line(SketchEntityId(line), [SketchEntityId(a), SketchEntityId(b)])
            .unwrap();
    }
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        look_at_plane: false,
    };
    let target = context_target(&sketch, edit, 1, [0, 0], &PlanarContextWindow).unwrap();
    let ids: Vec<_> = target
        .candidates
        .iter()
        .map(|candidate| candidate.entity.0)
        .collect();
    assert_eq!(ids, [1, 2, 10, 20]);
}

#[test]
fn context_offers_circle_alongside_point() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch
        .add_defining_point(SketchEntityId(2), [-20., 0.])
        .unwrap();
    sketch
        .add_circle(SketchEntityId(3), SketchEntityId(2), 20.)
        .unwrap();
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        look_at_plane: false,
    };
    let target = context_target(&sketch, edit, 1, [0, 0], &PlanarContextWindow).unwrap();
    let offered: Vec<_> = target
        .candidates
        .iter()
        .map(|item| (item.entity, item.label.as_str()))
        .collect();
    assert_eq!(
        offered,
        [(SketchEntityId(1), "Point"), (SketchEntityId(3), "Circle")]
    );
}
