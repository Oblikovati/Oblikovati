// SPDX-License-Identifier: GPL-2.0-only
//! Native import scenarios with pending ribbon edits.
use super::*;
use transaction_steps::{add_command, choose_target, target_is_empty, target_label, target_rows};

#[given("ribbon customization shows 2D Sketches applied to the Part Tools target")]
fn applied_sketches(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    stage_sketch_visibility(world);
    for option in ["Large", "Text"] {
        toggle_row_option(world, "2DSketches", option);
        assert!(row_option_checked(world, "2DSketches", option));
    }
    click_in_customize(world, "Apply");
    assert_eq!(target_rows(world), ["2DSketches"]);
    assert_eq!(target_label(world), "Part|Tools");
}

#[given("a settings file contains 2D Sketches on Part Tools with Large and Text unchecked")]
#[given("a settings file contains 2D Sketches on Part Tools and no custom entries for Part Manage")]
fn tools_settings_file(world: &mut ApplicationWorld) {
    new_xml_path(world);
    std::fs::write(
        world.saved_path.as_ref().unwrap(),
        r#"<ribbon-customization version="5" appearance="normal">
    <tab id="tools">
        <command id="sketch-visibility" large="false" text="false" expanded="false" />
    </tab>
</ribbon-customization>"#,
    )
    .unwrap();
}

#[given("3D Annotations is staged after 2D Sketches on Part Tools but has not been applied")]
fn annotations_after_sketches(world: &mut ApplicationWorld) {
    stage_tools_annotations(world);
    assert_eq!(target_rows(world), ["2DSketches", "3DAnnotations"]);
    tools_pending(world);
}

#[given(
    "ribbon customization shows All Commands and initially empty Part Tools and Part Manage targets"
)]
fn empty_tools_and_manage(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    choose_target(world, "Manage");
    target_is_empty(world);
    choose_target(world, "Tools");
}

#[given("3D Annotations is staged on Part Tools")]
fn stage_tools_annotations(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools");
    add_command(world, "3DAnnotations");
}

#[given("2D Sketches is staged on Part Manage")]
fn stage_manage_sketches(world: &mut ApplicationWorld) {
    choose_target(world, "Manage");
    add_command(world, "2DSketches");
}

#[given("Part Manage is the selected target")]
fn manage_is_selected(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world).trim_end_matches('*'), "Part|Manage");
}

#[when("the user imports the settings file")]
fn import_settings_file(world: &mut ApplicationWorld) {
    import_customization(world);
    import_exported_settings(world);
}

#[then("the Part Tools target shows only 2D Sketches with Large and Text unchecked")]
#[then("the target contains 2D Sketches with Large and Text unchecked")]
fn imported_sketches(world: &mut ApplicationWorld) {
    sketches_still_staged(world);
    for option in ["Large", "Text"] {
        assert!(!row_option_checked(world, "2DSketches", option));
    }
}

#[then("the Part Manage target list is empty")]
fn manage_is_empty(world: &mut ApplicationWorld) {
    manage_is_selected(world);
    target_is_empty(world);
}

#[then("Part Manage shows a pending-change marker")]
fn manage_is_pending(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Manage*");
}

#[then("Part Tools shows a pending-change marker")]
fn tools_are_pending(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools*");
}

#[test]
fn cancel_after_import_restores_applied_ribbon_options() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    applied_sketches(&mut world);
    tools_settings_file(&mut world);
    annotations_after_sketches(&mut world);
    import_settings_file(&mut world);
    import_closed(&mut world);
    imported_sketches(&mut world);
    cancel_customize(&mut world);
    open_tools_customization(&mut world);
    assert_eq!(target_label(&world), "Part|Tools");
    assert_eq!(target_rows(&world), ["2DSketches"]);
    for option in ["Large", "Text"] {
        assert!(row_option_checked(&world, "2DSketches", option));
    }
    world.close();
}
