// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for sketch marking-menu commands.
use super::*;
use oblikovati_api::{
    client::Client,
    types,
    wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
};

#[test]
fn marking_menu_horizontal_constrains_a_drawn_line() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("Line");
    world.sketch_click([620, 300]);
    world.sketch_click([840, 420]);
    world.escape();
    let before = world.edited_geometry().constraints().len();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Horizontal".into(),
        }]),
        overflow: None,
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Horizontal" && run.center()[0] > 700. && run.center()[1] > 250.)
        .expect("expected Horizontal in sketch marking menu")
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.sketch_click([730, 360]);
    assert_eq!(world.edited_geometry().constraints().len(), before + 1);
    world.close();
}

#[test]
fn customization_remembers_marking_and_keyboard_tabs_in_the_session() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let before = world.edited_geometry();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "MarkingMenu");
    assert!(
        world
            .window_text("Customize")
            .contains("Environment:2DSketch")
    );
    click_in_customize(&mut world, "Close");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(
        world
            .window_text("Customize")
            .contains("Environment:2DSketch")
    );
    click_in_customize(&mut world, "Keyboard");
    assert!(world.window_text("Customize").contains("Searchcommands"));
    click_in_customize(&mut world, "Close");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(world.window_text("Customize").contains("Searchcommands"));
    click_in_customize(&mut world, "Close");
    assert!(world.edited_geometry().same_recipe(&before));
    world.close();
}

