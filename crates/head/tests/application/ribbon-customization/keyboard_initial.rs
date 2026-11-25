// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the current keyboard editor and its active-document context.
use super::*;

#[then("the Categories selector shows \"All\"")]
fn all_categories(world: &mut ApplicationWorld) {
    displayed_all(world, "Categories");
    if world.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch {
        world.record_initial_sketch();
    }
}

#[then("the command-type selector shows \"All\"")]
fn all_command_types(world: &mut ApplicationWorld) {
    displayed_all(world, "Commandtype");
}

fn displayed_all(world: &ApplicationWorld, name: &str) {
    let label = world.run_in_window("Customize", name);
    let x = label.minimum[0];
    let y = label.center()[1];
    assert!(
        world.frame.runs.iter().any(|run| {
            run.list == label.list
                && run.text == "All"
                && (run.center()[1] - y).abs() < 3.
                && (0. ..140.).contains(&(x - run.minimum[0]))
        }),
        "expected All in the {name} control"
    );
}

#[then("the keyword search field is empty")]
fn keyword_search_is_empty(world: &mut ApplicationWorld) {
    let label = world.run_in_window("Customize", "Searchcommands");
    let y = label.center()[1];
    assert!(!world.frame.runs.iter().any(|run| {
        run.list == label.list
            && (run.center()[1] - y).abs() < 3.
            && (2. ..260.).contains(&(label.minimum[0] - run.minimum[0]))
            && !run.text.is_empty()
    }));
}

#[then("the command list has Keys, Command Name, Type and Category columns")]
fn keyboard_columns(world: &mut ApplicationWorld) {
    let columns = ["Keys", "CommandName", "Type", "Category"]
        .map(|name| world.run_in_window("Customize", name));
    for pair in columns.windows(2) {
        assert_eq!(pair[0].list, pair[1].list);
        assert!(pair[0].minimum[0] < pair[1].minimum[0]);
        assert!((pair[0].center()[1] - pair[1].center()[1]).abs() < 3.);
    }
}

#[then("the command list is populated")]
fn keyboard_rows_exist(world: &mut ApplicationWorld) {
    let list = world.run_in_window("Customize", "Keys").list;
    let bindings = world.state().keymap_bindings;
    let visible = bindings
        .iter()
        .filter(|binding| {
            let name: String = binding.display_name.split_whitespace().collect();
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == name)
        })
        .count();
    assert!(visible >= 3, "expected a populated keyboard catalog");
}

#[then("\"Use default multi-character command aliases\" is unchecked")]
fn use_default_aliases_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Use default multi-character command aliases"));
    assert!(!world.state().keymap.unwrap().default_aliases());
}

#[then("the Reset All Keys and Copy to Clipboard controls are visible")]
fn keyboard_buttons(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "ResetAllKeys");
    world.run_in_window("Customize", "CopytoClipboard");
}

#[then("the customization dialog closes without changing the active sketch")]
fn active_sketch_unchanged(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    let original = world
        .observed_sketch
        .as_ref()
        .expect("expected a sketch in edit before opening keyboard customization")
        .geometry
        .clone();
    assert!(world.edited_geometry().same_recipe(&original));
}
