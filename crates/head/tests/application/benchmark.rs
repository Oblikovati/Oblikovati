// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{
    Parser as _, World as _, WriterExt as _,
    writer::{self, Stats as _},
};
use futures::StreamExt as _;
use std::{fmt, fs, path::PathBuf};
mod binding_inventory;
mod diagnostics;
mod documents;
mod gap_runner;
mod refinements;
mod scenario_revision;
mod solid_visibility;
mod viewport_background;
mod world;
use world::ApplicationWorld;
pub(super) fn window_geometry(title: &str) -> Option<([f32; 2], [f32; 2])> {
    diagnostics::window_geometry(title)
}
pub(super) fn pointer_pixels(position: [f32; 2]) -> [i32; 2] {
    diagnostics::pointer_pixels(position)
}
pub(super) fn rendered_label_center(label: &str) -> [i32; 2] {
    let frame = diagnostics::rendered_frame();
    let wanted: String = label.chars().filter(|ch| !ch.is_whitespace()).collect();
    let run = frame
        .runs
        .iter()
        .filter(|run| {
            run.text
                .chars()
                .filter(|ch| !ch.is_whitespace())
                .eq(wanted.chars())
        })
        .min_by(|a, b| a.minimum[1].total_cmp(&b.minimum[1]))
        .unwrap_or_else(|| {
            panic!(
                "label {label:?}: expected visible control in {:?}",
                frame.runs
            )
        });
    diagnostics::pointer_pixels(run.center())
}
pub(super) fn rendered_label_in_rect(label: &str, min: [f32; 2], max: [f32; 2]) -> [i32; 2] {
    let frame = diagnostics::rendered_frame();
    let run = frame
        .runs
        .iter()
        .find(|run| {
            run.text == label
                && (0..2)
                    .all(|axis| run.center()[axis] >= min[axis] && run.center()[axis] < max[axis])
        })
        .unwrap_or_else(|| {
            panic!(
                "label {label:?}: expected in rectangle {min:?}..{max:?}; found {:?}",
                frame.runs
            )
        });
    diagnostics::pointer_pixels(run.center())
}
pub(super) fn rendered_label_start(label: &str) -> [i32; 2] {
    let wanted: String = label.chars().filter(|ch| !ch.is_whitespace()).collect();
    let frame = diagnostics::rendered_frame();
    let run = frame
        .runs
        .iter()
        .find(|run| {
            run.text
                .chars()
                .filter(|ch| !ch.is_whitespace())
                .collect::<String>()
                .starts_with(&wanted)
        })
        .unwrap_or_else(|| {
            panic!(
                "label {label:?}: expected visible control in {:?}",
                frame.runs
            )
        });
    diagnostics::pointer_pixels([run.minimum[0] + 5., run.center()[1]])
}
pub(super) fn rendered_sketch_row_center() -> [i32; 2] {
    let frame = diagnostics::rendered_frame();
    let rows: Vec<_> = frame
        .runs
        .iter()
        .filter(|run| run.text.contains("[DOF"))
        .collect();
    let [row] = rows.as_slice() else {
        panic!("sketch rows {rows:?}: expected exactly one visible sketch")
    };
    diagnostics::pointer_pixels(row.center())
}
#[path = "body-properties/steps.rs"]
mod body_properties;
#[path = "document-creation/steps.rs"]
mod document_creation;
#[path = "document-opening/steps.rs"]
mod document_opening;
#[path = "document-saving/steps.rs"]
mod document_saving;
#[path = "document-tabs/steps.rs"]
mod document_tabs;
mod extrude_resume_native;
#[path = "file-menu/steps.rs"]
mod file_menu;
#[path = "home-recent-documents/steps.rs"]
mod home_recent_documents;
#[path = "model-browser/steps.rs"]
mod model_browser;
#[path = "project-dialog/steps.rs"]
mod project_dialog;
#[path = "quick-access-toolbar/steps.rs"]
mod quick_access_toolbar;
mod rectangle_visibility;
#[path = "ribbon-appearance/steps.rs"]
mod ribbon_appearance;
#[path = "ribbon-customization/steps.rs"]
mod ribbon_customization;
#[path = "ribbon-display/steps.rs"]
mod ribbon_display;
#[path = "ribbon-docking/steps.rs"]
mod ribbon_docking;
#[path = "ribbon-keyboard-navigation/steps.rs"]
mod ribbon_keyboard_navigation;
#[path = "ribbon-panel-expansion/steps.rs"]
mod ribbon_panel_expansion;
#[path = "ribbon-panel-visibility/steps.rs"]
mod ribbon_panel_visibility;
#[path = "../sketch/benchmark.rs"]
mod sketch;
#[path = "../solid-model/extrusion/steps.rs"]
mod solid_extrusion;
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
        .spawn(|| {
            let configured = std::env::var_os("OBK_BDD_AREA");
            let setting = configured
                .as_deref()
                .map(|value| value.to_str().expect("BDD area: expected UTF-8"));
            let area = selected_area(setting).unwrap();
            run_scenarios(feature_root(area), |_| true);
        })
        .unwrap()
        .join()
        .unwrap();
}
fn selected_area(setting: Option<&str>) -> Result<&str, String> {
    match setting {
        None | Some("application") => Ok("application"),
        Some("solid-model") => Ok("solid-model"),
        Some(other) => Err(format!(
            "BDD area {other:?}: expected application or solid-model"
        )),
    }
}
#[test]
fn native_area_selection_keeps_application_default_and_rejects_invalid_values() {
    assert_eq!(selected_area(None).unwrap(), "application");
    assert_eq!(selected_area(Some("solid-model")).unwrap(), "solid-model");
    assert!(
        selected_area(Some("sketch"))
            .unwrap_err()
            .contains("expected application or solid-model")
    );
}
/// Run one area's features. `OBK_BDD_SCENARIO` narrows a run to matching scenario names.
fn run_scenarios(root: PathBuf, keep_feature: fn(&cucumber::gherkin::Feature) -> bool) {
    let before = scenario_revision::inspect(&root);
    let source = gap_runner::read_sources(&root);
    if std::env::var_os("OBK_BDD_DIAGNOSTIC").is_some() {
        gap_runner::run(&root, keep_feature);
        let after = scenario_revision::inspect(&root);
        let source_changed = source != gap_runner::read_sources(&root);
        scenario_revision::save(
            &world::reports_folder().join("source-revision.json"),
            &before,
            &after,
            source_changed,
        );
        scenario_revision::require_unchanged(
            &before,
            &after,
            source_changed,
            &[&world::reports_folder().join("gap-report.json")],
        );
        return;
    }
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
            .filter_run(root.clone(), move |feature, _, scenario| {
                keep_feature(feature) && scenario_selected(&scenario.name)
            }),
    );
    let after = scenario_revision::inspect(&root);
    let source_changed = source != gap_runner::read_sources(&root);
    scenario_revision::save(
        &reports.join("source-revision.json"),
        &before,
        &after,
        source_changed,
    );
    scenario_revision::require_unchanged(
        &before,
        &after,
        source_changed,
        &[&reports.join("cucumber.json"), &reports.join("junit.xml")],
    );
    if result.execution_has_failed() {
        report_parser_errors(&root);
    }
    assert!(
        !result.execution_has_failed(),
        "scenario benchmark: expected no failed, undefined, or skipped steps"
    );
    assert!(
        result.passed_steps() > 0,
        "scenario benchmark: expected at least one executed scenario"
    );
}
fn report_parser_errors(root: &std::path::Path) {
    let parsed = cucumber::parser::Basic::default().parse(root, Default::default());
    let errors = futures::executor::block_on(parsed.collect::<Vec<_>>());
    for error in errors.into_iter().filter_map(Result::err) {
        eprintln!("{}", parser_diagnostic(&error));
    }
}
fn parser_diagnostic(error: &cucumber::parser::Error) -> String {
    format!("scenario source error: {error:#?}")
}
#[test]
fn parser_diagnostic_includes_the_underlying_read_error() {
    let cause = std::io::Error::new(std::io::ErrorKind::NotFound, "fixture absent");
    let error = cucumber::parser::Error::from(cucumber::gherkin::ParseFileError::Reading {
        path: "broken.feature".into(),
        source: cause,
    });
    let message = parser_diagnostic(&error);
    assert!(message.contains("broken.feature"));
    assert!(message.contains("fixture absent"));
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
