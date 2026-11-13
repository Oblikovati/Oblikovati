// SPDX-License-Identifier: GPL-2.0-only
//! Diagnostic execution is not acceptance: later steps can see an invalid world.
use super::{ApplicationWorld, scenario_selected, world};
use cucumber::{
    World as _,
    feature::Ext as _,
    gherkin::{Background, Feature, Scenario, Step},
};
use futures::FutureExt as _;
use serde::Serialize;
use std::{
    any::Any,
    collections::BTreeMap,
    fs,
    panic::AssertUnwindSafe,
    path::{Path, PathBuf},
};

#[derive(Serialize)]
struct GapReport {
    kind: &'static str,
    source_files: BTreeMap<PathBuf, String>,
    scenarios: Vec<ScenarioReport>,
}
#[derive(Serialize)]
struct ScenarioReport {
    source: Option<PathBuf>,
    scenario: String,
    line: usize,
    classification: &'static str,
    setup_error: Option<String>,
    steps: Vec<StepReport>,
}
/// A fixture provider may raise this when a required external fixture is unavailable.
#[derive(Debug)]
pub(super) struct FixtureBlocked(pub(super) String);
#[derive(Serialize)]
struct StepReport {
    line: usize,
    step: String,
    result: &'static str,
    tainted: bool,
    detail: Option<String>,
}

pub(super) fn run(root: &Path, keep: fn(&Feature) -> bool) {
    let report_path = world::reports_folder().join("gap-report.json");
    if report_path.exists() {
        fs::remove_file(&report_path).unwrap();
    }
    let source = read_sources(root);
    let features = parse_sources(&source);
    let mut report = new_report(&source);
    for feature in &features {
        if keep(feature) {
            inspect_feature(feature, &mut report);
        }
    }
    assert!(
        !report.scenarios.is_empty(),
        "diagnostic source {root:?}: expected selected scenarios"
    );
    write_report(root, &source, &report_path, &report);
    println!(
        "Diagnostic report: {} (not acceptance)",
        report_path.display()
    );
}

fn new_report(source: &BTreeMap<PathBuf, Vec<u8>>) -> GapReport {
    let source_files = source
        .iter()
        .map(|(path, bytes)| {
            (
                path.clone(),
                String::from_utf8(bytes.clone()).expect("source must be UTF-8"),
            )
        })
        .collect();
    GapReport {
        kind: "diagnostic only; not acceptance",
        source_files,
        scenarios: Vec::new(),
    }
}

fn write_report(root: &Path, source: &BTreeMap<PathBuf, Vec<u8>>, path: &Path, report: &GapReport) {
    assert!(
        source == &read_sources(root),
        "diagnostic source {root:?}: files changed during execution; report discarded"
    );
    for scenario in &report.scenarios {
        println!("{}: {}", scenario.classification, scenario.scenario);
    }
    fs::create_dir_all(path.parent().unwrap()).unwrap();
    fs::write(path, serde_json::to_vec_pretty(report).unwrap()).unwrap();
}

