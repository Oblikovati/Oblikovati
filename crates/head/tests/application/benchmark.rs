// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{
    Parser as _, World as _, WriterExt as _,
    writer::{self, Stats as _},
};
use futures::StreamExt as _;
use std::{fmt, fs, path::PathBuf};
mod binding_inventory;
mod capture_only;
#[cfg(target_os = "linux")]
mod cube_os_entry;
mod diagnostics;
mod documents;
mod gap_runner;
#[cfg(target_os = "linux")]
#[path = "main-window-frame/steps.rs"]
mod main_window_frame;
#[cfg(target_os = "linux")]
mod main_window_frame_accessibility;
#[cfg(target_os = "linux")]
mod main_window_frame_native;
#[cfg(target_os = "linux")]
#[path = "main-window-frame/resize.rs"]
mod main_window_frame_resize;
mod quick_access_icons;
mod refinements;
mod report_destination;
mod scenario_revision;
mod scenario_source;
mod solid_visibility;
mod viewport_background;
mod world;
#[cfg(target_os = "linux")]
mod x11_input;
use world::ApplicationWorld;
pub(super) fn window_geometry(title: &str) -> Option<([f32; 2], [f32; 2])> {
    diagnostics::window_geometry(title)
}
/// Read GUI window bounds; example: `window_occlusions()` before a viewport pixel check.
pub(super) fn window_occlusions() -> Vec<[f32; 4]> {
    diagnostics::window_occlusions()
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
        .map(|run| run.center())
        .or_else(|| quick_access_icons::standard_icon_center(label))
        .unwrap_or_else(|| {
            panic!(
                "label {label:?}: expected visible control in {:?}",
                frame.runs
            )
        });
    diagnostics::pointer_pixels(run)
}
fn tab_label_run<'a>(
    frame: &'a diagnostics::RenderedFrame,
    label: &str,
) -> &'a diagnostics::RenderedRun {
    let (position, size) =
        diagnostics::window_geometry("##open-document-tabs").expect("document tab strip");
    frame
        .runs
        .iter()
        .find(|run| {
            run.text.starts_with(label)
                && (0..2).all(|axis| {
                    run.center()[axis] >= position[axis]
                        && run.center()[axis] < position[axis] + size[axis]
                })
        })
        .unwrap_or_else(|| panic!("tab {label:?}: expected a label in the tab strip"))
}
pub(super) fn tab_label_center(label: &str) -> [i32; 2] {
    let frame = diagnostics::rendered_frame();
    diagnostics::pointer_pixels(tab_label_run(&frame, label).center())
}
pub(super) fn tab_close_mark(label: &str) -> [i32; 2] {
    let frame = diagnostics::rendered_frame();
    let run = tab_label_run(&frame, label);
    let marks: Vec<_> = frame
        .marks
        .iter()
        .filter(|mark| {
            mark.list == run.list
                && mark.position[0] > run.maximum[0]
                && mark.position[0] < run.maximum[0] + 30.
                && mark.position[1] >= run.minimum[1] - 2.
                && mark.position[1] <= run.maximum[1] + 2.
        })
        .collect();
    assert!(
        !marks.is_empty(),
        "tab {label:?}: expected its visible close mark"
    );
    let point = std::array::from_fn(|axis| {
        marks.iter().map(|mark| mark.position[axis]).sum::<f32>() / marks.len() as f32
    });
    diagnostics::pointer_pixels(point)
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
#[path = "application-closing/steps.rs"]
mod application_closing;
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
#[cfg(test)]
mod offset_plane_native;
mod offset_plane_overlay;
#[cfg(target_os = "linux")]
mod part_marking_menu;
#[path = "project-dialog/steps.rs"]
mod project_dialog;
#[path = "../solid-model/projection-fixtures/steps.rs"]
mod projection_fixtures;
#[path = "quick-access-toolbar/steps.rs"]
mod quick_access_toolbar;
pub(crate) mod rectangle_visibility;
#[path = "ribbon-appearance/steps.rs"]
mod ribbon_appearance;
#[path = "ribbon-command-availability/steps.rs"]
mod ribbon_command_availability;
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
#[path = "workspace-layout/wireframe_ribbon.rs"]
mod wireframe_ribbon;
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
            let _reports = report_destination::reserve(&world::reports_folder())
                .unwrap_or_else(|error| panic!("{error}"));
            run_scenarios(feature_root(area), |_| true);
        })
        .unwrap()
        .join()
        .unwrap();
}
fn scenario_capture_namespace(
    feature: &cucumber::gherkin::Feature,
    scenario: &cucumber::gherkin::Scenario,
) -> String {
    use sha2::{Digest, Sha256};
    let path = fs::canonicalize(feature.path.as_ref().expect("scenario feature path"))
        .expect("scenario feature path must exist");
    let root = fs::canonicalize(scenario_root()).expect("scenario root must exist");
    let relative = path
        .strip_prefix(root)
        .expect("scenario feature must be inside scenario root");
    let identity = format!(
        "{}:{}:{}",
        relative.display(),
        scenario.position.line,
        scenario.name
    );
    format!("{:x}", Sha256::digest(identity.as_bytes()))
}
fn capture_name(
    feature: &cucumber::gherkin::Feature,
    scenario: &cucumber::gherkin::Scenario,
) -> String {
    if scenario.examples.is_empty() {
        return scenario.name.clone();
    }
    use sha2::{Digest, Sha256};
    let path = fs::canonicalize(feature.path.as_ref().expect("outline feature path"))
        .expect("outline feature path must exist");
    let root = fs::canonicalize(scenario_root()).expect("scenario root must exist");
    let relative = path
        .strip_prefix(root)
        .expect("outline feature must be inside scenario root")
        .components()
        .map(|part| part.as_os_str().to_str().expect("UTF-8 feature path"))
        .collect::<Vec<_>>()
        .join("/");
    let digest = Sha256::digest(relative.as_bytes());
    format!("example-{digest:x}-{}", scenario.position.line)
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
        let result = world::reports_folder().join("gap-report.json");
        scenario_revision::require_pinned(&scenario_root(), &[&result]);
        scenario_revision::require_unchanged(&before, &after, source_changed, &[&result]);
        return;
    }
    let reports = world::reports_folder();
    fs::create_dir_all(&reports).unwrap();
    let json = report_destination::create_file(&reports.join("cucumber.json")).unwrap();
    let junit = report_destination::create_file(&reports.join("junit.xml")).unwrap();
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
            .before(|feature, _, scenario, world| {
                let namespace = scenario_capture_namespace(feature, scenario);
                Box::pin(async move { world.set_capture_namespace(namespace) })
            })
            .after(|feature, _, scenario, _, world| {
                let name = capture_name(feature, scenario);
                Box::pin(async move {
                    if let Some(world) = world {
                        world.save_final_capture(&name);
                        world.close();
                    }
                })
            })
            .filter_run(root.clone(), move |feature, rule, scenario| {
                if let Some(reason) = capture_only::reason(feature, rule, scenario) {
                    println!("Not executed ({reason}): {}", scenario.name);
                    return false;
                }
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
    let cucumber = reports.join("cucumber.json");
    let junit = reports.join("junit.xml");
    let results = [cucumber.as_path(), junit.as_path()];
    scenario_revision::require_pinned(&scenario_root(), &results);
    scenario_revision::require_unchanged(&before, &after, source_changed, &results);
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
    std::env::var("OBK_BDD_SCENARIO").map_or(true, |wanted| scenario_matches(name, &wanted))
}
fn scenario_matches(name: &str, wanted: &str) -> bool {
    let mut remainder = name;
    for part in wanted.split('<') {
        let literal = part.split_once('>').map_or(part, |(_, suffix)| suffix);
        let Some((_, after)) = remainder.split_once(literal) else {
            return false;
        };
        remainder = after;
    }
    true
}
#[test]
fn scenario_filter_matches_expanded_outline_names() {
    assert!(scenario_matches("Value one ends", "Value <x> ends"));
    assert!(scenario_matches("Value two ends", "Value <x> ends"));
    assert!(!scenario_matches("Value one stops", "Value <x> ends"));
    assert!(scenario_matches("regular scenario", "regular"));
}
fn scenario_root() -> PathBuf {
    scenario_root_from_override(std::env::var_os("OBK_BDD_ROOT"))
}
fn scenario_root_from_override(explicit: Option<std::ffi::OsString>) -> PathBuf {
    scenario_source::root(explicit)
}
fn verify_scenario_source(root: &std::path::Path) -> Result<(), String> {
    let strict = std::env::var_os("OBK_BDD_REQUIRE_PINNED").is_some();
    scenario_source::validate_gate_filters(strict, &scenario_source::configured_filters())?;
    scenario_source::verify_required(root, std::env::var_os("OBK_BDD_ROOT").is_some(), strict)
}

fn feature_root(area: &str) -> PathBuf {
    let root = scenario_root();
    verify_scenario_source(&root).unwrap_or_else(|error| panic!("{error}"));
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

#[test]
fn default_scenario_source_is_the_pinned_checkout() {
    let root = scenario_root_from_override(None);
    assert!(
        root.join(".git").is_dir(),
        "scenario root {root:?}: expected the pinned Git checkout"
    );
    scenario_source::verify_required(&root, false, false).unwrap();
}
