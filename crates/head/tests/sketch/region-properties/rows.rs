// SPDX-License-Identifier: GPL-2.0-only
use super::*;

const TWO_SQUARES: [SketchRegionMeasure; 2] = [
    SketchRegionMeasure {
        area: 1.,
        perimeter: 4.,
        centroid: [0.5, 0.5],
    },
    SketchRegionMeasure {
        area: 1.,
        perimeter: 4.,
        centroid: [2.5, 0.5],
    },
];
#[test]
fn one_loop_reads_in_millimetres_with_its_centroid() {
    let rows = region_result_rows(&TWO_SQUARES[..1], &UnitsOfMeasure::default());
    assert_eq!(
        rows,
        [
            "Area 100.000 mm^2",
            "Perimeter 40.000 mm",
            "Centroid X 5.000 Y 5.000 mm"
        ]
    );
}
#[test]
fn two_loops_add_and_weigh_their_centroids() {
    let rows = region_result_rows(&TWO_SQUARES, &UnitsOfMeasure::default());
    assert_eq!(rows[0], "Area 200.000 mm^2");
    assert_eq!(rows[2], "Centroid X 15.000 Y 5.000 mm");
}
#[test]
fn clicking_an_accepted_loop_again_releases_it_and_any_change_drops_the_result() {
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: oblikovati_model::document_sketch::DocumentSketchId(1),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
        look_at_plane: true,
    };
    let mut state = SketchRegionState {
        edit,
        generation: 0,
        loops: Vec::new(),
        results: vec!["old".into()],
    };
    state.toggle_loop(TWO_SQUARES[0]);
    assert!(state.results.is_empty());
    assert_eq!(state.loop_names(), ["Sketch Loop1"]);
    state.toggle_loop(TWO_SQUARES[0]);
    assert!(state.loops.is_empty());
}
