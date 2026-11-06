// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch` on the same native window and owner as the application scenarios.
use super::*;
#[path = "fixtures.rs"]
pub(super) mod fixtures;
#[path = "line/steps.rs"]
mod line;
#[path = "point/steps.rs"]
mod point;
#[path = "precise-input/steps.rs"]
mod precise_input;

/// Features the gate requires; each is one finished folder under `features/sketch`.
const ACCEPTED_FEATURES: &[&str] = &["line", "point", "precise-input"];
#[test]
fn sketch_scenarios_use_the_native_window() {
    std::thread::Builder::new()
        .name("sketch-scenarios".into())
        .stack_size(SCENARIO_STACK_BYTES)
        .spawn(|| run_scenarios(feature_root("sketch"), accepted_sketch_feature))
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
