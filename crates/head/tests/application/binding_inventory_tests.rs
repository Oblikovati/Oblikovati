// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::feature::Ext as _;
#[test]
fn capture_only_is_counted_but_not_checked_as_an_executable_binding() {
    let feature = Feature::parse("Feature: source\n @capture_only\n Scenario: historical\n  Given a missing binding\n Scenario: live\n  Given another missing binding\n", Default::default()).unwrap();
    let mut report = BindingInventory::default();
    report.inspect_feature(&feature, &cucumber::step::Collection::new());
    assert_eq!(
        (report.scenarios, report.excluded_capture_only, report.steps),
        (2, 1, 1)
    );
    assert_eq!(report.unresolved.len(), 1);
}
#[test]
fn capture_host_only_is_counted_as_blocked_without_host_attestation() {
    let feature = Feature::parse("@capture_host_only\nFeature: host\n @capture_only\n Scenario: archived\n  Given missing\n Scenario: host fixture\n  Given missing\n", Default::default()).unwrap();
    let mut report = BindingInventory::default();
    report.inspect_feature(&feature, &cucumber::step::Collection::new());
    assert_eq!(
        (
            report.scenarios,
            report.excluded_capture_only,
            report.blocked_capture_host_only,
            report.steps
        ),
        (2, 1, 1, 0)
    );
}
#[test]
fn outlines_and_both_backgrounds_count_as_executable_steps() {
    let feature = Feature::parse("Feature: source\n  Background: shared\n    Given source setup\n  Rule: rules\n    Background: rule setup\n      Given rule setup\n    Scenario Outline: values\n      When value <value>\n      Then result\n      Examples:\n        | value |\n        | one |\n        | two |\n", Default::default()).unwrap().expand_examples().unwrap();
    let mut report = BindingInventory::default();
    report.inspect_feature(&feature, &cucumber::step::Collection::new());
    assert_eq!((report.features, report.scenarios, report.steps), (1, 2, 8));
    assert_eq!(report.unresolved.len(), 8);
    assert!(
        report
            .unresolved
            .iter()
            .any(|step| step.step == "value one")
    );
    assert!(
        report
            .unresolved
            .iter()
            .any(|step| step.step == "value two")
    );
    assert!(
        report
            .unresolved
            .iter()
            .all(|step| step.line > 0 && step.cause == "undefined step")
    );
}
#[test]
fn matching_the_real_registry_does_not_construct_or_drive_a_window() {
    let feature = Feature::parse(
        "Feature: source\n  Scenario: binding\n    When I select File\n",
        Default::default(),
    )
    .unwrap();
    let mut report = BindingInventory::default();
    report.inspect_feature(&feature, &ApplicationWorld::collection());
    assert_eq!((report.features, report.scenarios, report.steps), (1, 1, 1));
    assert!(report.unresolved.is_empty());
}

#[test]
fn shared_browser_preference_step_has_exactly_one_binding() {
    let feature = Feature::parse("Feature: shared browser\n  Scenario: checked preference\n    Then Hide Work Features is checked\n", Default::default()).unwrap();
    let mut report = BindingInventory::default();
    report.inspect_feature(&feature, &ApplicationWorld::collection());
    assert_eq!(report.steps, 1);
    assert!(
        report.unresolved.is_empty(),
        "expected a unique shared step; {}",
        serde_json::to_string(&report).unwrap()
    );
}
