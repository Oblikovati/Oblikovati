// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn switching_targets_retains_only_available_source_selections() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Manage);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    draft.switch(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.extend_source(1);
    draft.switch(RibbonTab::Model);
    assert_eq!(draft.source, Some(Command::Sketches));
    assert_eq!(draft.source_extra, [Command::Annotations]);
    draft.switch(RibbonTab::Manage);
    assert_eq!(draft.source, Some(Command::Annotations));
    assert!(draft.source_extra.is_empty());
    draft.add();
    assert_eq!(draft.rows()[1].command, Command::Annotations);
}

#[test]
fn changing_categories_replaces_a_hidden_selection_with_the_separator() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Model);
    draft.set_category(Category::View);
    assert_eq!(draft.source, None);
    draft.reset_source_selection(Command::Sketches);
    draft.set_category(Category::All);
    assert_eq!(draft.source, Some(Command::Sketches));
    draft.set_category(Category::Sketch);
    assert_eq!(draft.source, Some(Command::Separator));
    draft.set_category(Category::All);
    draft.add();
    assert_eq!(draft.rows()[0].command, Command::Separator);
}

#[test]
fn zoom_commands_are_sorted_in_view_with_stable_ids() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.set_category(Category::View);
    draft.reverse_source_order();
    let commands = draft.available_commands();
    let labels: Vec<_> = commands.iter().map(|command| command.label()).collect();
    assert_eq!(
        labels,
        [
            "<Separator>",
            "Zoom Window",
            "Zoom Selected",
            "Zoom All",
            "Zoom",
            "2D Sketches"
        ]
    );
    let ids: Vec<_> = commands[1..5].iter().map(|command| command.id()).collect();
    assert_eq!(
        ids,
        [
            "view-zoom-window",
            "view-zoom-selected",
            "view-zoom-all",
            "view-zoom"
        ]
    );
}

#[test]
fn returned_command_reappears_in_its_selected_category() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    draft.set_category(Category::View);
    assert!(!draft.available(Command::Sketches));
    draft.remove();
    assert_eq!(
        draft.available_commands(),
        vec![
            Command::Separator,
            Command::Sketches,
            Command::Zoom(ZoomAction::Zoom),
            Command::Zoom(ZoomAction::All),
            Command::Zoom(ZoomAction::Selected),
            Command::Zoom(ZoomAction::Window),
        ]
    );
    assert!(draft.target_changed());
    draft.open(RibbonTab::Tools);
    assert!(!draft.changed());
}

#[test]
fn empty_macro_category_retains_separator_and_staged_changes() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    draft.set_category(Category::Macros);
    assert_eq!(draft.available_commands(), vec![Command::Separator]);
    assert_eq!(draft.source, None);
    draft.reset_source_selection(Command::Separator);
    draft.add();
    assert_eq!(draft.available_commands(), vec![Command::Separator]);
    assert_eq!(
        draft
            .rows()
            .iter()
            .map(|entry| entry.command)
            .collect::<Vec<_>>(),
        vec![Command::Sketches, Command::Separator]
    );
    draft.open(RibbonTab::Tools);
    assert!(draft.rows().is_empty());
    assert_eq!(draft.category, Category::All);
}

