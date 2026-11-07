// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::feature::Ext as _;
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
