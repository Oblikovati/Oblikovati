// SPDX-License-Identifier: GPL-2.0-only
//! Keep source observations without a reproducible fixture out of native execution.
use cucumber::gherkin::{Feature, Rule, Scenario};

pub(super) fn reason(
    feature: &Feature,
    rule: Option<&Rule>,
    scenario: &Scenario,
) -> Option<&'static str> {
    for (tag, reason) in [
        ("capture_only", "capture only"),
        ("capture_host_only", "capture host only"),
    ] {
        if has_tag(&feature.tags, tag)
            || rule.is_some_and(|value| has_tag(&value.tags, tag))
            || has_tag(&scenario.tags, tag)
        {
            return Some(reason);
        }
    }
    None
}

fn has_tag(tags: &[String], wanted: &str) -> bool {
    tags.iter().any(|tag| tag.trim_start_matches('@') == wanted)
}

#[test]
fn capture_only_applies_at_scenario_rule_and_feature_levels() {
    let source = "Feature: cases\n @capture_only\n Scenario: captured\n  Given a step\n Scenario: runnable\n  Given a step\n @capture_only\n Rule: old fixture\n  Scenario: inherited\n   Given a step\n";
    let feature = Feature::parse(source, Default::default()).unwrap();
    assert_eq!(
        reason(&feature, None, &feature.scenarios[0]),
        Some("capture only")
    );
    assert_eq!(reason(&feature, None, &feature.scenarios[1]), None);
    assert_eq!(
        reason(
            &feature,
            Some(&feature.rules[0]),
            &feature.rules[0].scenarios[0]
        ),
        Some("capture only")
    );
    let feature = Feature::parse(
        "@capture_only\nFeature: all\n Scenario: one\n  Given a step\n",
        Default::default(),
    )
    .unwrap();
    assert_eq!(
        reason(&feature, None, &feature.scenarios[0]),
        Some("capture only")
    );
}

#[test]
fn capture_host_only_requires_an_unimplemented_host_attestation() {
    let feature = Feature::parse("@capture_host_only\nFeature: host\n @capture_only\n Scenario: missing\n  Given a step\n Scenario: host\n  Given a step\n", Default::default()).unwrap();
    assert_eq!(
        reason(&feature, None, &feature.scenarios[0]),
        Some("capture only")
    );
    assert_eq!(
        reason(&feature, None, &feature.scenarios[1]),
        Some("capture host only")
    );
}
