// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{
    World as _, WriterExt as _,
    writer::{self, Stats as _},
};
use std::{fmt, fs, path::PathBuf};
mod diagnostics;
mod documents;
mod refinements;
mod world;
use world::ApplicationWorld;
#[path = "body-properties/steps.rs"]
mod body_properties;
#[path = "document-creation/steps.rs"]
mod document_creation;
#[path = "document-saving/steps.rs"]
mod document_saving;
#[path = "document-tabs/steps.rs"]
mod document_tabs;
#[path = "file-menu/steps.rs"]
mod file_menu;
#[path = "home-recent-documents/steps.rs"]
mod home_recent_documents;
#[path = "project-dialog/steps.rs"]
mod project_dialog;
#[path = "quick-access-toolbar/steps.rs"]
mod quick_access_toolbar;
#[path = "ribbon-display/steps.rs"]
mod ribbon_display;
#[path = "ribbon-docking/steps.rs"]
mod ribbon_docking;
#[path = "ribbon-panel-visibility/steps.rs"]
mod ribbon_panel_visibility;
#[path = "../sketch/benchmark.rs"]
mod sketch;
#[path = "viewport-navigation-wheel/steps.rs"]
mod viewport_navigation_wheel;
#[path = "viewport-orbit/steps.rs"]
mod viewport_orbit;
#[path = "viewport-orientation/steps.rs"]
mod viewport_orientation;
#[path = "viewport-pan/steps.rs"]
mod viewport_pan;
#[path = "viewport-zoom/steps.rs"]
mod viewport_zoom;
#[path = "workspace-layout/steps.rs"]
mod workspace_layout;

/// Unoptimized Cucumber futures hold every step on the stack, so the suite outgrows a test
/// thread's default as scenarios arrive. The window and its GUI context live on this one thread.
const SCENARIO_STACK_BYTES: usize = 256 * 1024 * 1024;
#[test]
fn application_scenarios_use_the_native_window() {
    std::thread::Builder::new()
        .name("application-scenarios".into())
        .stack_size(SCENARIO_STACK_BYTES)
        .spawn(|| run_scenarios(feature_root("application"), |_| true))
        .unwrap()
        .join()
        .unwrap();
}
/// Run one area's features. `OBK_BDD_SCENARIO` narrows a run to matching scenario names.
fn run_scenarios(root: PathBuf, keep_feature: fn(&cucumber::gherkin::Feature) -> bool) {
    let reports = world::reports_folder();
    fs::create_dir_all(&reports).unwrap();
    let json = fs::File::create(reports.join("cucumber.json")).unwrap();
    let junit = fs::File::create(reports.join("junit.xml")).unwrap();
    let writer = writer::Basic::stdout()
        .summarized()
        .tee::<ApplicationWorld, _>(writer::Json::for_tee(json))
        .tee::<ApplicationWorld, _>(writer::JUnit::for_tee(junit, 1))
        .normalized();
    let result = futures::executor::block_on(
        ApplicationWorld::cucumber()
            .max_concurrent_scenarios(1)
            .with_writer(writer)
            // After `with_writer`, which resets the CLI: libtest's arguments are not Cucumber's.
            .with_default_cli()
            .fail_on_skipped_with(|_, _, _| true)
            .after(|_, _, scenario, _, world| {
                let name = scenario.name.clone();
                Box::pin(async move {
                    if let Some(world) = world {
                        world.save_capture(&name);
                        world.close();
                    }
                })
            })
            .filter_run(root, move |feature, _, scenario| {
                keep_feature(feature) && scenario_selected(&scenario.name)
            }),
    );
    assert!(
        !result.execution_has_failed(),
        "scenario benchmark: expected no failed, undefined, or skipped steps"
    );
    assert!(
        result.passed_steps() > 0,
        "scenario benchmark: expected at least one executed scenario"
    );
}
fn scenario_selected(name: &str) -> bool {
    std::env::var("OBK_BDD_SCENARIO").map_or(true, |wanted| name.contains(&wanted))
}
fn feature_root(area: &str) -> PathBuf {
    let root = std::env::var_os("OBK_BDD_ROOT").map_or_else(
        || {
            PathBuf::from(env!("CARGO_MANIFEST_DIR"))
                .join("../../test-utilities/benchmarks/MCAD-Scenarios")
        },
        PathBuf::from,
    );
    assert!(
        root.is_absolute(),
        "scenario root {root:?}: expected an absolute path"
    );
    let features = root.join("features").join(area);
    assert!(
        features.is_dir(),
        "scenario root {root:?}: expected features/{area}"
    );
    let Some(feature) = std::env::var_os("OBK_BDD_FEATURE") else {
        return features;
    };
    let feature = feature.to_str().expect("BDD feature: expected UTF-8");
    assert!(
        !feature.is_empty()
            && feature
                .bytes()
                .all(|byte| byte.is_ascii_lowercase() || byte == b'-'),
        "BDD feature {feature:?}: expected a folder name"
    );
    let selected = features.join(feature);
    assert!(
        selected.is_dir(),
        "BDD feature {feature:?}: expected an existing feature folder"
    );
    selected
}
