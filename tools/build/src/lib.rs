// SPDX-License-Identifier: GPL-2.0-only
//! Repository-owned commands for building, testing, and distributing Oblikovati.

mod archive;
mod bootstrap;
mod cargo_tree;
mod dependencies;
mod distribution;
mod docs;
mod driver;
mod linux;
mod macos;
mod native_package;
mod packaging;
mod platform;
mod process;
mod quality;
mod release_config;
mod script_docs;
mod signing;
mod tree;
mod version;
mod windows;

#[cfg(test)]
#[path = "../tests/release_fixture.rs"]
mod release_fixture;

use process::{BuildResult, Runner, SystemRunner};
use std::path::Path;

/// Run an explicit repository task. Invalid commands return an error without side effects.
///
/// ```
/// assert!(oblikovati_build::run(&["unknown".into()], std::path::Path::new(".")).is_err());
/// ```
pub fn run(arguments: &[String], root: &Path) -> Result<(), String> {
    dispatch(
        &arguments.iter().map(String::as_str).collect::<Vec<_>>(),
        root,
        &SystemRunner,
    )
}

fn dispatch(arguments: &[&str], root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    match arguments {
        [task @ ("build" | "test" | "lint")] => quality::run(root, runner, task),
        ["matrix"] => print_matrix(),
        ["version", channel] => print_version(root, runner, channel),
        [task @ ("native" | "gate" | "docs" | "tools" | "script-docs" | "script-docs-check")] => {
            local_task(task, root, runner)
        }
        _ => release_task(arguments, root, runner),
    }
}
fn print_matrix() -> BuildResult<()> {
    println!("{}", serde_json::json!({"include": platform::PLATFORMS}));
    Ok(())
}
fn local_task(task: &str, root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    match task {
        "native" => native(root, runner),
        "gate" => quality::gate(root, runner),
        "docs" => docs::check(root),
        "tools" => bootstrap::install(root, runner),
        "script-docs" => script_docs::write(root, runner),
        "script-docs-check" => script_docs::check(root, runner),
        _ => Err(format!(
            "local task {task:?}: expected a repository validation or generation command"
        )),
    }
}
fn release_task(arguments: &[&str], root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    match arguments {
        ["preflight", mode @ ("build" | "publish")] => {
            driver::preflight(root, runner, *mode == "publish").map(|_| ())
        }
        ["release-request", channel] => driver::release_request(root, runner, channel),
        ["package", target, version] => driver::package(root, runner, target, version, false),
        ["package", target, version, "--signed"] => {
            driver::package(root, runner, target, version, true)
        }
        ["mac-bundle", version] => driver::mac_bundle(root, runner, version, false),
        ["mac-bundle", version, "--signed"] => driver::mac_bundle(root, runner, version, true),
        _ => distribution_task(arguments, root, runner),
    }
}
fn distribution_task(arguments: &[&str], root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    match arguments {
        ["collect", version, input, output] => {
            driver::collect(root, runner, version, Path::new(input), Path::new(output))
        }
        ["verify-release", version, output] => {
            driver::verify_release(root, runner, version, Path::new(output))
        }
        _ => Err(format!(
            "arguments {arguments:?}: expected build|test|lint|native|gate|docs|script-docs|script-docs-check|tools|matrix|version CHANNEL|preflight build/publish|release-request CHANNEL|package TARGET VERSION [--signed]|mac-bundle VERSION [--signed]|collect VERSION INPUT OUTPUT|verify-release VERSION OUTPUT"
        )),
    }
}

fn print_version(root: &Path, runner: &dyn Runner, channel: &str) -> BuildResult<()> {
    let config = release_config::load(root)?;
    println!(
        "{}",
        version::compute(
            root,
            runner,
            config.product.major,
            &config.api.version,
            channel,
            chrono::Utc::now()
        )?
    );
    Ok(())
}

fn native(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    for task in ["lint", "build", "test"] {
        quality::run(root, runner, task)?;
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/lib.rs"]
mod tests;
