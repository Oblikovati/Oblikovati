// SPDX-License-Identifier: GPL-2.0-only
//! Record the parsed arguments supplied to each native scenario (#4114).
use cucumber::gherkin::{Background, Feature, Rule, Scenario, Step};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::{
    path::Path,
    sync::{Arc, Mutex},
};

#[derive(Clone, Debug, Serialize, PartialEq, Eq)]
struct StepArguments {
    line: usize,
    name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    table: Option<Vec<Vec<String>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    docstring: Option<String>,
}
#[derive(Clone, Debug, Serialize, PartialEq, Eq)]
pub(super) struct ScenarioArguments {
    path: String,
    line: usize,
    name: String,
    background_steps: Vec<StepArguments>,
    scenario_steps: Vec<StepArguments>,
}
pub(super) type Capture = Arc<Mutex<Vec<ScenarioArguments>>>;

fn signature(step: &Step) -> StepArguments {
    StepArguments {
        line: step.position.line,
        name: step.value.clone(),
        table: step.table.as_ref().map(|table| table.rows.clone()),
        docstring: step.docstring.clone(),
    }
}
fn background_steps(background: Option<&Background>) -> impl Iterator<Item = &Step> {
    background.into_iter().flat_map(|value| value.steps.iter())
}
fn entry(
    path: String,
    feature: &Feature,
    rule: Option<&Rule>,
    scenario: &Scenario,
) -> ScenarioArguments {
    let backgrounds = background_steps(feature.background.as_ref()).chain(background_steps(
        rule.and_then(|value| value.background.as_ref()),
    ));
    ScenarioArguments {
        path,
        line: scenario.position.line,
        name: scenario.name.clone(),
        background_steps: backgrounds.map(signature).collect(),
        scenario_steps: scenario.steps.iter().map(signature).collect(),
    }
}
pub(super) fn record(
    capture: &Capture,
    root: &Path,
    feature: &Feature,
    rule: Option<&Rule>,
    scenario: &Scenario,
) {
    let path = relative_path(root, feature);
    capture
        .lock()
        .expect("argument capture mutex")
        .push(entry(path, feature, rule, scenario));
}
fn relative_path(root: &Path, feature: &Feature) -> String {
    let path = std::fs::canonicalize(feature.path.as_ref().expect("scenario feature path"))
        .expect("scenario feature path must exist");
    let root = std::fs::canonicalize(root).expect("scenario root must exist");
    let relative = path
        .strip_prefix(root)
        .expect("scenario feature must be inside scenario root");
    relative
        .components()
        .map(|part| part.as_os_str().to_str().expect("UTF-8 feature path"))
        .collect::<Vec<_>>()
        .join("/")
}
#[derive(Serialize)]
struct ArgumentReport<'a> {
    cucumber_sha256: String,
    scenarios: &'a [ScenarioArguments],
}
pub(super) fn save(path: &Path, cucumber: &Path, capture: &Capture) {
    let entries = capture.lock().expect("argument capture mutex");
    let report = std::fs::read(cucumber).expect("completed Cucumber report");
    let arguments = ArgumentReport {
        cucumber_sha256: format!("{:x}", Sha256::digest(report)),
        scenarios: entries.as_slice(),
    };
    let bytes = serde_json::to_vec(&arguments).expect("serializable scenario arguments");
    super::report_destination::write_file(path, &bytes).expect("new argument report");
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn captures_parsed_tables_and_docstrings() {
        let source = "Feature: args\n Background:\n  Given values\n   | a | b |\n Scenario: example\n  When sending\n   \"\"\"\n   body\n   \"\"\"\n";
        let feature = Feature::parse(source, Default::default()).unwrap();
        let result = entry(
            "features/test.feature".into(),
            &feature,
            None,
            &feature.scenarios[0],
        );
        assert_eq!(
            result.background_steps[0].table.as_ref().unwrap()[0],
            ["a", "b"]
        );
        assert_eq!(
            result.scenario_steps[0].docstring.as_deref(),
            Some("\nbody\n")
        );
    }
}
