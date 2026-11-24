// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for staged table columns across category switches (#4119).
use super::model_state_table::{category_fixture, open_property_table};
use super::model_state_table_extra::{click, grid_run};
use super::*;

#[given("the model-state table contains the Category property column")]
fn category_column(world: &mut ApplicationWorld) {
    category_fixture(world);
}
#[given("the Properties category is selected")]
fn properties_selected(world: &mut ApplicationWorld) {
    if !world.shows("Model State Table") {
        open_property_table(world);
        click(world, "Properties");
    }
    world.expect_text("Chosen properties");
}
#[when("the user selects Comments and adds it to the chosen properties")]
fn add_comments(world: &mut ApplicationWorld) {
    click(world, "Comments");
    click(world, "Add >");
}
#[then("the chosen properties include Category and Comments")]
fn selected_properties(world: &mut ApplicationWorld) {
    let heading = world.run("Chosen properties");
    for name in ["Category", "Comments"] {
        assert!(
            world.frame.runs.iter().any(|run| run.text == name
                && run.center()[1] > heading.center()[1]
                && (run.minimum[0] - heading.minimum[0]).abs() < 15.),
            "expected {name} in chosen properties"
        );
    }
}
#[then("the member grid contains Category and Comments columns")]
#[then("the member grid still contains Category and Comments columns")]
fn grid_columns(world: &mut ApplicationWorld) {
    grid_run(world, "Category");
    grid_run(world, "Comments");
}
#[when("the user selects the Suppression category")]
fn suppression(world: &mut ApplicationWorld) {
    click(world, "Suppression");
}
#[when("the user returns to the Properties category")]
fn properties(world: &mut ApplicationWorld) {
    click(world, "Properties");
}
#[when("the user cancels the table dialog")]
fn cancel(world: &mut ApplicationWorld) {
    click(world, "Cancel");
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
    );
}
