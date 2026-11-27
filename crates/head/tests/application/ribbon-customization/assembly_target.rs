// SPDX-License-Identifier: GPL-2.0-only
//! Keep Part Tools and Assembly Design drafts independent.
use super::super::{
    ApplicationWorld, click_in_customize, diagnostics, open_tools_customization,
    transaction_steps::{
        add_command, choose_target, expect_target_pending, target_is_empty, target_label,
        target_rows,
    },
};
use super::open_target_selector;
use cucumber::{given, then, when};

#[given("the Part | Tools and Assembly | Design custom targets are empty")]
fn both_targets_are_empty(world: &mut ApplicationWorld) {
    tools_empty(world);
    choose_assembly_design(world);
    assembly_empty(world);
    choose_target(world, "Tools");
}

#[when("the user selects the Assembly | Design target")]
fn choose_assembly_design(world: &mut ApplicationWorld) {
    open_target_selector(world);
    world.click_text("Assembly | Design");
    assert!(target_label(world).starts_with("Assembly|Design"));
}

#[when("the user adds \"2D Sketches\" to the Assembly | Design target")]
fn add_to_assembly(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Assembly|Design");
    add_command(world, "2DSketches");
}

#[then("the Part | Tools target is marked as changed")]
#[then("the Part | Tools target remains marked as changed")]
fn tools_pending(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools*");
}

#[then("the Assembly | Design target is marked as changed")]
fn assembly_pending(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Assembly|Design*");
}

#[then("\"2D Sketches\" appears in the assembly target list")]
#[then("\"2D Sketches\" remains in the part target list")]
fn one_sketch_command(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2DSketches"]);
}

#[then("the Part | Tools target is empty")]
fn tools_empty(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools");
    target_is_empty(world);
}

#[then("the assembly target is empty")]
#[then("the assembly target list is empty")]
fn assembly_empty(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Assembly|Design");
    target_is_empty(world);
}

#[then("the part document remains active")]
fn original_part_active(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    assert_eq!(world.state().document, world.previous_document);
}

#[test]
fn applied_assembly_design_panel_reloads_on_the_assemble_tab() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    choose_assembly_design(&mut world);
    add_to_assembly(&mut world);
    expect_target_pending(&world, true);
    let row = super::super::transaction_steps::target_row_runs(&world)[0];
    let text = world.run_in_window("Customize", "Text");
    world.click(diagnostics::pointer_pixels([
        text.center()[0],
        row.center()[1],
    ]));
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Close");
    world.reopen_application();
    world.add_part();
    world.click_text("Tools");
    world.expect_no_text("User Commands");
    world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    world.click_text("Assemble");
    let ribbon = world.window_text("##application-ribbon");
    assert!(
        ribbon
            .split_whitespace()
            .any(|run| diagnostics::panel_caption_matches(run, "User Commands")),
        "expected User Commands in the assembly ribbon, got {ribbon:?}"
    );
    world.run_in_window("##application-ribbon", "2DSketches");
    world.close();
}
