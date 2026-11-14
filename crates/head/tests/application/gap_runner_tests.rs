// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{given, then, when};

#[derive(Debug, Default, cucumber::World)]
struct DiagnosticWorld {
    actions: usize,
}
#[given("diagnostic failure")]
fn fail_step(_: &mut DiagnosticWorld) {
    panic!("expected diagnostic failure");
}
#[when("diagnostic action")]
fn action_step(world: &mut DiagnosticWorld) {
    world.actions += 1;
}
#[then("diagnostic action happened")]
fn verify_step(world: &mut DiagnosticWorld) {
    assert_eq!(world.actions, 1);
}

#[test]
fn continues_after_failure_and_marks_later_results_tainted() {
    let feature = Feature::parse("Feature: test\n Scenario: continued\n  Given diagnostic failure\n  When diagnostic action\n  Then diagnostic action happened\n", Default::default()).unwrap();
    let mut world = DiagnosticWorld::default();
    let outcomes = execute_steps(
        &mut world,
        &DiagnosticWorld::collection(),
        feature.scenarios[0].steps.iter(),
    );
    assert_eq!(
        outcomes.iter().map(|step| step.result).collect::<Vec<_>>(),
        ["failed", "passed", "passed"]
    );
    assert_eq!(
        outcomes.iter().map(|step| step.tainted).collect::<Vec<_>>(),
        [false, true, true]
    );
    assert_eq!(world.actions, 1);
    assert_eq!(classify(&outcomes), "bound but fails");
}
#[test]
fn unbound_steps_taint_later_steps_and_are_not_passes() {
    let feature = Feature::parse("Feature: test\n Scenario: missing\n  Given missing diagnostic binding\n  When diagnostic action\n", Default::default()).unwrap();
    let mut world = DiagnosticWorld::default();
    let outcomes = execute_steps(
        &mut world,
        &DiagnosticWorld::collection(),
        feature.scenarios[0].steps.iter(),
    );
    assert_eq!(outcomes[0].result, "unbound");
    assert!(outcomes[1].tainted);
    assert_eq!(classify(&outcomes), "unbound step");
}
#[test]
fn snapshots_expand_outlines_and_detect_source_changes() {
    let folder = tempfile::tempdir().unwrap();
    let path = folder.path().join("test.feature");
    fs::write(&path, "Feature: test\n Scenario Outline: examples\n  When diagnostic action <value>\n  Examples:\n   | value |\n   | one |\n   | two |\n").unwrap();
    let original = read_sources(folder.path());
    let features = parse_sources(&original);
    assert_eq!(features[0].scenarios.len(), 2);
    assert_eq!(features[0].path.as_ref(), Some(&path));
    fs::write(&path, "Feature: changed\n").unwrap();
    assert_ne!(original, read_sources(folder.path()));
}

#[given(regex = r"^ambiguous diagnostic .+$")]
fn ambiguous_first(_: &mut DiagnosticWorld) {}
#[given(regex = r"^ambiguous diagnostic value$")]
fn ambiguous_second(_: &mut DiagnosticWorld) {}
#[test]
fn ambiguous_binding_is_reported_without_executing_it() {
    let feature = Feature::parse("Feature: test\n Scenario: ambiguous\n  Given ambiguous diagnostic value\n  When diagnostic action\n", Default::default()).unwrap();
    let mut world = DiagnosticWorld::default();
    let outcomes = execute_steps(
        &mut world,
        &DiagnosticWorld::collection(),
        feature.scenarios[0].steps.iter(),
    );
    assert_eq!(outcomes[0].result, "ambiguous");
    assert!(
        outcomes[0]
            .detail
            .as_ref()
            .unwrap()
            .contains("Possible matches")
    );
    assert!(outcomes[1].tainted);
    assert_eq!(classify(&outcomes), "ambiguous binding");
}

#[test]
fn changed_source_prevents_writing_the_report() {
    let folder = tempfile::tempdir().unwrap();
    let path = folder.path().join("test.feature");
    fs::write(&path, "Feature: original\n").unwrap();
    let original = read_sources(folder.path());
    fs::write(&path, "Feature: changed\n").unwrap();
    let output = folder.path().join("gap-report.json");
    let report = GapReport {
        kind: "diagnostic",
        source_files: BTreeMap::new(),
        scenarios: Vec::new(),
    };
    let result =
        std::panic::catch_unwind(|| write_report(folder.path(), &original, &output, &report));
    assert!(result.is_err());
    assert!(!output.exists());
}

#[given("diagnostic fixture is unavailable")]
fn unavailable_fixture(_: &mut DiagnosticWorld) {
    std::panic::panic_any(FixtureBlocked("missing drawing fixture".into()));
}
#[test]
fn explicit_fixture_failure_is_distinct_from_a_product_failure() {
    let feature = Feature::parse("Feature: test\n Scenario: fixture\n  Given diagnostic fixture is unavailable\n  When diagnostic action\n", Default::default()).unwrap();
    let mut world = DiagnosticWorld::default();
    let outcomes = execute_steps(
        &mut world,
        &DiagnosticWorld::collection(),
        feature.scenarios[0].steps.iter(),
    );
    assert_eq!(
        outcomes[0].detail.as_deref(),
        Some("missing drawing fixture")
    );
    assert_eq!(classify(&outcomes), "blocked by fixture");
    assert!(outcomes[1].tainted);
}

#[test]
fn failed_window_setup_marks_every_step_unexecuted() {
    let feature = Feature::parse(
        "Feature: test\n Scenario: setup\n  Given diagnostic failure\n  When diagnostic action\n",
        Default::default(),
    )
    .unwrap();
    let outcomes = unexecuted_steps(feature.scenarios[0].steps.iter());
    assert_eq!(outcomes.len(), 2);
    assert!(
        outcomes
            .iter()
            .all(|step| step.result == "not executed" && step.tainted)
    );
}

#[test]
fn empty_scenario_is_not_classified_as_passing() {
    assert_eq!(classify(&[]), "no steps");
}

#[given("diagnostic fixture has a textual decline")]
fn textual_fixture_decline(_: &mut DiagnosticWorld) {
    block_fixture("missing sketch plane");
}
#[test]
fn textual_fixture_decline_keeps_the_reason_in_diagnostic_and_normal_runs() {
    let feature = Feature::parse(
        "Feature: test\n Scenario: fixture\n  Given diagnostic fixture has a textual decline\n",
        Default::default(),
    )
    .unwrap();
    let mut world = DiagnosticWorld::default();
    let outcomes = execute_steps(
        &mut world,
        &DiagnosticWorld::collection(),
        feature.scenarios[0].steps.iter(),
    );
    assert_eq!(classify(&outcomes), "blocked by fixture");
    assert_eq!(outcomes[0].detail.as_deref(), Some("missing sketch plane"));
}