#[test]
fn source_sort_preserves_selection_and_transfers_in_display_order() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    let ascending = draft.available_commands();
    assert_eq!(ascending[0], Command::Separator);
    assert!(
        ascending[1..]
            .windows(2)
            .all(|pair| sort_name(pair[0]) <= sort_name(pair[1]))
    );
    draft.reset_source_selection(Command::Sketches);
    draft.extend_source(1);
    draft.reverse_source_order();
    let descending = draft.available_commands();
    assert_eq!(descending[0], Command::Separator);
    assert!(
        descending[1..]
            .windows(2)
            .all(|pair| sort_name(pair[0]) >= sort_name(pair[1]))
    );
    assert!(!draft.changed());
    draft.add();
    assert_eq!(
        draft
            .rows()
            .iter()
            .map(|row| row.command)
            .collect::<Vec<_>>(),
        vec![Command::Annotations, Command::Sketches]
    );
    draft.reverse_source_order();
    assert_eq!(draft.available_commands()[0], Command::Separator);
}
#[test]
fn staged_tabs_cancel_together_and_applied_tabs_survive_reopening() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Model);
    draft.source = Some(Command::Sketches);
    draft.add();
    assert!(draft.changed() && draft.target_changed());
    draft.switch(RibbonTab::Sketch);
    assert!(draft.rows().is_empty() && draft.changed() && !draft.target_changed());
    draft.source = Some(Command::Tool(GuiSketchTool::Circle));
    draft.add();
    draft.open(RibbonTab::Model);
    assert!(!draft.changed() && draft.rows().is_empty());
    draft.source = Some(Command::Sketches);
    draft.add();
    draft.apply();
    draft.open(RibbonTab::Model);
    assert_eq!(draft.rows()[0].command, Command::Sketches);
    assert!(!draft.changed());
}
#[test]
fn transfer_reorder_options_and_reusable_separators_have_one_draft() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    for command in [Command::Sketches, Command::Separator, Command::Annotations] {
        draft.source = Some(command);
        draft.add();
    }
    assert!(draft.available(Command::Separator));
    assert!(!draft.available(Command::Sketches));
    draft.move_selected(-1);
    assert_eq!(draft.rows()[1].command, Command::Annotations);
    draft.set_options(1, true, false);
    draft.apply();
    assert!(draft.applied(RibbonTab::Tools)[1].large);
    assert!(!draft.applied(RibbonTab::Tools)[1].text);
    draft.remove();
    assert!(draft.available(Command::Annotations));
    assert!(draft.changed());
}
#[test]
fn reversed_addition_stays_pending_until_apply_or_cancel() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.source = Some(Command::Sketches);
    draft.add();
    draft.remove();
    assert!(draft.rows().is_empty());
    assert!(draft.changed() && draft.target_changed());
    draft.switch(RibbonTab::Model);
    assert!(draft.changed() && !draft.target_changed());
    draft.apply();
    assert!(!draft.changed());
    draft.switch(RibbonTab::Tools);
    assert!(!draft.target_changed());
}
#[test]
fn sketch_category_contains_supported_sketch_commands_once() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Sketch);
    draft.set_category(Category::Sketch);
    let labels: Vec<_> = COMMANDS
        .iter()
        .copied()
        .filter(|command| draft.available(*command))
        .map(Command::label)
        .collect();
    for expected in [
        "Line",
        "Rectangle",
        "Circle (Center Point)",
        "Horizontal",
        "Dimension",
    ] {
        assert_eq!(labels.iter().filter(|label| **label == expected).count(), 1);
    }
    assert!(!labels.contains(&"Measure"));
}
#[test]
fn multiple_commands_transfer_in_order_and_return_together() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.source = Some(Command::Sketches);
    draft.extend_source(1);
    assert_eq!(draft.source_extra, vec![Command::Annotations]);
    draft.add();
    assert_eq!(
        draft
            .rows()
            .iter()
            .map(|row| row.command)
            .collect::<Vec<_>>(),
        vec![Command::Sketches, Command::Annotations]
    );
    assert_eq!(
        draft.target_extra.iter().copied().collect::<Vec<_>>(),
        vec![1]
    );
    draft.move_selected(1);
    assert_eq!(draft.rows()[0].command, Command::Sketches);
    draft.remove();
    assert!(draft.rows().is_empty());
    assert!(draft.available(Command::Sketches) && draft.available(Command::Annotations));
    assert!(draft.target_changed());
}
#[test]
fn shift_selection_extends_and_shrinks_in_both_directions() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Annotations);
    draft.extend_source(-1);
    assert_eq!(draft.source_extra, vec![Command::Sketches]);
    draft.extend_source(1);
    assert!(draft.source_extra.is_empty());
    draft.extend_source(-1);
    draft.add();
    draft.reset_target_selection(1);
    draft.extend_target(-1);
    assert!(draft.target_extra.contains(&0));
    draft.extend_target(1);
    assert!(draft.target_extra.is_empty());
}
#[test]
fn removed_command_restores_its_options_on_its_own_target() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    assert!(!draft.rows()[0].large && !draft.rows()[0].text);
    draft.set_options(0, true, true);
    draft.remove();
    draft.switch(RibbonTab::Manage);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    assert!(!draft.rows()[0].large && !draft.rows()[0].text);
    draft.switch(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    assert!(draft.rows()[0].large && draft.rows()[0].text);
}
#[test]
fn select_all_uses_visible_source_and_all_target_rows() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.select_visible_source(&[Command::Sketches, Command::Annotations]);
    draft.add();
    assert_eq!(draft.rows().len(), 2);
    draft.reset_target_selection(1);
    draft.select_all_target();
    assert_eq!(draft.target_extra, [0].into_iter().collect());
    draft.remove();
    assert!(draft.rows().is_empty());
}
#[test]
fn import_replaces_drafts_without_applying_and_cancel_restores_applied() {
    let mut draft = Draft::default();
    draft.open(RibbonTab::Tools);
    draft.reset_source_selection(Command::Sketches);
    draft.add();
    draft.apply();
    draft.switch(RibbonTab::Manage);
    draft.reset_source_selection(Command::Annotations);
    draft.add();
    draft.import(BTreeMap::new());
    assert!(draft.target_changed());
    assert!(draft.rows().is_empty());
    assert_eq!(
        draft.applied(RibbonTab::Tools)[0].command,
        Command::Sketches
    );
    draft.open(RibbonTab::Tools);
    assert_eq!(draft.rows()[0].command, Command::Sketches);
    assert!(!draft.changed());
}
