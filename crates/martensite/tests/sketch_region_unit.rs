// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn chain(sketch: &mut Sketch2D, first_id: u64, corners: &[[f64; 2]]) {
    let points: Vec<SketchEntityId> = (0..corners.len() as u64)
        .map(|offset| SketchEntityId(first_id + offset))
        .collect();
    for (id, corner) in points.iter().zip(corners) {
        sketch.add_point(*id, *corner).unwrap();
    }
    for index in 0..points.len() {
        let line = SketchEntityId(first_id + 100 + index as u64);
        sketch
            .add_line(line, [points[index], points[(index + 1) % points.len()]])
            .unwrap();
    }
}
fn close(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < 1e-9,
        "{actual}: expected {expected}"
    );
}
#[test]
fn a_closed_triangle_reports_its_area_perimeter_and_centroid() {
    let mut sketch = Sketch2D::default();
    chain(&mut sketch, 1, &[[0., 0.], [3., 0.], [0., 4.]]);
    let region = sketch_region_at(&sketch, [1., 1.]).unwrap().unwrap();
    close(region.area, 6.);
    close(region.perimeter, 12.);
    close(region.centroid[0], 1.);
    close(region.centroid[1], 4. / 3.);
}
#[test]
fn an_open_chain_bounds_nothing_and_an_outside_point_finds_nothing() {
    let mut sketch = Sketch2D::default();
    let points = [1, 2, 3].map(SketchEntityId);
    for (id, at) in points.iter().zip([[0., 0.], [3., 0.], [0., 4.]]) {
        sketch.add_point(*id, at).unwrap();
    }
    sketch
        .add_line(SketchEntityId(10), [points[0], points[1]])
        .unwrap();
    sketch
        .add_line(SketchEntityId(11), [points[1], points[2]])
        .unwrap();
    assert_eq!(sketch_region_at(&sketch, [1., 1.]).unwrap(), None);
    chain(&mut sketch, 100, &[[20., 20.], [21., 20.], [21., 21.]]);
    assert_eq!(sketch_region_at(&sketch, [0., 50.]).unwrap(), None);
}
#[test]
fn a_nested_loop_is_its_own_region_and_the_ring_excludes_it() {
    let mut sketch = Sketch2D::default();
    chain(&mut sketch, 1, &[[0., 0.], [6., 0.], [6., 6.], [0., 6.]]);
    chain(&mut sketch, 20, &[[2., 2.], [4., 2.], [4., 4.], [2., 4.]]);
    close(
        sketch_region_at(&sketch, [3., 3.]).unwrap().unwrap().area,
        4.,
    );
    let ring = sketch_region_at(&sketch, [1., 1.]).unwrap().unwrap();
    close(ring.area, 32.);
    close(ring.perimeter, 32.);
}
