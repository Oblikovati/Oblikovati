// SPDX-License-Identifier: GPL-2.0-only
//! Native property-table interactions for model-state drafts.
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};

#[given("an empty part with Primary and two named model states is active in the MCAD Software")]
fn three_states(world: &mut ApplicationWorld) {
    world.add_part();
    let client = Client::new(world.app().owner.document_host());
    let large = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Large".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Small".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .model_states()
        .activate(&RepRef { id: large })
        .unwrap();
    world.pump(Vec::new());
}
#[when("the user chooses Edit Table from the Model State branch context menu")]
fn open_table(world: &mut ApplicationWorld) {
    right_click_row(world, "Model State");
    world.click_text("Edit Table");
    world.pump(Vec::new());
}
#[then("the model-state table dialog opens")]
fn table_opens(world: &mut ApplicationWorld) {
    world.expect_text("Model State Table");
}
#[then("its member table lists Primary and both named states")]
fn members_listed(world: &mut ApplicationWorld) {
    let header = world.run("Member");
    for name in ["Primary", "Large", "Small"] {
        assert!(
            world.frame.runs.iter().any(|run| run.text.trim() == name
                && run.center()[1] > header.center()[1]
                && (run.minimum[0] - header.minimum[0]).abs() < 12.),
            "expected member {name:?} in grid"
        );
    }
}

#[then("the member table has a Member column")]
fn member_column(world: &mut ApplicationWorld) {
    world.expect_text("Member");
}
#[when("the user selects the Properties category")]
fn properties_category(world: &mut ApplicationWorld) {
    world.click_text("Properties");
}
#[when("the user selects Category under the Summary group")]
fn select_category(world: &mut ApplicationWorld) {
    world.click_text("Category");
}
#[when("the user adds the selected property to the table")]
fn add_category(world: &mut ApplicationWorld) {
    world.click_text("Add >");
}
#[then("Category appears in the chosen-property list")]
fn category_chosen(world: &mut ApplicationWorld) {
    world.expect_text("Chosen properties");
    let occurrences = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text.trim() == "Category")
        .count();
    assert!(
        occurrences >= 3,
        "expected Category in available, chosen, and grid rows"
    );
}

fn grid_has_category(world: &ApplicationWorld) -> bool {
    let baseline = world.run("Member").center()[1];
    world
        .frame
        .runs
        .iter()
        .any(|run| run.text.trim() == "Category" && (run.center()[1] - baseline).abs() < 3.)
}
#[then("a Category column appears in the member table")]
fn category_in_grid(world: &mut ApplicationWorld) {
    assert!(grid_has_category(world));
}
#[then("it does not have a Category column")]
fn category_not_in_grid(world: &mut ApplicationWorld) {
    assert!(!grid_has_category(world));
}
#[when("the user cancels the dialog and reopens Edit Table")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
    open_table(world);
}

#[cfg(target_os = "linux")]
fn native_click(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
#[cfg(target_os = "linux")]
fn native_open_table(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(world.run("Model State").center());
    super::super::x11_input::right_click(point);
    world.pump(Vec::new());
    native_click(world, "Edit Table");
}
#[cfg(target_os = "linux")]
#[test]
fn native_table_column_cancel_discards_draft() {
    let mut world = ApplicationWorld::new();
    three_states(&mut world);
    native_click(&mut world, "Model State");
    native_open_table(&mut world);
    native_click(&mut world, "Category");
    native_click(&mut world, "Add >");
    assert!(grid_has_category(&world));
    native_click(&mut world, "Cancel");
    native_open_table(&mut world);
    assert!(!grid_has_category(&world));
    native_click(&mut world, "Cancel");
    world.close();
}

#[given("the model-state table has no Category column")]
fn no_category_initially(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&oblikovati_api::json::EmptyObject::default())
        .unwrap();
    assert!(view.columns.unwrap_or_default().iter().all(|column| {
        column
            .property
            .as_deref()
            .is_none_or(|key| key.name != "Category")
    }));
}
#[when("the user accepts the table change")]
fn accept_table(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
}
#[when("the user reopens Edit Table in the same session")]
fn reopen_table(world: &mut ApplicationWorld) {
    open_table(world);
}
#[then("the member table has Member and Category columns")]
fn member_and_category(world: &mut ApplicationWorld) {
    world.expect_text("Member");
    assert!(grid_has_category(world));
}
#[then("it lists Primary and both named states")]
fn lists_named_states(world: &mut ApplicationWorld) {
    members_listed(world);
}

