// SPDX-License-Identifier: GPL-2.0-only
//! Inspect target names and verify Escape preserves the active draft.
use super::super::*;

#[given("ribbon customization is open on the Ribbon tab with Part | Tools selected")]
fn open_tools_target(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    assert_eq!(transaction_steps::target_label(world), "Part|Tools");
}

#[given("the Part | Tools custom panel list is empty")]
fn part_tools_is_empty(world: &mut ApplicationWorld) {
    transaction_steps::target_is_empty(world);
}

#[then(
    "the visible part targets include Tools, Manage, View, Environments, Collaborate and the final available command tab"
)]
#[then(
    "the visible part targets include Tools, Manage, View, Environments, Collaborate and Fusion"
)]
fn primary_part_targets(world: &mut ApplicationWorld) {
    target_names(
        world,
        "Part",
        &[
            "Tools",
            "Manage",
            "View",
            "Environments",
            "Collaborate",
            "Fusion",
        ],
    );
}

#[then(
    "they also include 3D Sketch, Construction, Edit Base Solid, Repair, Route, Render, Analysis, 3D Print, Freeform, Hidden Tab and BIM Content"
)]
fn secondary_part_targets(world: &mut ApplicationWorld) {
    target_names(
        world,
        "Part",
        &[
            "3DSketch",
            "Construction",
            "EditBaseSolid",
            "Repair",
            "Route",
            "Render",
            "Analysis",
            "3DPrint",
            "Freeform",
            "HiddenTab",
            "BIMContent",
        ],
    );
}

#[then("the visible assembly targets include Mold Layout, Core/Cavity, Mold Assembly and Assemble")]
fn assembly_targets(world: &mut ApplicationWorld) {
    target_names(
        world,
        "Assembly",
        &["MoldLayout", "Core/Cavity", "MoldAssembly", "Assemble"],
    );
}

fn target_names(world: &ApplicationWorld, environment: &str, names: &[&str]) {
    let list = world.run("Part | 3D Model").list;
    for tab in names {
        let target = format!("{environment}|{tab}");
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == target),
            "expected {target:?} in the target selector"
        );
    }
}

#[then("Part | Tools remains the selected target")]
fn tools_remains_selected(world: &mut ApplicationWorld) {
    world.hover([1250, 780]);
    assert!(world.row_accented_beside("Part | Tools", "Part | Manage"));
}

#[when("the user closes the selector with Escape")]
fn escape_target_selector(world: &mut ApplicationWorld) {
    world.escape();
}

#[then("the selector closes without changing the target or custom panel list")]
fn tools_target_unchanged(world: &mut ApplicationWorld) {
    world.expect_no_text("Assembly | Assemble");
    assert_eq!(transaction_steps::target_label(world), "Part|Tools");
    transaction_steps::target_is_empty(world);
    transaction_steps::expect_target_pending(world, false);
}

#[then("the dialog closes and the part remains active")]
fn original_part_remains_active(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    assert_eq!(world.state().document, world.previous_document);
}
