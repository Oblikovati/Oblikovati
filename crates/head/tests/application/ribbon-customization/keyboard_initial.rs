// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the current keyboard editor and its active-document context.
use super::*;

#[then("the Categories selector shows \"All\"")]
fn all_categories(world: &mut ApplicationWorld) {
    displayed_all(world);
    world.previous_document = world.state().document;
    if world.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch {
        world.record_initial_sketch();
    }
}

#[then("the command-type selector shows \"All\"")]
fn all_command_types(world: &mut ApplicationWorld) {
    displayed_all(world);
}

pub(super) fn displayed_all(world: &ApplicationWorld) {
    let anchor = world.run("Commandtype");
    let list = diagnostics::window_draw_list("Customize").unwrap();
    assert_eq!(anchor.list, list);
    let caption: String = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == list && (run.center()[1] - anchor.center()[1]).abs() < 3.)
        .map(|run| run.text.as_str())
        .collect();
    assert!(
        caption.contains("AllCategoriesAllCommandtype"),
        "keyboard filter row: {caption:?}"
    );
}

#[then("the keyword search field is empty")]
pub(super) fn keyword_search_is_empty(world: &mut ApplicationWorld) {
    let label = search_label(world);
    let y = label.center()[1];
    assert!(!world.frame.runs.iter().any(|run| {
        run.list == label.list
            && (run.center()[1] - y).abs() < 3.
            && (2. ..260.).contains(&(label.minimum[0] - run.minimum[0]))
            && !run.text.is_empty()
    }));
}

pub(super) fn search_label(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    let label = world.run("Searchcommands");
    assert_eq!(Some(label.list), diagnostics::window_draw_list("Customize"));
    label
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
    if let Some(original) = world
        .observed_sketch
        .as_ref()
        .map(|sketch| sketch.geometry.clone())
    {
        assert!(world.edited_geometry().same_recipe(&original));
    } else {
        let state = world.state();
        assert_eq!(
            state.document,
            Some(
                world
                    .previous_document
                    .expect("expected the original active Part")
            )
        );
        assert!(state.active_sketch.is_none());
    }
}
