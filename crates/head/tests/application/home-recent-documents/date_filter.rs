// SPDX-License-Identifier: GPL-2.0-only
//! Combine the Today filter with Part and Assembly types in the Recent list.
use super::*;

#[given("recent parts and an assembly modified today are available")]
fn native_rows_modified_today(world: &mut ApplicationWorld) {
    use oblikovati_gui::GuiTemplateKind::{Assembly, Part};
    parts_and_assemblies(world);
    let rows = world.state().recent_documents;
    for row in rows
        .iter()
        .filter(|row| matches!(row.kind, Part | Assembly))
    {
        assert_eq!(
            row.modified_age_days,
            Some(0),
            "expected {} modified today",
            row.name
        );
    }
}

#[when("the user opens Filter and chooses Today under Date Modified")]
#[when("I open Filter and choose Today under Date Modified")]
fn filter_today(world: &mut ApplicationWorld) {
    open_filter(world);
    world.click_text("Today");
    world.pump(Vec::new());
}
#[then("the recent list includes the parts and assembly modified today")]
#[then("the list displays the parts and assembly modified today")]
fn today_list(world: &mut ApplicationWorld) {
    parts_today(world);
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_some(),
        "expected {assembly}"
    );
}
#[when("the user selects Part under File Types")]
#[when("I select Part under File Types")]
fn select_part_type(world: &mut ApplicationWorld) {
    world.click_text("Part");
    world.pump(Vec::new());
}
#[then("the filter control indicates two active filters")]
fn two_filters(world: &mut ApplicationWorld) {
    expect_filter_count(world, 2);
}
#[then("the list displays the parts modified today")]
fn parts_today(world: &mut ApplicationWorld) {
    let names = [
        SEARCHED,
        other_recent_names(world)[0],
        other_recent_names(world)[1],
    ];
    for name in names {
        assert!(
            recent_name(world, name).is_some(),
            "expected today's Part {name}"
        );
    }
}
#[then("the assembly is not displayed")]
fn assembly_hidden(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_none(),
        "expected no {assembly}"
    );
}
#[when("the user also selects Assembly")]
#[when("I also select Assembly")]
fn also_assembly(world: &mut ApplicationWorld) {
    world.click_text("Assembly");
    world.pump(Vec::new());
}
#[then("the filter control indicates three active filters")]
fn three_filters(world: &mut ApplicationWorld) {
    expect_filter_count(world, 3);
}
#[then("All is selected for Date Modified")]
#[then("All is selected under Date Modified")]
fn all_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("All"), "expected All selected");
}
#[then("the file-type checkboxes are clear")]
fn types_clear(world: &mut ApplicationWorld) {
    for choice in ["Part", "Assembly", "Drawing"] {
        assert!(!world.checked(choice), "expected {choice} clear");
    }
}
