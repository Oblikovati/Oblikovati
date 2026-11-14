// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{
    DisplayTolerance, ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId,
    Resolution, SketchProfile, ToleranceCap, build_planar_regions,
};
use oblikovati_model::sketch::SketchEntityId;

fn ring(first: u64, coordinates: [[f64; 2]; 4]) -> (Vec<ProfileVertex>, Vec<ProfileCurve>) {
    let vertices = coordinates
        .into_iter()
        .enumerate()
        .map(|(index, point)| ProfileVertex {
            id: ProfileVertexId(first + index as u64),
            point,
        })
        .collect();
    let curves = (0..4)
        .map(|index| ProfileCurve::Line {
            id: ProfileCurveId(first + 10 + index),
            start: ProfileVertexId(first + index),
            end: ProfileVertexId(first + (index + 1) % 4),
        })
        .collect();
    (vertices, curves)
}
fn rectangle(first: u64, low: f64, high: f64) -> (Vec<ProfileVertex>, Vec<ProfileCurve>) {
    ring(first, [[low, low], [high, low], [high, high], [low, high]])
}
fn nested_regions(inner_first: bool) -> PlanarRegionSet {
    let (mut vertices, mut curves) = rectangle(1, 0., 10.);
    let (inner_vertices, inner_curves) = rectangle(101, 2., 4.);
    vertices.extend(inner_vertices);
    if inner_first {
        curves.splice(0..0, inner_curves);
    } else {
        curves.extend(inner_curves);
    }
    let profile = SketchProfile {
        plane: ProfilePlane::world_xy(),
        vertices,
        curves,
    };
    let resolution = Resolution::new(
        1e-6,
        1e-7,
        1e-9,
        DisplayTolerance {
            chordal: 1e-3,
            angular: 0.1,
        },
        ToleranceCap::Refuse(1e-5),
    );
    build_planar_regions(&profile, resolution).unwrap()
}
fn boundary(first: u64) -> Vec<SketchDirectedCurve> {
    (0..4)
        .map(|offset| SketchDirectedCurve {
            curve: SketchEntityId(first + 10 + offset),
            reversed: false,
        })
        .collect()
}
fn reverse(boundary: &mut [SketchDirectedCurve]) {
    boundary.reverse();
    for edge in boundary {
        edge.reversed = !edge.reversed;
    }
}
#[test]
fn saved_ring_survives_region_reordering_and_reversed_loop_traversal() {
    let mut outer = boundary(1);
    let mut hole = boundary(101);
    outer.rotate_left(2);
    reverse(&mut outer);
    hole.rotate_left(1);
    let selection = SketchPrismSelection {
        outer,
        holes: vec![hole],
    };
    let first = nested_regions(false);
    let second = nested_regions(true);
    let before = resolve_region_by_boundary(&first, &selection).unwrap();
    let after = resolve_region_by_boundary(&second, &selection).unwrap();
    assert_ne!(before.id(), after.id());
    assert!(before.area().encloses(96.));
    assert!(after.area().encloses(96.));
}
#[test]
fn missing_or_incorrect_boundary_is_a_named_decline() {
    let regions = nested_regions(false);
    let mut selection = SketchPrismSelection {
        outer: boundary(1),
        holes: vec![boundary(101)],
    };
    selection.outer.swap(1, 2);
    assert!(matches!(
        resolve_region_by_boundary(&regions, &selection),
        Err(RegionSelectionError::Missing { .. })
    ));
    selection.outer = boundary(101);
    selection.holes.clear();
    assert!(
        resolve_region_by_boundary(&regions, &selection)
            .unwrap()
            .area()
            .encloses(4.)
    );
}

#[test]
fn matching_requires_consistent_direction_for_every_edge() {
    let regions = nested_regions(false);
    let mut selection = SketchPrismSelection {
        outer: boundary(1),
        holes: vec![boundary(101)],
    };
    selection.outer[1].reversed = true;
    assert!(matches!(
        resolve_region_by_boundary(&regions, &selection),
        Err(RegionSelectionError::Missing { .. })
    ));
}
