// SPDX-License-Identifier: GPL-2.0-only
//! Scenario declarations, not acceptance results or expanded example counts.
use super::{ScenarioSource, describe};
use crate::process::{BuildResult, Runner};
use gherkin::{Background, Feature, Scenario, Step};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::{collections::BTreeSet, path::Path};

#[derive(Debug, Serialize, PartialEq, Eq)]
pub(super) struct ScenarioLedger {
    source_revision: String,
    scenarios: Vec<ScenarioRow>,
}
#[derive(Debug, Serialize, PartialEq, Eq)]
struct ScenarioRow {
    id: String,
    path: String,
    area: String,
    feature: String,
    source_sha256: String,
    line: usize,
    scenario: String,
    rule: Option<String>,
    background_steps: Vec<StepSignature>,
    scenario_steps: Vec<StepSignature>,
    outline: bool,
    milestone: Option<String>,
    owner: Option<String>,
    state: &'static str,
    source_error: Option<String>,
    last_receipt: Option<String>,
}
pub(super) fn verify_checkout(root: &Path, revision: &str, runner: &dyn Runner) -> BuildResult<()> {
    if !root.join(".git").exists() {
        return Err(format!(
            "scenario root {root:?}: expected its own Git checkout"
        ));
    }
    let actual = runner.output(root, "git", &["rev-parse", "HEAD"])?;
    if actual.trim() != revision {
        return Err(format!(
            "scenario revision {:?}: expected checkout commit {revision}",
            actual.trim()
        ));
    }
    verify_clean(root, runner)
}
fn verify_clean(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    let dirty = runner.output(
        root,
        "git",
        &["status", "--porcelain", "--untracked-files=all"],
    )?;
    let ignored = runner.output(
        root,
        "git",
        &[
            "ls-files",
            "--others",
            "--ignored",
            "--exclude-standard",
            "--",
            "features",
        ],
    )?;
    if !dirty.trim().is_empty() || !ignored.trim().is_empty() {
        return Err(format!(
            "scenario root {root:?}: expected clean source, got {dirty:?} and ignored files {ignored:?}"
        ));
    }
    Ok(())
}

pub(super) fn inventory(
    root: &Path,
    revision: &str,
    source: &mut impl ScenarioSource,
) -> BuildResult<ScenarioLedger> {
    if revision.len() != 40 || !revision.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err(format!(
            "scenario revision {revision:?}: expected a 40-character Git commit"
        ));
    }
    let mut paths = source.feature_paths(root)?;
    paths.sort();
    let rows = collect_rows(root, paths, source)?;
    if rows.is_empty() {
        return Err(format!(
            "scenario root {root:?}: expected scenario declarations"
        ));
    }
    Ok(ScenarioLedger {
        source_revision: revision.to_owned(),
        scenarios: rows,
    })
}
fn collect_rows(
    root: &Path,
    paths: Vec<std::path::PathBuf>,
    source: &mut impl ScenarioSource,
) -> BuildResult<Vec<ScenarioRow>> {
    let mut rows = Vec::new();
    let mut errors = Vec::new();
    for path in paths {
        match read_feature_rows(root, &path, source) {
            Ok(feature_rows) => rows.extend(feature_rows),
            Err(error) => errors.push(error),
        }
    }
    if !errors.is_empty() {
        return Err(errors.join("\n"));
    }
    Ok(rows)
}

fn read_feature_rows(
    root: &Path,
    path: &Path,
    source: &mut impl ScenarioSource,
) -> BuildResult<Vec<ScenarioRow>> {
    let feature_source = describe(root, path, source)?;
    let bytes = source.read(path)?;
    if format!("{:x}", Sha256::digest(&bytes)) != feature_source.sha256 {
        return Err(format!(
            "scenario source {path:?}: expected unchanged bytes during inventory"
        ));
    }
    let relative = Path::new(&feature_source.path);
    let (declarations, error) = match parse_feature(relative, &bytes) {
        Ok(feature) => (parsed_declarations(&feature), None),
        Err(error) => (fallback_declarations(relative, &bytes)?, Some(error)),
    };
    unique_rows(path, &feature_source, declarations, error.as_deref())
}
fn unique_rows(
    path: &Path,
    feature: &super::ScenarioFeatureSource,
    declarations: Vec<Declaration>,
    error: Option<&str>,
) -> BuildResult<Vec<ScenarioRow>> {
    let mut ids = BTreeSet::new();
    declarations
        .into_iter()
        .map(|declaration| {
            let row = describe_scenario(feature, &declaration, error);
            if !ids.insert(row.id.clone()) {
                return Err(format!(
                    "scenario {:?} in {path:?}: expected unique names within a feature",
                    declaration.name
                ));
            }
            Ok(row)
        })
        .collect()
}

