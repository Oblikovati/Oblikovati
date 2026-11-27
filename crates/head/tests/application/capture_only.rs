// SPDX-License-Identifier: GPL-2.0-only
//! Keep source observations without a reproducible fixture out of native execution.
use cucumber::gherkin::{Feature, Rule, Scenario};

/// Documented pinned-source defects (see `architecture/rust-port/scenario-defects/`).
const PINNED_SOURCE_DEFECTS: &[&str] = &[
    // scenario-defects/ribbon-category-escape-measure.md (#3856)
    "Discard a pending edit from an open command category list",
    // scenario-defects/temporary-modify-pin.md (#3856)
    "Minimize to tabs after pinning the Modify expansion",
    // scenario-defects/ribbon-shift-down-clean-close.md (#3856)
    "Select adjacent available commands with Shift and Down",
    // Same clean-draft Cancel defect as ribbon-shift-down-clean-close.md (#3856)
    "Replace the available-command selection with a secondary click",
    // Expects dropdown Minimize to Panel Buttons to defer minimization until clicked twice;
    // RibbonDisplayState::choose minimizes immediately (tested in ribbon-display/modes.rs).
    "Access a ribbon tab in tabs-only mode",
];

/// Pinned scenarios requiring unimplemented Inventor-only catalog entries or commands.
const UNIMPLEMENTED_UPSTREAM_FEATURES: &[&str] = &[
    // Assembly Finish and Simplify ribbon commands are not implemented yet.
    "Open an assembly in express mode and load it fully",
    // Keyboard catalog does not yet include Pattern commands (keyboard_catalog.rs).
    "Filter keyboard commands by keyword and clear the filter",
    // Three Points work plane constructor is disabled in this build (ribbon_plane_group.rs).
    "Use a grouped command choice and then the main button",
    // Expects Inventor's unlabeled row after <Separator> / below 2-D Drawing.
    "Reverse the available command list using its Name header",
    "Navigate the descending available command list to its endpoints",
    // Expects Inventor's "Adjust Orientation(Adjust Orientation)" catalog entry.
    "Page through visible available commands",
    // Fusion tab does not exist in Oblikovati.
    "Stop wheel navigation at the visible ends of the part ribbon",
    // Sketch tab lists Exit rather than an Inventor Sketch panel (dismissal is tested natively
    // in sketch_constrain_menu::command_submenu_escape_closes_submenu_then_unpinned_area).
    "Dismiss Show Panels from the expanded Constrain command menu",
    // 0-face hollow Shell preview before face selection (1-face Shell is verified natively).
    "Continue a modeling command after switching ribbon tabs",
    "Minimize and restore the ribbon during a modeling preview",
    "Switch tabs on a left-docked ribbon during a modeling command",
    "Launch a main-panel command while its expansion is open",
    "Launch a command moved to the expanded panel",
    "Use a panel expansion while a modeling command is active",
    "Keep a pinned panel expansion through face selection and command cancellation",
    "Choose an additional command while another modeling command is active",
    "Focus a command property while an unpinned panel is expanded",
    "Focus a command property while a pinned panel is expanded",
    "Hide and restore the panel of an active modeling command",
    "Cancel an active command while the panel selector is open",
];

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
    if PINNED_SOURCE_DEFECTS.contains(&scenario.name.as_str()) {
        return Some("pinned source defect");
    }
    if UNIMPLEMENTED_UPSTREAM_FEATURES.contains(&scenario.name.as_str()) {
        return Some("unimplemented upstream feature");
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
