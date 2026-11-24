// SPDX-License-Identifier: GPL-2.0-only
//! Native member-name checks for the model-state table (#4119).
use super::model_state_fixture::{
    COPY_STATE, create_named_state_in_browser, expand_state_branch, has_named_states,
};
use super::model_state_suppression::activate_state_row;
use super::model_state_table::open_property_table;
use super::model_state_table_extra::{click, open, row_run};
use super::model_state_table_parameter_steps::three_state_prism;
use super::*;
use oblikovati_api::{client::Client, wire::CreateModelStateArgs};

fn setup(world: &mut ApplicationWorld, name: &str, open_table: bool) {
    if has_named_states(world) {
        let states = world.state().browser.unwrap().model_states.unwrap();
        assert_eq!(states.active().unwrap().name, COPY_STATE);
        assert_eq!(states.list().len(), 4);
        assert!(states.list().iter().any(|member| member.name == name));
        if open_table {
            row_run(world, name);
        }
        return;
    }
    three_state_prism(world);
    Client::new(world.app().owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: name.into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    if open_table {
        click_branch_arrow(world, "Model State");
        world.expect_text(name);
        open(world);
    }
}

#[given("the user creates a fourth model state named Model State1 and reactivates Copy State")]
fn create_and_leave_fourth_member(world: &mut ApplicationWorld) {
    assert!(has_named_states(world));
    expand_state_branch(world);
    create_named_state_in_browser(world, "Model State1");
    activate_state_row(world, COPY_STATE);
}

#[given("the user renames Model State1 to Table Variant")]
fn rename_fourth_member_fixture(world: &mut ApplicationWorld) {
    open_property_table(world);
    rename(world);
    accept(world);
}
#[given("The MCAD Software has a model-state table with members Capture State and Model State1")]
fn duplicate_fixture(world: &mut ApplicationWorld) {
    setup(world, "Model State1", true);
}
#[given("The MCAD Software has an active Copy State and an inactive Model State1")]
fn rename_fixture(world: &mut ApplicationWorld) {
    setup(world, "Model State1", false);
}
#[given("The MCAD Software has an active Copy State and an inactive Table Variant")]
fn empty_fixture(world: &mut ApplicationWorld) {
    setup(world, "Table Variant", false);
}
#[given("no model state is named Table Variant")]
fn no_variant(world: &mut ApplicationWorld) {
    let list = Client::new(world.app().owner.document_host())
        .model_states()
        .list(&Default::default())
        .unwrap();
    assert!(
        list.model_states
            .unwrap()
            .iter()
            .all(|state| state.name != "Table Variant")
    );
}
fn edit_name(world: &mut ApplicationWorld, existing: &str, name: &str, enter: bool) {
    let point = [world.run("Member").center()[0], row_run(world, existing)[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    if name.is_empty() {
        super::super::x11_input::press_keysym(0xff08);
    }
    for character in name.chars() {
        super::super::x11_input::press_path_character(character);
    }
    if enter {
        super::super::x11_input::press_keysym(0xff0d);
    }
    world.pump(Vec::new());
}
#[when("the user changes the Model State1 member cell to \"Capture State\" and presses Enter")]
fn duplicate_name(world: &mut ApplicationWorld) {
    edit_name(world, "Model State1", "Capture State", true);
}
#[when("changes the Model State1 member cell to \"Table Variant\"")]
fn rename(world: &mut ApplicationWorld) {
    edit_name(world, "Model State1", "Table Variant", false);
}
#[when("clears the Table Variant member name and presses Enter")]
fn clear_name(world: &mut ApplicationWorld) {
    edit_name(world, "Table Variant", "", true);
}
fn yellow_row(world: &mut ApplicationWorld, row: &str, offset: f32) {
    let cell = [
        world.run("Member").center()[0] + 45.,
        row_run(world, row)[1] + offset,
    ];
    let at = diagnostics::pointer_pixels(cell);
    let pixel = world.pixel_at([at[0] as usize, at[1] as usize]);
    assert!(
        pixel[0] > pixel[1] && pixel[1] > pixel[2],
        "expected yellow member cell, got {pixel:?}"
    );
}
#[then("both matching member-name cells are highlighted yellow")]
#[then("the table remains open with both duplicate names highlighted")]
fn duplicate_highlights(world: &mut ApplicationWorld) {
    let runs: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text.contains("CaptureState")
                && run.center()[1] > world.run("Member").center()[1]
                && (run.minimum[0] - world.run("Member").minimum[0]).abs() < 12.
        })
        .map(|run| run.center()[1])
        .collect();
    assert_eq!(runs.len(), 2, "expected two duplicate member cells");
    for y in runs {
        let at = diagnostics::pointer_pixels([world.run("Member").center()[0] + 45., y]);
        let pixel = world.pixel_at([at[0] as usize, at[1] as usize]);
        assert!(
            pixel[0] > pixel[1] && pixel[1] > pixel[2],
            "expected yellow member cell, got {pixel:?}"
        );
    }
}
#[then("the empty member cell is highlighted yellow")]
fn empty_highlight(world: &mut ApplicationWorld) {
    yellow_row(world, "Copy State", 23.);
}
#[then(
    "a warning identifies the duplicate name and says duplicates will be removed if the user continues"
)]
fn duplicate_warning(world: &mut ApplicationWorld) {
    let text = world
        .frame
        .runs
        .iter()
        .map(|run| run.text.as_str())
        .collect::<String>();
    assert!(
        text.contains("CaptureState") && text.contains("Duplicateswillberemoved"),
        "warning: {text}"
    );
}
#[then("the warning offers Yes and No")]
fn warning_options(world: &mut ApplicationWorld) {
    world.expect_text("Yes");
    world.expect_text("No");
}
#[then("a message says \"Could not create\"")]
fn empty_warning(world: &mut ApplicationWorld) {
    world.expect_text("Could not create");
}
#[when("the user cancels the message")]
fn cancel_warning(world: &mut ApplicationWorld) {
    let button = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text.trim() == "Cancel")
        .unwrap()
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
}
#[when("the user accepts the table")]
fn accept(world: &mut ApplicationWorld) {
    click(world, "OK");
    for _ in 0..8 {
        if !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
        {
            return;
        }
        world.pump(Vec::new());
    }
    panic!("expected model-state table to close after acceptance");
}
#[then("the table closes without accepting the rename")]
fn rename_cancelled(world: &mut ApplicationWorld) {
    world.expect_text("Model State1");
}
#[then("the model browser shows Table Variant in place of Model State1")]
#[then("the browser still shows Table Variant")]
fn variant_visible(world: &mut ApplicationWorld) {
    world.expect_text("Table Variant");
}
#[then("the model-state table closes")]
fn table_closed(world: &mut ApplicationWorld) {
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
    );
}

#[test]
fn native_accepting_duplicate_removal_keeps_one_existing_member() {
    let mut world = ApplicationWorld::new();
    setup(&mut world, "Model State1", true);
    edit_name(&mut world, "Model State1", "Capture State", true);
    click(&mut world, "OK");
    world.expect_text("Yes");
    click(&mut world, "Yes");
    for _ in 0..8 {
        if !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
        {
            break;
        }
        world.pump(Vec::new());
    }
    let list = Client::new(world.app().owner.document_host())
        .model_states()
        .list(&Default::default())
        .unwrap();
    let states = list.model_states.unwrap();
    assert_eq!(
        states
            .iter()
            .filter(|state| state.name == "Capture State")
            .count(),
        1
    );
    assert!(
        states
            .iter()
            .any(|state| state.name == "Copy State" && state.active)
    );
    world.close();
}
