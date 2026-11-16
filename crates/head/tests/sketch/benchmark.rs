// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch` on the same native window and owner as the application scenarios.
use super::*;
#[path = "arc-center-point/steps.rs"]
mod arc_center_point;
#[path = "arc-three-point/steps.rs"]
mod arc_three_point;
#[path = "circle-center-point/steps.rs"]
mod circle_center_point;
#[path = "coincident/context.rs"]
mod coincident_context;
#[path = "coincident/move_relation.rs"]
mod coincident_move_relation;
#[path = "coincident/multi_move.rs"]
mod coincident_multi_move;
#[path = "coincident/points.rs"]
mod coincident_points;
#[path = "coincident/segments.rs"]
mod coincident_segments;
#[path = "dimension-expression/steps.rs"]
mod dimension_expression;
#[path = "ellipse/steps.rs"]
mod ellipse;
#[path = "fixtures.rs"]
pub(super) mod fixtures;
#[path = "fixtures_midpoint.rs"]
mod fixtures_midpoint;
#[path = "line/steps.rs"]
mod line;
#[path = "move/cancel.rs"]
mod move_cancel;
#[path = "move/close.rs"]
mod move_close;
#[path = "move/fade.rs"]
mod move_fade;
#[path = "move/pin.rs"]
mod move_pin;
#[path = "move/polar.rs"]
mod move_polar;
#[path = "move/precise.rs"]
mod move_precise;
#[path = "new-sketch/steps.rs"]
mod new_sketch;
#[path = "new-sketch/face.rs"]
mod new_sketch_face;
#[path = "new-sketch/work_plane.rs"]
mod new_sketch_work_plane;
#[path = "point/steps.rs"]
mod point;
#[path = "precise-input/steps.rs"]
mod precise_input;
#[path = "rectangle-center/steps.rs"]
mod rectangle_center;
#[path = "rectangle-three-point/steps.rs"]
mod rectangle_three_point;
#[path = "rectangle-three-point-center/steps.rs"]
mod rectangle_three_point_center;
#[path = "rectangle-two-point/steps.rs"]
mod rectangle_two_point;

/// Features the gate requires; each is one finished folder under `features/sketch`.
const ACCEPTED_FEATURES: &[&str] = &[
    "arc-three-point",
    "circle-center-point",
    "dimension-expression",
    "line",
    "point",
    "precise-input",
    "rectangle-center",
    "rectangle-three-point",
    "rectangle-three-point-center",
    "rectangle-two-point",
];
#[test]
fn sketch_scenarios_use_the_native_window() {
    std::thread::Builder::new()
        .name("sketch-scenarios".into())
        .stack_size(SCENARIO_STACK_BYTES)
        .spawn(|| {
            let _reports = report_destination::reserve(&world::reports_folder())
                .unwrap_or_else(|error| panic!("{error}"));
            run_scenarios(feature_root("sketch"), accepted_sketch_feature)
        })
        .unwrap()
        .join()
        .unwrap();
}
/// A folder asked for by name runs alone; otherwise only finished features count against the gate.
fn accepted_sketch_feature(feature: &cucumber::gherkin::Feature) -> bool {
    if std::env::var_os("OBK_BDD_FEATURE").is_some() {
        return true;
    }
    feature.path.as_ref().is_some_and(|path| {
        path.parent()
            .and_then(|folder| folder.file_name())
            .and_then(|name| name.to_str())
            .is_some_and(|name| ACCEPTED_FEATURES.contains(&name))
    })
}