fn read_sources(root: &Path) -> BTreeMap<PathBuf, Vec<u8>> {
    let mut sources = BTreeMap::new();
    collect_sources(root, &mut sources);
    sources
}
fn collect_sources(path: &Path, sources: &mut BTreeMap<PathBuf, Vec<u8>>) {
    if path.is_dir() {
        for entry in fs::read_dir(path).unwrap() {
            collect_sources(&entry.unwrap().path(), sources);
        }
    } else if path
        .extension()
        .is_some_and(|extension| extension == "feature")
    {
        sources.insert(path.to_owned(), fs::read(path).unwrap());
    }
}
fn parse_sources(sources: &BTreeMap<PathBuf, Vec<u8>>) -> Vec<Feature> {
    sources
        .iter()
        .map(|(path, bytes)| {
            let mut text = std::str::from_utf8(bytes)
                .unwrap_or_else(|error| panic!("source {path:?}: expected UTF-8: {error}"))
                .to_owned();
            if !text.ends_with('\n') {
                text.push('\n');
            }
            let mut feature = Feature::parse(&text, Default::default())
                .unwrap_or_else(|error| panic!("source {path:?}: expected valid Gherkin: {error}"))
                .expand_examples()
                .unwrap_or_else(|error| panic!("source {path:?}: invalid examples: {error}"));
            feature.path = Some(path.clone());
            feature
        })
        .collect()
}
fn background_steps(background: Option<&Background>) -> impl Iterator<Item = &Step> {
    background
        .into_iter()
        .flat_map(|background| background.steps.iter())
}
fn inspect_feature(feature: &Feature, report: &mut GapReport) {
    for scenario in &feature.scenarios {
        inspect_scenario(feature, None, scenario, report);
    }
    for rule in &feature.rules {
        for scenario in &rule.scenarios {
            inspect_scenario(feature, rule.background.as_ref(), scenario, report);
        }
    }
}
fn inspect_scenario(
    feature: &Feature,
    background: Option<&Background>,
    scenario: &Scenario,
    report: &mut GapReport,
) {
    if !scenario_selected(&scenario.name) {
        return;
    }
    let steps = background_steps(feature.background.as_ref())
        .chain(background_steps(background))
        .chain(scenario.steps.iter());
    let (outcomes, setup_error) = run_native_steps(steps);
    let classification = if setup_error.is_some() {
        "setup failed"
    } else {
        classify(&outcomes)
    };
    report.scenarios.push(ScenarioReport {
        source: feature.path.clone(),
        scenario: scenario.name.clone(),
        line: scenario.position.line,
        classification,
        setup_error,
        steps: outcomes,
    });
}
fn run_native_steps<'a>(
    steps: impl Iterator<Item = &'a Step>,
) -> (Vec<StepReport>, Option<String>) {
    match std::panic::catch_unwind(ApplicationWorld::new) {
        Ok(mut world) => {
            let outcomes = execute_steps(&mut world, &ApplicationWorld::collection(), steps);
            world.close();
            (outcomes, None)
        }
        Err(error) => (unexecuted_steps(steps), Some(panic_message(error))),
    }
}
fn unexecuted_steps<'a>(steps: impl Iterator<Item = &'a Step>) -> Vec<StepReport> {
    steps
        .map(|step| StepReport {
            line: step.position.line,
            step: step.value.clone(),
            result: "not executed",
            tainted: true,
            detail: None,
        })
        .collect()
}

fn execute_steps<'a, W>(
    world: &mut W,
    bindings: &cucumber::step::Collection<W>,
    steps: impl Iterator<Item = &'a Step>,
) -> Vec<StepReport> {
    let mut tainted = false;
    steps
        .map(|step| {
            let (result, detail) = execute_step(world, bindings, step);
            let outcome = StepReport {
                line: step.position.line,
                step: step.value.clone(),
                result,
                tainted,
                detail,
            };
            tainted |= result != "passed";
            outcome
        })
        .collect()
}
fn execute_step<W>(
    world: &mut W,
    bindings: &cucumber::step::Collection<W>,
    step: &Step,
) -> (&'static str, Option<String>) {
    let (binding, _, _, context) = match bindings.find(step) {
        Ok(Some(binding)) => binding,
        Ok(None) => return ("unbound", None),
        Err(error) => return ("ambiguous", Some(error.to_string())),
    };
    let task = AssertUnwindSafe(async { binding(world, context).await }).catch_unwind();
    match futures::executor::block_on(task) {
        Ok(()) => ("passed", None),
        Err(error) => match error.downcast_ref::<FixtureBlocked>() {
            Some(blocked) => ("fixture blocked", Some(blocked.0.clone())),
            None => ("failed", Some(panic_message(error))),
        },
    }
}
fn panic_message(error: Box<dyn Any + Send>) -> String {
    error
        .downcast_ref::<String>()
        .cloned()
        .or_else(|| error.downcast_ref::<&str>().map(|s| (*s).to_owned()))
        .unwrap_or_else(|| "step panicked without a string message".into())
}
fn classify(steps: &[StepReport]) -> &'static str {
    if steps.is_empty() {
        return "no steps";
    }
    steps
        .iter()
        .find_map(|step| match step.result {
            "unbound" => Some("unbound step"),
            "ambiguous" => Some("ambiguous binding"),
            "failed" => Some("bound but fails"),
            "fixture blocked" => Some("blocked by fixture"),
            _ => None,
        })
        .unwrap_or("passes")
}
#[cfg(test)]
#[path = "gap_runner_tests.rs"]
mod tests;
