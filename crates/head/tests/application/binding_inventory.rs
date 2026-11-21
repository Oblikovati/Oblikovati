// SPDX-License-Identifier: GPL-2.0-only
//! Fast source coverage only. Matching a step does not establish native behavior.
use super::{
    ApplicationWorld, feature_root, gap_runner, report_destination, scenario_revision,
    scenario_root, world,
};
use cucumber::{
    Parser as _, World as _,
    gherkin::{Background, Feature, Rule, Scenario, Step},
};
use futures::StreamExt as _;
use serde::Serialize;
use std::path::{Path, PathBuf};

#[derive(Default, Serialize)]
struct BindingInventory {
    features: usize,
    scenarios: usize,
    excluded_capture_only: usize,
    blocked_capture_host_only: usize,
    steps: usize,
    unresolved: Vec<UnresolvedBinding>,
    source_revision: Option<scenario_revision::ScenarioRevision>,
}
#[derive(Serialize)]
struct UnresolvedBinding {
    source: Option<PathBuf>,
    scenario: String,
    line: usize,
    step: String,
    cause: String,
}
#[test]
fn application_step_bindings_cover_selected_source() {
    report_step_bindings(&feature_root("application"), |_| true, true);
}

/// Report selected bindings, for example `report_step_bindings(root, |_| true, false)`.
pub(super) fn report_step_bindings(
    root: &Path,
    accepted: fn(&Feature) -> bool,
    require_complete: bool,
) {
    let folder = world::reports_folder();
    let output = folder.join("step-bindings.json");
    let _reports = report_destination::reserve(&folder).unwrap_or_else(|error| panic!("{error}"));
    let before = scenario_revision::inspect(root);
    let sources = gap_runner::read_sources(root);
    let mut report = collect_bindings(root, accepted);
    let after = scenario_revision::inspect(root);
    scenario_revision::require_pinned(&scenario_root(), &[&output]);
    scenario_revision::require_unchanged(
        &before,
        &after,
        sources != gap_runner::read_sources(root),
        &[&output],
    );
    report.source_revision = Some(before);
    report_destination::write_file(&output, &serde_json::to_vec_pretty(&report).unwrap()).unwrap();
    check_binding_report(root, &output, &report, require_complete);
}

fn collect_bindings(root: &Path, accepted: fn(&Feature) -> bool) -> BindingInventory {
    let parser = cucumber::parser::Basic::default();
    let features = futures::executor::block_on(
        parser
            .parse(root.to_owned(), Default::default())
            .collect::<Vec<_>>(),
    );
    let mut report = BindingInventory::default();
    let bindings = ApplicationWorld::collection();
    let selected = features
        .into_iter()
        .map(|feature| feature.expect("expected valid benchmark Gherkin"))
        .filter(accepted);
    for feature in selected {
        report.inspect_feature(&feature, &bindings);
    }
    report
}

fn check_binding_report(
    root: &Path,
    output: &Path,
    report: &BindingInventory,
    require_complete: bool,
) {
    assert!(
        report.features > 0 && report.scenarios > 0,
        "binding source {root:?}: expected selected features with scenarios"
    );
    assert!(
        report.steps > 0,
        "binding source {root:?}: expected at least one non-capture step"
    );
    let unresolved = report.unresolved.len();
    println!(
        "{unresolved} unresolved step bindings in {output:?}. Run the native benchmark for behavior."
    );
    if require_complete {
        assert_eq!(
            unresolved, 0,
            "binding report {output:?}: expected no unresolved steps"
        );
    }
}
impl BindingInventory {
    fn inspect_feature(
        &mut self,
        feature: &Feature,
        bindings: &cucumber::step::Collection<ApplicationWorld>,
    ) {
        self.features += 1;
        for scenario in &feature.scenarios {
            self.inspect_scenario(feature, None, scenario, bindings);
        }
        for rule in &feature.rules {
            for scenario in &rule.scenarios {
                self.inspect_scenario(feature, Some(rule), scenario, bindings);
            }
        }
    }
    fn inspect_scenario(
        &mut self,
        feature: &Feature,
        rule: Option<&Rule>,
        scenario: &Scenario,
        bindings: &cucumber::step::Collection<ApplicationWorld>,
    ) {
        self.scenarios += 1;
        if let Some(reason) = super::capture_only::reason(feature, rule, scenario) {
            if reason == "capture only" {
                self.excluded_capture_only += 1;
            } else {
                self.blocked_capture_host_only += 1;
            }
            return;
        }
        let steps = background_steps(feature.background.as_ref())
            .chain(background_steps(
                rule.and_then(|value| value.background.as_ref()),
            ))
            .chain(scenario.steps.iter());
        for step in steps {
            self.inspect_step(feature, scenario, step, bindings);
        }
    }
    fn inspect_step(
        &mut self,
        feature: &Feature,
        scenario: &Scenario,
        step: &Step,
        bindings: &cucumber::step::Collection<ApplicationWorld>,
    ) {
        self.steps += 1;
        let cause = match bindings.find(step) {
            Ok(Some(_)) => return,
            Ok(None) => "undefined step".into(),
            Err(cause) => cause.to_string(),
        };
        self.unresolved.push(UnresolvedBinding {
            source: feature.path.clone(),
            scenario: scenario.name.clone(),
            line: step.position.line,
            step: step.value.clone(),
            cause,
        });
    }
}
fn background_steps(background: Option<&Background>) -> impl Iterator<Item = &Step> {
    background
        .into_iter()
        .flat_map(|background| background.steps.iter())
}
#[cfg(test)]
#[path = "binding_inventory_tests.rs"]
mod tests;
