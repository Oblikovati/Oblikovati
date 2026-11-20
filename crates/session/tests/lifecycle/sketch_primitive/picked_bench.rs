// SPDX-License-Identifier: GPL-2.0-only
//! Release-only placement preview timing; this is not a desktop frame benchmark.
use super::*;
use oblikovati_model::sketch::{
    SketchPlacementBinding, SketchPlacementRecipe, SketchPlacementRelation,
};
use std::time::Instant;

fn large_session(count: usize) -> DocumentSession<RecordingObserver> {
    let mut geometry = Sketch2D::default();
    for index in 0..count {
        let id = 100 + (index as u64) * 3;
        let x = 10. + index as f64 * 2.;
        geometry.add_point(SketchEntityId(id), [x, 0.]).unwrap();
        geometry
            .add_point(SketchEntityId(id + 1), [x + 1., 0.])
            .unwrap();
        geometry
            .add_line(
                SketchEntityId(id + 2),
                [SketchEntityId(id), SketchEntityId(id + 1)],
            )
            .unwrap();
    }
    geometry.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    super::super::sketch_fixture::owned_sketch_session(None, "Timing", geometry)
}

fn timed_recipe(pick: bool) -> SketchPlacementRecipe {
    let mut recipe = SketchPlacementRecipe::line_chain(&[[-0.5, 0.], [0.5, 0.]], false).unwrap();
    recipe.points.push([0., 0.]);
    recipe
        .relations
        .push(SketchPlacementRelation::Midpoint { point: 2, line: 0 });
    recipe.fields[1].binding = SketchPlacementBinding::LineAngle(0);
    if pick {
        recipe.picked_points.insert(2, SketchEntityId(1));
    }
    recipe
}

#[test]
#[ignore = "release timing only; run with --ignored --nocapture"]
fn picked_point_preview_large_sketch_timing() {
    let count: usize = std::env::var("OBK_POINT_BENCH_LINES")
        .unwrap()
        .parse()
        .unwrap();
    let pick = std::env::var("OBK_POINT_BENCH_PICK").unwrap() == "1";
    let session = large_session(count);
    let recipe = timed_recipe(pick);
    let locks = ["10 mm", "0 deg"].map(|expression| {
        Some(oblikovati_model::history::SketchPlacementFieldLock {
            reference: false,
            expression: expression.into(),
            text_point: None,
        })
    });
    let verifier = super::super::placement_verification::RecordingCircularVerifier::default();
    let sketch = DocumentSketchId(1);
    for _ in 0..7 {
        let started = Instant::now();
        let result = session
            .preview_sketch_placement(sketch, &recipe, &locks, &verifier)
            .unwrap();
        assert_eq!(result.members.len(), 1);
        eprintln!(
            "lines={count} pick={pick} elapsed_ms={:.3}",
            started.elapsed().as_secs_f64() * 1000.
        );
    }
}
