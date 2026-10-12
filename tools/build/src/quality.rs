// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    cargo_tree::{Metadata, inventory, requires_display},
    dependencies, docs,
    process::{BuildResult, Runner, execute},
    tree::{read_text, write_text},
};
use serde::Deserialize;
use std::{collections::BTreeMap, path::Path};

#[derive(Deserialize)]
struct CoverageReport {
    data: Vec<CoverageUnit>,
}
#[derive(Deserialize)]
struct CoverageUnit {
    files: Vec<CoverageFile>,
}
#[derive(Deserialize)]
struct CoverageFile {
    filename: String,
    summary: CoverageSummary,
}
#[derive(Deserialize)]
struct CoverageSummary {
    lines: LineCount,
}
#[derive(Deserialize)]
struct LineCount {
    count: u64,
    covered: u64,
}

pub(crate) fn workspace_command(
    runner: &dyn Runner,
    workspace: &Path,
    task: &str,
) -> BuildResult<()> {
    match task {
        "build" => execute(
            runner,
            workspace,
            &[
                "cargo",
                "build",
                "--locked",
                "--workspace",
                "--all-targets",
                "--all-features",
            ],
        ),
        "test" => {
            execute(
                runner,
                workspace,
                &[
                    "cargo",
                    "test",
                    "--locked",
                    "--workspace",
                    "--all-targets",
                    "--all-features",
                ],
            )?;
            doctests(runner, workspace)
        }
        "lint" => {
            execute(
                runner,
                workspace,
                &["cargo", "fmt", "--all", "--", "--check"],
            )?;
            execute(
                runner,
                workspace,
                &[
                    "cargo",
                    "clippy",
                    "--locked",
                    "--workspace",
                    "--all-targets",
                    "--all-features",
                    "--",
                    "-D",
                    "warnings",
                ],
            )
        }
        _ => Err(format!(
            "workspace task {task:?}: expected build, test, or lint"
        )),
    }
}

fn doctests(runner: &dyn Runner, workspace: &Path) -> BuildResult<()> {
    execute(
        runner,
        workspace,
        &[
            "cargo",
            "test",
            "--locked",
            "--workspace",
            "--all-features",
            "--doc",
        ],
    )
}

pub(crate) fn run(root: &Path, runner: &dyn Runner, task: &str) -> BuildResult<()> {
    let workspaces = inventory(root, runner)?;
    check_display(
        &workspaces,
        cfg!(target_os = "linux"),
        std::env::var_os("DISPLAY").is_some() || std::env::var_os("WAYLAND_DISPLAY").is_some(),
    )?;
    for workspace in &workspaces {
        workspace_command(runner, &workspace.workspace_root, task)?;
    }
    Ok(())
}

fn check_display(workspaces: &[Metadata], linux: bool, present: bool) -> BuildResult<()> {
    for package in workspaces.iter().flat_map(|workspace| &workspace.packages) {
        if requires_display(package)? && linux && !present {
            return Err(format!(
                "{}: required DISPLAY or WAYLAND_DISPLAY is missing; run the gate under a real display or Xvfb",
                package.name
            ));
        }
    }
    Ok(())
}

pub(crate) fn gate(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    dependencies::check_pins(root)?;
    let workspaces = inventory(root, runner)?;
    check_display(
        &workspaces,
        cfg!(target_os = "linux"),
        std::env::var_os("DISPLAY").is_some() || std::env::var_os("WAYLAND_DISPLAY").is_some(),
    )?;
    gate_tool_versions(root, runner)?;
    for (index, workspace) in workspaces.iter().enumerate() {
        gate_workspace(root, runner, workspace, index)?;
    }
    gate_repository_checks(root, runner)?;
    println!(
        "Gate passed for {} discovered Cargo workspace(s). Runtime evidence covers only the crates present in this tree.",
        workspaces.len()
    );
    Ok(())
}

fn gate_tool_versions(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    require_tool(runner, root, "rustc", &["--version"], "rustc 1.99.0 ")?;
    require_tool(
        runner,
        root,
        "cargo",
        &["llvm-cov", "--version"],
        "cargo-llvm-cov 0.9.1",
    )?;
    Ok(())
}

fn gate_repository_checks(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    execute(
        runner,
        root,
        &["npm", "exec", "--offline", "--", "markdownlint-cli2"],
    )?;
    docs::check(root)?;
    execute(runner, root, &["actionlint", "-shellcheck=", "-pyflakes="])?;
    gate_duplication_check(root, runner)?;
    execute(runner, root, &["git", "diff", "--check"])?;
    Ok(())
}

fn gate_duplication_check(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    execute(
        runner,
        root,
        &[
            "npm",
            "exec",
            "--offline",
            "--",
            "jscpd",
            "--config",
            ".jscpd.json",
            ".",
        ],
    )
}

fn require_tool(
    runner: &dyn Runner,
    root: &Path,
    program: &str,
    arguments: &[&str],
    version: &str,
) -> BuildResult<()> {
    let actual = runner.output(root, program, arguments)?;
    if !actual.trim().starts_with(version) {
        return Err(format!("tool version {actual:?}: expected {version}"));
    }
    Ok(())
}

fn gate_workspace(
    root: &Path,
    runner: &dyn Runner,
    workspace: &Metadata,
    index: usize,
) -> BuildResult<()> {
    let directory = &workspace.workspace_root;
    workspace_command(runner, directory, "lint")?;
    workspace_command(runner, directory, "build")?;
    let report = root.join(format!("target/quality/coverage-{index}.json"));
    write_text(&report, "")?;
    execute(
        runner,
        directory,
        &[
            "cargo",
            "llvm-cov",
            "--locked",
            "--workspace",
            "--all-targets",
            "--all-features",
            "--json",
            "--output-path",
            &report.to_string_lossy(),
        ],
    )?;
    check_coverage(&read_text(&report)?, workspace)?;
    doctests(runner, directory)
}

fn check_coverage(json: &str, workspace: &Metadata) -> BuildResult<()> {
    let report: CoverageReport =
        serde_json::from_str(json).map_err(|error| format!("coverage JSON: {error}"))?;
    let mut totals = BTreeMap::<String, (u64, u64)>::new();
    for file in report.data.iter().flat_map(|unit| &unit.files) {
        add_coverage(file, workspace, &mut totals);
    }
    for package in &workspace.packages {
        let (covered, count) = totals.get(&package.name).copied().unwrap_or_default();
        if count == 0 || (covered as f64 / count as f64) <= 0.80 {
            return Err(format!(
                "{} line coverage {covered}/{count}: expected strictly greater than 80%",
                package.name
            ));
        }
        println!(
            "{} line coverage: {:.2}% ({covered}/{count})",
            package.name,
            100.0 * covered as f64 / count as f64
        );
    }
    Ok(())
}

fn add_coverage(
    file: &CoverageFile,
    workspace: &Metadata,
    totals: &mut BTreeMap<String, (u64, u64)>,
) {
    for package in &workspace.packages {
        let directory = package.manifest_path.parent().unwrap_or(Path::new("."));
        let path = Path::new(&file.filename);
        if !path.starts_with(directory) || path.starts_with(directory.join("tests")) {
            continue;
        }
        let total = totals.entry(package.name.clone()).or_default();
        total.0 += file.summary.lines.covered;
        total.1 += file.summary.lines.count;
    }
}

#[cfg(test)]
#[path = "../tests/quality.rs"]
mod tests;