#[derive(Clone, Debug, Serialize, PartialEq, Eq)]
struct StepSignature {
    line: usize,
    name: String,
}
#[derive(Debug)]
struct Declaration {
    name: String,
    line: usize,
    rule: Option<String>,
    background_steps: Vec<StepSignature>,
    scenario_steps: Vec<StepSignature>,
    outline: bool,
}
fn parsed_declarations(feature: &Feature) -> Vec<Declaration> {
    let mut declarations = Vec::new();
    for scenario in &feature.scenarios {
        declarations.push(parsed_declaration(
            scenario,
            None,
            feature.background.as_ref(),
            None,
        ));
    }
    for rule in &feature.rules {
        for scenario in &rule.scenarios {
            declarations.push(parsed_declaration(
                scenario,
                Some(&rule.name),
                feature.background.as_ref(),
                rule.background.as_ref(),
            ));
        }
    }
    declarations
}
fn parsed_declaration(
    scenario: &Scenario,
    rule: Option<&String>,
    feature_background: Option<&Background>,
    rule_background: Option<&Background>,
) -> Declaration {
    let backgrounds = background_steps(feature_background).chain(background_steps(rule_background));
    Declaration {
        name: scenario.name.clone(),
        line: scenario.position.line,
        rule: rule.cloned(),
        background_steps: backgrounds.map(step_signature).collect(),
        scenario_steps: scenario.steps.iter().map(step_signature).collect(),
        outline: !scenario.examples.is_empty(),
    }
}
fn background_steps(background: Option<&Background>) -> impl Iterator<Item = &Step> {
    background.into_iter().flat_map(|item| item.steps.iter())
}
fn step_signature(step: &Step) -> StepSignature {
    StepSignature {
        line: step.position.line,
        name: step.value.clone(),
    }
}

fn fallback_declarations(path: &Path, bytes: &[u8]) -> BuildResult<Vec<Declaration>> {
    let text = std::str::from_utf8(bytes)
        .map_err(|error| format!("scenario source {path:?}: expected UTF-8: {error}"))?;
    let mut delimiter: Option<&str> = None;
    let mut declarations = Vec::new();
    for (index, line) in text.lines().enumerate() {
        let line = line.trim_start();
        if let Some(current) = delimiter {
            if line.starts_with(current) {
                delimiter = None;
            }
            continue;
        }
        if line.starts_with("\"\"\"") {
            delimiter = Some("\"\"\"");
            continue;
        }
        if line.starts_with("```") {
            delimiter = Some("```");
            continue;
        }
        if let Some(declaration) = fallback_declaration(line, index + 1) {
            declarations.push(declaration);
        }
    }
    if declarations.is_empty() {
        return Err(format!(
            "scenario source {path:?}: expected scenario declarations in invalid Gherkin"
        ));
    }
    Ok(declarations)
}
fn fallback_declaration(line: &str, position: usize) -> Option<Declaration> {
    for (prefix, outline) in [
        ("Scenario:", false),
        ("Scenario Outline:", true),
        ("Scenario Template:", true),
    ] {
        if let Some(name) = line.strip_prefix(prefix) {
            return Some(Declaration {
                name: name.trim().to_owned(),
                line: position,
                rule: None,
                background_steps: Vec::new(),
                scenario_steps: Vec::new(),
                outline,
            });
        }
    }
    None
}

fn parse_feature(path: &Path, bytes: &[u8]) -> BuildResult<Feature> {
    let text = std::str::from_utf8(bytes)
        .map_err(|error| format!("scenario source {path:?}: expected UTF-8: {error}"))?;
    let mut content = text.to_owned();
    if !content.ends_with('\n') {
        content.push('\n');
    }
    Feature::parse(&content, Default::default())
        .map_err(|error| format!("scenario source {path:?}: expected valid Gherkin: {error}"))
}
fn describe_scenario(
    feature: &super::ScenarioFeatureSource,
    scenario: &Declaration,
    error: Option<&str>,
) -> ScenarioRow {
    let mut identity = Sha256::new();
    identity.update(feature.path.as_bytes());
    identity.update([0]);
    identity.update(scenario.name.as_bytes());
    ScenarioRow {
        id: format!("scenario:{:x}", identity.finalize()),
        path: feature.path.clone(),
        area: feature.area.clone(),
        feature: feature.feature.clone(),
        source_sha256: feature.sha256.clone(),
        line: scenario.line,
        scenario: scenario.name.clone(),
        rule: scenario.rule.clone(),
        background_steps: scenario.background_steps.clone(),
        scenario_steps: scenario.scenario_steps.clone(),
        outline: scenario.outline,
        milestone: None,
        owner: None,
        state: if error.is_some() {
            "blocked"
        } else {
            "inventoried"
        },
        source_error: error.map(str::to_owned),
        last_receipt: None,
    }
}
#[cfg(test)]
#[path = "../../tests/scenarios/ledger.rs"]
mod tests;