#[cfg(target_os = "linux")]
#[test]
fn native_table_column_accept_survives_reopening() {
    let mut world = ApplicationWorld::new();
    three_states(&mut world);
    native_click(&mut world, "Model State");
    native_open_table(&mut world);
    native_click(&mut world, "Category");
    native_click(&mut world, "Add >");
    let x = world.run("Category").center()[0];
    let y = world.run("Large").center()[1];
    super::super::x11_input::click(diagnostics::pointer_pixels([x, y]));
    world.pump(Vec::new());
    for character in "Prototype".chars() {
        super::super::x11_input::press_path_character(character);
    }
    world.pump(Vec::new());
    world.expect_text("Prototype");
    native_click(&mut world, "OK");
    world.undo();
    native_open_table(&mut world);
    assert!(!grid_has_category(&world));
    native_click(&mut world, "Cancel");
    world.redo();
    native_open_table(&mut world);
    assert!(grid_has_category(&world));
    native_click(&mut world, "Cancel");
    let document = world.active_id();
    let path = world.save_part_to_file(document, "state-table.opd");
    reopen_part(&mut world, document, &path);
    native_open_table(&mut world);
    assert!(grid_has_category(&world));
    world.expect_text("Prototype");
    native_click(&mut world, "Cancel");
    world.close();
}

#[cfg(target_os = "linux")]
fn reopen_part(world: &mut ApplicationWorld, document: u64, path: &std::path::Path) {
    use oblikovati_api::wire::{CloseDocumentArgs, OpenDocumentArgs};
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

#[given("the second named state is active")]
fn activate_second(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let states = client
        .model_states()
        .list(&oblikovati_api::json::EmptyObject::default())
        .unwrap()
        .model_states
        .unwrap();
    let second = states.iter().find(|member| member.name == "Small").unwrap();
    client
        .model_states()
        .activate(&RepRef { id: second.id })
        .unwrap();
    world.pump(Vec::new());
}
#[given("the model-state table has a Category column with empty cells")]
pub(super) fn category_fixture(world: &mut ApplicationWorld) {
    use oblikovati_api::{
        types::{MODEL_STATE_TABLE_PROPERTY_COLUMN, ModelStatePropertyKey},
        wire::{
            ModelStateTableColumnRef, ModelStateTableProperty, ModelStateTablePropertyCell,
            SetModelStateTableArgs,
        },
    };
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    let cells = view
        .rows
        .as_deref()
        .unwrap()
        .iter()
        .map(|row| ModelStateTablePropertyCell {
            row: row.ref_.clone(),
            value: None,
        })
        .collect();
    let base = view
        .available_properties
        .as_deref()
        .unwrap()
        .iter()
        .find(|choice| choice.property == key)
        .and_then(|choice| choice.base.clone());
    let mut columns = view.columns.unwrap_or_default();
    columns.push(ModelStateTableColumnRef {
        kind: MODEL_STATE_TABLE_PROPERTY_COLUMN,
        property: Some(Box::new(key.clone())),
        ..Default::default()
    });
    let mut properties = view.properties.unwrap_or_default();
    properties.push(ModelStateTableProperty {
        property: key,
        base,
        cells: Some(cells),
    });
    client
        .model_states()
        .set_table(&SetModelStateTableArgs {
            expected_generation: view.generation,
            rows: view.rows,
            columns: Some(columns),
            properties: Some(properties),
            suppressions: view.suppressions,
            parameters: view.parameters,
        })
        .unwrap();
    world.pump(Vec::new());
}
#[when("the user opens the model-state table")]
#[when("the user reopens the model-state table")]
#[when("the user reopens the model-state table in the same session")]
pub(super) fn open_property_table(world: &mut ApplicationWorld) {
    if !world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "Primary" || run.text.starts_with("Primary("))
    {
        world.click_text("Model State");
    }
    open_table(world);
}
#[when("the user double-clicks the active state's Category cell")]
fn edit_active_cell(world: &mut ApplicationWorld) {
    let x = world.run("Category").center()[0];
    let y = world.run("Small").center()[1];
    world.double_click(diagnostics::pointer_pixels([x, y]));
}
#[when(regex = r#"^the user enters "([^"]*)"$"#)]
fn enter_cell(world: &mut ApplicationWorld, text: String) {
    world.type_text(&text);
}
#[when(regex = r#"^the user replaces its value with "([^"]*)"$"#)]
fn replace_cell(world: &mut ApplicationWorld, text: String) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(&text);
}
#[then(regex = r#"^the cell displays "([^"]*)"$"#)]
#[then(regex = r#"^the active state's Category cell displays "([^"]*)"$"#)]
fn cell_text(world: &mut ApplicationWorld, text: String) {
    world.expect_text(&text);
}
#[then("the other states' Category cells are empty")]
fn other_cells_empty(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&oblikovati_api::json::EmptyObject::default())
        .unwrap();
    let column = &view.properties.unwrap()[0];
    assert!(
        column
            .cells
            .as_ref()
            .unwrap()
            .iter()
            .take(2)
            .all(|cell| cell.value.is_none())
    );
}
#[when("the user cancels the table")]
fn cancel_table(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}