#[test]
fn overflow_editor_saves_a_command_and_the_sketch_menu_runs_it() {
    use oblikovati_api::wire::GetMarkingMenuArgs;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "MarkingMenu");
    click_in_customize(&mut world, "Addcommand");
    world.click_text("Rectangle");
    world.pump(Vec::new());
    click_in_customize(&mut world, "Apply");
    let saved = Client::new(world.app().owner.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert_eq!(saved.overflow, Some(vec!["Sketch.Rectangle".into()]));
    click_in_customize(&mut world, "Close");
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entries: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .map(|run| run.center())
        .collect();
    let overflow = entries
        .into_iter()
        .max_by(|a, b| a[1].total_cmp(&b[1]))
        .expect("expected Rectangle in the saved overflow");
    world.click(diagnostics::pointer_pixels(overflow));
    world.sketch_click([760, 380]);
    world.sketch_click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn overflow_edits_are_staged_and_removal_is_saved() {
    use oblikovati_api::wire::GetMarkingMenuArgs;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "MarkingMenu");
    click_in_customize(&mut world, "Addcommand");
    world.click_text("Rectangle");
    click_in_customize(&mut world, "Apply");
    let remove = world.run_in_window("Customize", "Remove").center();
    world.click([(remove[0] - 100.) as i32, remove[1] as i32]);
    world.click_text("Circle");
    world.pump(Vec::new());
    assert!(world.frame.runs.iter().any(|run| run.text == "Circle"
        && (run.center()[1] - remove[1]).abs() < 8.
        && run.center()[0] < remove[0]
        && run.center()[0] > remove[0] - 200.));
    let menu = Client::new(world.app().owner.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert_eq!(menu.overflow, Some(vec!["Sketch.Rectangle".into()]));
    click_in_customize(&mut world, "Cancel");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(world.window_text("Customize").contains("Rectangle"));
    click_in_customize(&mut world, "Remove");
    click_in_customize(&mut world, "Apply");
    let menu = Client::new(world.app().owner.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert!(menu.overflow.is_none());
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn marking_menu_disables_commands_during_an_open_transaction() {
    use oblikovati_api::wire::TransactionBeginArgs;
    for classic in [false, true] {
        let mut world = ApplicationWorld::new();
        if classic {
            std::fs::write(
                world.preferences_directory().join("markingmenu.yaml"),
                "classic: true\n",
            )
            .unwrap();
            world.reopen_application();
        }
        world.start_empty_sketch();
        let menu = MarkingMenuView {
            environment: types::SKETCH_ENVIRONMENT,
            quadrants: Some(vec![MarkingMenuItem {
                quadrant: types::QUADRANT_NORTH,
                command_id: "Sketch.Line".into(),
            }]),
            overflow: Some(vec!["Sketch.Rectangle".into()]),
        };
        Client::new(world.app().owner.document_host())
            .ui()
            .set_marking_menu(&SetMarkingMenuArgs { menu })
            .unwrap();
        Client::new(world.app().owner.document_host())
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Caller".into(),
            })
            .unwrap();
        world.right_click([900, 450]);
        world.pump(Vec::new());
        for label in ["Line", "Rectangle"] {
            let run = world
                .frame
                .runs
                .iter()
                .find(|run| run.text == label && run.center()[0] > 700. && run.center()[1] > 250.)
                .unwrap_or_else(|| panic!("expected {label} in marking menu"));
            assert!(run.alpha < u8::MAX, "{label}: expected disabled menu item");
        }
        let line = world
            .frame
            .runs
            .iter()
            .find(|run| run.text == "Line" && run.center()[0] > 700. && run.center()[1] > 250.)
            .unwrap()
            .center();
        world.click(diagnostics::pointer_pixels(line));
        assert!(world.state().sketch_at_ready);
        Client::new(world.app().owner.document_host())
            .transactions()
            .abort()
            .unwrap();
        world.pump(Vec::new());
        let line = world
            .frame
            .runs
            .iter()
            .find(|run| run.text == "Line" && run.center()[0] > 700. && run.center()[1] > 250.)
            .expect("expected Line after the transaction");
        assert_eq!(line.alpha, u8::MAX);
        let position = line.center();
        world.click(diagnostics::pointer_pixels(position));
        assert!(!world.state().sketch_at_ready);
        world.close();
    }
}

#[test]
fn sketch_context_delete_requires_selection_and_undo_restores_the_line() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("Line");
    world.sketch_click([620, 300]);
    world.sketch_click([840, 420]);
    world.escape();
    world.right_click([620, 300]);
    world.pump(Vec::new());
    let delete = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Delete" && run.center()[0] > 500. && run.center()[1] > 200.)
        .expect("expected Delete in sketch context menu");
    assert!(
        delete.alpha < u8::MAX,
        "Delete entries: {:?}",
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "Delete")
            .collect::<Vec<_>>()
    );
    let position = delete.center();
    world.click(diagnostics::pointer_pixels(position));
    assert_eq!(world.edited_geometry().lines().len(), 1);
    world.escape();
    world.sketch_click([730, 360]);
    assert!(world.state().sketch_delete_available);
    let selected = Client::new(world.app().owner.document_host())
        .model()
        .selection()
        .unwrap();
    world.right_click([620, 300]);
    world.pump(Vec::new());
    assert_eq!(
        Client::new(world.app().owner.document_host())
            .model()
            .selection()
            .unwrap(),
        selected
    );
    let delete = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Delete" && run.center()[0] > 500. && run.center()[1] > 200.)
        .expect("expected Delete in sketch context menu");
    assert_eq!(delete.alpha, u8::MAX);
    let position = delete.center();
    world.click(diagnostics::pointer_pixels(position));
    assert!(world.edited_geometry().lines().is_empty());
    world.undo();
    assert_eq!(world.edited_geometry().lines().len(), 1);
    world.close();
}

#[test]
fn context_menu_repeats_the_latest_sketch_constraint() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("Line");
    world.sketch_click([620, 300]);
    world.sketch_click([750, 350]);
    world.escape();
    world.click_text("Line");
    world.sketch_click([850, 400]);
    world.sketch_click([980, 480]);
    world.escape();
    world.click_text("Coincident");
    world.sketch_click([750, 350]);
    world.sketch_click([850, 400]);
    assert_eq!(world.edited_geometry().constraints().len(), 1);
    world.escape();
    world.right_click([900, 550]);
    world.pump(Vec::new());
    let repeat = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "RepeatCoincident" && run.center()[0] > 700. && run.center()[1] > 250.
        })
        .expect("expected Repeat Coincident in the sketch context menu");
    let position = repeat.center();
    world.click(diagnostics::pointer_pixels(position));
    assert!(!world.state().sketch_at_ready);
    world.expect_status("Coincident:");
    world.escape();
    world.close();
}
