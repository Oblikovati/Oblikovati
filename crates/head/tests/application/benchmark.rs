// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{
    World as _, WriterExt as _,
    writer::{self, Stats as _},
};
use std::{fmt, fs, path::PathBuf};
mod diagnostics;
mod world;
use world::ApplicationWorld;
#[path = "document-creation/steps.rs"]
mod document_creation;
#[path = "file-menu/steps.rs"]
mod file_menu;
#[path = "workspace-layout/steps.rs"]
mod workspace_layout;

#[test]
fn application_scenarios_use_the_native_window() {
    let root = feature_root();
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
            .run(root),
    );
    assert!(
        !result.execution_has_failed(),
        "application benchmark: expected no failed, undefined, or skipped steps"
    );
    assert!(
        result.passed_steps() > 0,
        "application benchmark: expected at least one executed scenario"
    );
}
fn feature_root() -> PathBuf {
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
    let application = root.join("features/application");
    assert!(
        application.is_dir(),
        "scenario root {root:?}: expected features/application"
    );
    let Some(feature) = std::env::var_os("OBK_BDD_FEATURE") else {
        return application;
    };
    let feature = feature.to_str().expect("BDD feature: expected UTF-8");
    assert!(
        !feature.is_empty()
            && feature
                .bytes()
                .all(|byte| byte.is_ascii_lowercase() || byte == b'-'),
        "BDD feature {feature:?}: expected a folder name"
    );
    let selected = application.join(feature);
    assert!(
        selected.is_dir(),
        "BDD feature {feature:?}: expected an existing feature folder"
    );
    selected
}