#[given("an empty part with multiple model states is active in the MCAD Software")]
fn multiple_states(world: &mut ApplicationWorld) {
    three_states(world);
}
#[given("its model-state table includes a Category column")]
fn existing_category(world: &mut ApplicationWorld) {
    category_fixture(world);
}
#[given("the active state's Category value is \"Prototype\"")]
fn existing_category_value(world: &mut ApplicationWorld) {
    use oblikovati_api::{
        types::{Variant, VariantValue},
        wire::SetPropertyArgs,
    };
    let document = world.active_id();
    Client::new(world.app().owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: "Summary Information".into(),
            name: "Category".into(),
            value: Variant {
                value: VariantValue::String("Prototype".into()),
                unit: String::new(),
            },
        })
        .unwrap();
    world.pump(Vec::new());
}
#[when("the user opens the model-state table and selects Properties")]
fn open_properties(world: &mut ApplicationWorld) {
    open_property_table(world);
    properties_category(world);
}
#[when("the user selects Category in the chosen-property list")]
fn select_chosen_category(world: &mut ApplicationWorld) {
    let point = chosen_category_position(world).expect("Category in chosen list");
    world.click(point);
}
fn chosen_category_position(world: &ApplicationWorld) -> Option<[i32; 2]> {
    let label = world.run("Chosen properties");
    let bottom = world.run("Member").center()[1];
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text.trim() == "Category"
                && run.center()[1] > label.center()[1]
                && run.center()[1] < bottom
                && (run.minimum[0] - label.minimum[0]).abs() < 12.
        })
        .map(|run| diagnostics::pointer_pixels(run.center()))
}
#[then("the remove transfer control is enabled")]
fn remove_enabled(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("< Remove"));
}
#[when("the user removes the selected property from the table")]
fn remove_chosen(world: &mut ApplicationWorld) {
    world.click_text("< Remove");
}
#[then("Category disappears from the chosen-property list")]
fn category_not_chosen(world: &mut ApplicationWorld) {
    assert!(chosen_category_position(world).is_none());
}
#[then("the Category column disappears from the member table")]
fn category_disappears(world: &mut ApplicationWorld) {
    assert!(!grid_has_category(world));
}
#[when("the user cancels the table and reopens it in the same session")]
fn cancel_table_and_reopen(world: &mut ApplicationWorld) {
    cancel_table(world);
    open_property_table(world);
}
#[then("the Category column is present")]
fn category_present(world: &mut ApplicationWorld) {
    assert!(grid_has_category(world));
}
#[then("the active state's Category value is \"Prototype\"")]
fn retained_category_value(world: &mut ApplicationWorld) {
    world.expect_text("Prototype");
}

#[cfg(target_os = "linux")]
#[test]
fn native_cancel_column_removal_keeps_value() {
    let mut world = ApplicationWorld::new();
    multiple_states(&mut world);
    existing_category(&mut world);
    existing_category_value(&mut world);
    native_click(&mut world, "Model State");
    native_open_table(&mut world);
    let point = chosen_category_position(&world).expect("Category in chosen list");
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    native_click(&mut world, "< Remove");
    assert!(!grid_has_category(&world));
    native_click(&mut world, "Cancel");
    native_open_table(&mut world);
    assert!(grid_has_category(&world));
    world.expect_text("Prototype");
    native_click(&mut world, "Cancel");
    world.close();
}
