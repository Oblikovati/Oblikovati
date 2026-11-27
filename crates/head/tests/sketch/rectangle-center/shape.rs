// SPDX-License-Identifier: GPL-2.0-only
//! Check that the two construction diagonals connect opposite rectangle corners.
use oblikovati_model::sketch::{Sketch2D, SketchEntityId, SketchLine2D};
use std::collections::{BTreeMap, BTreeSet};

pub(super) fn expect_centered_rectangle(sketch: &Sketch2D) {
    let (sides, diagonals) = rectangle_members(sketch);
    assert_eq!(sides.len(), 4, "expected four shared corners");
    assert!(sides.values().all(|count| *count == 2));
    let diagonal_points: BTreeSet<_> = diagonals.iter().flatten().copied().collect();
    assert_eq!(diagonals.len(), 2);
    assert_eq!(diagonal_points, sides.keys().copied().collect());
    assert!(diagonals.iter().all(|pair| pair[0] != pair[1]));
    let [first, second] = [
        midpoint(sketch, diagonals[0]),
        midpoint(sketch, diagonals[1]),
    ];
    assert!((first[0] - second[0]).abs() < 1e-8);
    assert!((first[1] - second[1]).abs() < 1e-8);
}

fn rectangle_members(
    sketch: &Sketch2D,
) -> (BTreeMap<SketchEntityId, usize>, Vec<[SketchEntityId; 2]>) {
    let index = sketch.read_index();
    let mut sides = BTreeMap::<SketchEntityId, usize>::new();
    let mut diagonals = Vec::new();
    for line in sketch.lines() {
        if index.is_construction(line.id).unwrap() {
            diagonals.push(line.endpoints);
            continue;
        }
        for point in line.endpoints {
            *sides.entry(point).or_default() += 1;
        }
    }
    (sides, diagonals)
}

fn midpoint(sketch: &Sketch2D, pair: [SketchEntityId; 2]) -> [f64; 2] {
    let [a, b] = pair.map(|id| sketch.point(id).unwrap().position);
    [(a[0] + b[0]) * 0.5, (a[1] + b[1]) * 0.5]
}

pub(super) fn expect_latest_sizes(sketch: &Sketch2D, short: f64, long: f64) {
    let index = sketch.read_index();
    let mut lengths: Vec<f64> = sketch
        .lines()
        .iter()
        .rev()
        .filter(|line| !index.is_construction(line.id).unwrap())
        .take(4)
        .map(|line| side_length(sketch, line))
        .collect();
    lengths.sort_by(f64::total_cmp);
    for (actual, expected) in lengths.iter().zip([short, short, long, long]) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "size {actual}: expected {expected}"
        );
    }
    assert_eq!(lengths.len(), 4, "expected four rectangle sides");
}

fn side_length(sketch: &Sketch2D, line: &SketchLine2D) -> f64 {
    let [a, b] = line.endpoints.map(|id| sketch.point(id).unwrap().position);
    (a[0] - b[0]).hypot(a[1] - b[1])
}

pub(super) fn expect_axis_sizes(sketch: &Sketch2D, sizes: [f64; 2]) {
    let index = sketch.read_index();
    let mut extents = [0.0_f64; 2];
    for line in sketch
        .lines()
        .iter()
        .filter(|line| !index.is_construction(line.id).unwrap())
    {
        let [a, b] = line.endpoints.map(|id| sketch.point(id).unwrap().position);
        for axis in 0..2 {
            extents[axis] = extents[axis].max((a[axis] - b[axis]).abs());
        }
    }
    for (actual, expected) in extents.into_iter().zip(sizes) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "axis size {actual}: expected {expected}"
        );
    }
}
