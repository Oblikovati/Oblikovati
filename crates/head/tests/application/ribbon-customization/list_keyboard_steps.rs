// SPDX-License-Identifier: GPL-2.0-only
//! Inspect keyboard selection in the two rendered ribbon command lists.
use super::list_steps::source_rows;
use super::transaction_steps::{add_command, target_row_runs, target_rows};
use super::*;
use oblikovati_viewport::{ViewportCapture, ViewportPhysicalKey};

#[when("the user presses the Z key while the available commands list has focus")]
fn jump_to_z(world: &mut ApplicationWorld) {
    assert!(!source_rows(world).iter().any(|row| row.text == "ZParallel"));
    world.key(ViewportPhysicalKey::Z, false);
    world.pump(Vec::new());
}

#[then("the available commands list scrolls to and selects \"Z Parallel\"")]
#[then("the list scrolls to and selects \"Z Parallel\"")]
fn selected_z_parallel(world: &mut ApplicationWorld) {
    expect_source_highlight(world, "ZParallel", "ZoomAll");
}

#[then("the list scrolls to its end and selects \"Zoom Window\"")]
fn selected_source_end(world: &mut ApplicationWorld) {
    assert_eq!(
        source_rows(world).last().map(|row| row.text.as_str()),
        Some("ZoomWindow")
    );
    expect_source_highlight(world, "ZoomWindow", "ZoomAll");
}

#[then("the list scrolls to its beginning and selects \"<Separator>\"")]
fn selected_source_start(world: &mut ApplicationWorld) {
    assert_eq!(
        source_rows(world).first().map(|row| row.text.as_str()),
        Some("<Separator>")
    );
    expect_source_highlight(world, "<Separator>", "2DSketches");
}

fn expect_source_highlight(world: &mut ApplicationWorld, selected: &str, plain: &str) {
    world.hover([800, 900]);
    let rows = source_rows(world);
    for label in [selected, plain] {
        assert!(
            rows.iter().any(|row| row.text == label),
            "expected {label} in the source list"
        );
    }
    assert!(
        world.row_accented_beside(selected, plain),
        "expected {selected} highlighted"
    );
}

#[then("\"2D Sketches\" and \"3D Annotations\" are both highlighted in the available-command list")]
fn adjacent_source_commands_highlighted(world: &mut ApplicationWorld) {
    for label in ["2DSketches", "3DAnnotations"] {
        expect_source_highlight(world, label, "2-DDrawing");
    }
}

#[when("the user closes customization with \"Cancel\"")]
fn click_published_cancel(world: &mut ApplicationWorld) {
    click_in_customize(world, "Cancel");
}

#[then("the customization dialog closes without changing the target")]
fn closed_with_same_target(world: &mut ApplicationWorld) {
    customization_closed(world);
    open_tools_customization(world);
    transaction_steps::target_is_empty(world);
    close_customization_button(world);
}

#[test]
fn shift_selection_keeps_the_range_endpoint_visible_and_the_target_clean() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    row_selection::shift_down(&mut world);
    adjacent_source_commands_highlighted(&mut world);
    assert!(!world.window_text("Customize").contains("ZoomWindow"));
    for _ in 0..48 {
        row_selection::shift_down(&mut world);
    }
    world.pump(Vec::new());
    assert!(
        world.window_text("Customize").contains("ZoomWindow"),
        "expected the final command to be visible after extending the selection"
    );
    assert!(!world.run_is_enabled("Apply"));
    assert!(!world.run_is_enabled("Cancel"));
    close_customization_button(&mut world);
    closed_with_same_target(&mut world);
    world.close();
}

#[test]
fn remove_continues_at_the_next_staged_command_after_an_upward_selection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    let staged = ["2-DDrawing", "2DSketches", "3DAnnotations", "3-PointArc"];
    for label in staged {
        add_command(&mut world, label);
    }
    assert_eq!(target_rows(&world), staged);
    click_in_customize(&mut world, "2DSketches");
    super::row_selection::shift_arrow(&mut world, ViewportPhysicalKey::UpArrow);
    expect_target_selections(&mut world, &staged[..2]);

    click_in_customize(&mut world, "<");
    assert_eq!(target_rows(&world), &staged[2..]);
    expect_target_selection(&mut world, staged[2]);
    click_in_customize(&mut world, "<");
    assert_eq!(target_rows(&world), &staged[3..]);
    expect_target_selection(&mut world, staged[3]);
    click_in_customize(&mut world, "<");
    assert!(target_rows(&world).is_empty());
    super::transaction_steps::expect_target_pending(&world, true);
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn secondary_click_focuses_source_commands_for_keyboard_selection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    capture_plain_source_rows(&mut world);
    let item = source_rows(&world)
        .into_iter()
        .find(|row| row.text == "2DSketches")
        .expect("expected 2D Sketches in the source list");
    world.right_click(diagnostics::pointer_pixels(item.center()));
    expect_source_selection(&mut world, Some("2DSketches"), &["2DSketches"]);
    world.key(ViewportPhysicalKey::A, true);
    expect_source_selection(
        &mut world,
        None,
        &["<Separator>", "2DSketches", "3DAnnotations"],
    );
    assert!(target_rows(&world).is_empty());
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn secondary_click_focuses_staged_commands_for_keyboard_selection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    stage_two_commands(&mut world);
    click_in_customize(&mut world, "<Separator>");
    let item = target_row_runs(&world)
        .into_iter()
        .find(|row| row.text == "2DSketches")
        .expect("expected staged 2D Sketches");
    world.right_click(diagnostics::pointer_pixels(item.center()));
    expect_target_selection(&mut world, "2DSketches");
    world.key(ViewportPhysicalKey::A, true);
    expect_target_selections(&mut world, &["2DSketches", "3DAnnotations"]);
    assert_eq!(target_rows(&world), ["2DSketches", "3DAnnotations"]);
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn source_page_keys_select_the_edges_of_the_visible_command_page() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    capture_plain_source_rows(&mut world);
    let rows: Vec<_> = source_rows(&world)
        .into_iter()
        .map(|row| row.text.clone())
        .collect();
    assert_eq!(rows.first().unwrap(), "<Separator>");
    let last = rows.last().expect("expected at least one visible command");
    assert_ne!(last, "2DSketches");
    assert!(!rows.iter().any(|label| label == "ZoomWindow"));

    click_in_customize(&mut world, "2DSketches");
    world.key(ViewportPhysicalKey::PageDown, false);
    expect_source_selection(&mut world, Some(last), &[last, "<Separator>"]);
    world.key(ViewportPhysicalKey::PageUp, false);
    expect_source_selection(&mut world, Some("<Separator>"), &[last, "<Separator>"]);
    assert!(target_rows(&world).is_empty());
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    click_in_customize(&mut world, "Close");
    world.close();
}

#[given("ribbon customization is open in the MCAD Software for the Part Tools target")]
fn open_tools_draft(world: &mut ApplicationWorld) {
    if world.state().document.is_none() {
        world.add_part();
    }
    empty_tools_customization(world);
}

#[given("\"2D Sketches\" and \"3D Annotations\" are staged in that order")]
fn stage_pair(world: &mut ApplicationWorld) {
    stage_two_commands(world);
    staged_rows_unchanged(world);
}

#[given("both staged rows have Large and Text unchecked")]
#[then("both rows remain in their original order with their display options unchecked")]
#[then("both staged rows and their display options remain unchanged")]
fn staged_rows_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2DSketches", "3DAnnotations"]);
    both_options_clear(world);
}

#[when("the user selects \"2D Sketches\" in the target list and presses Page Down")]
fn page_down_in_target(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    click_in_customize(world, "2DSketches");
    world.key(ViewportPhysicalKey::PageDown, false);
}

#[when("the user presses Page Up")]
fn page_up_in_list(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::PageUp, false);
}

#[then("\"3D Annotations\" is highlighted")]
fn staged_annotations_selected(world: &mut ApplicationWorld) {
    expect_target_selection(world, "3DAnnotations");
}

#[then("\"2D Sketches\" is highlighted")]
fn staged_sketches_selected(world: &mut ApplicationWorld) {
    expect_target_selection(world, "2DSketches");
}

#[when("the user selects \"2-D Drawing\" and extends the selection down to \"2D Sketches\"")]
fn select_first_target_pair(world: &mut ApplicationWorld) {
    let rows = target_row_runs(world);
    let [first, second, ..] = rows.as_slice() else {
        panic!("expected at least two staged rows");
    };
    assert_eq!(first.text, "2-DDrawing");
    assert_eq!(second.text, "2DSketches");
    world.click(diagnostics::pointer_pixels(first.center()));
    super::row_selection::shift_down(world);
}

#[then("the first two rows remain selected")]
fn first_target_pair_remains_selected(world: &mut ApplicationWorld) {
    expect_target_selections(world, &["2-DDrawing", "2DSketches"]);
}

#[when("the user selects \"3D Annotations\" in the target list")]
fn select_target_annotations(world: &mut ApplicationWorld) {
    let row = target_row_runs(world)
        .into_iter()
        .find(|row| row.text == "3DAnnotations")
        .expect("expected staged 3D Annotations");
    world.click(diagnostics::pointer_pixels(row.center()));
}

#[then("\"2-D Drawing\" remains available in the source list")]
fn drawing_remains_available(world: &mut ApplicationWorld) {
    assert!(
        source_rows(world)
            .iter()
            .any(|row| row.text == "2-DDrawing")
    );
}

const THREE_TARGET_ROWS: [&str; 3] = ["2-DDrawing", "2DSketches", "3DAnnotations"];

#[given("\"2-D Drawing\", \"2D Sketches\" and \"3D Annotations\" are staged in that order")]
fn stage_three_target_commands(world: &mut ApplicationWorld) {
    for command in THREE_TARGET_ROWS {
        add_command(world, command);
    }
}

#[given("all three staged rows have Large and Text unchecked")]
#[then("the three rows remain in their original order with their display options unchecked")]
fn three_rows_keep_display_options(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), THREE_TARGET_ROWS);
    for row in THREE_TARGET_ROWS {
        for option in ["Large", "Text"] {
            assert!(!row_option_checked(world, row, option), "{option} on {row}");
        }
    }
}

#[when("the user selects \"3D Annotations\" in the target list and presses Home")]
fn home_from_target_annotations(world: &mut ApplicationWorld) {
    select_target_annotations(world);
    world.key(ViewportPhysicalKey::Home, false);
}

#[then("\"2-D Drawing\" is highlighted")]
fn staged_drawing_selected(world: &mut ApplicationWorld) {
    expect_target_selection(world, "2-DDrawing");
}

#[when("the user clicks the command name in the target row")]
fn click_only_target_name(world: &mut ApplicationWorld) {
    let rows = target_row_runs(world);
    let [row] = rows.as_slice() else {
        panic!("expected one staged command name");
    };
    world.click(diagnostics::pointer_pixels(row.center()));
}

#[when("the user presses Space")]
#[when("the user presses Space again")]
fn press_space(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Space, false);
}

#[when("the user selects 2-D Drawing and clicks the add button")]
fn add_drawing_for_size_edit(world: &mut ApplicationWorld) {
    capture_plain_source_rows(world);
    add_command(world, "2-DDrawing");
}

#[then("2-D Drawing is staged and 2D Sketches is selected in the available list")]
fn drawing_staged_before_size_edit(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2-DDrawing"]);
    expect_source_selection(world, Some("2DSketches"), &["2DSketches"]);
}

#[when("the user clicks the staged 2-D Drawing Large checkbox")]
fn check_staged_drawing_large(world: &mut ApplicationWorld) {
    toggle_row_option(world, "2-DDrawing", "Large");
}

#[then("Large is checked and Text is unchecked for 2-D Drawing")]
#[then("Large remains checked for 2-D Drawing")]
#[then("2-D Drawing still has Large checked")]
fn staged_drawing_is_large(world: &mut ApplicationWorld) {
    assert!(row_option_checked(world, "2-DDrawing", "Large"));
    assert!(!row_option_checked(world, "2-DDrawing", "Text"));
}

#[then("2D Sketches is added after 2-D Drawing")]
fn staged_sketches_follow_drawing(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2-DDrawing", "2DSketches"]);
}

#[then("3D Annotations is selected in the available list")]
fn annotations_ready_for_transfer(world: &mut ApplicationWorld) {
    expect_source_selection(world, Some("3DAnnotations"), &["3DAnnotations"]);
}

#[then("3D Annotations is added after 2D Sketches")]
fn staged_annotations_follow_sketches(world: &mut ApplicationWorld) {
    assert_eq!(
        target_rows(world),
        ["2-DDrawing", "2DSketches", "3DAnnotations"]
    );
}

#[when("the user cancels customization and reopens it for Part Tools")]
fn cancel_and_reopen_tools_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "Cancel");
    assert!(diagnostics::window_geometry("Customize").is_none());
    empty_tools_customization(world);
}

#[when("the user presses Delete")]
fn press_delete(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Delete, false);
}

#[then("the command row remains selected")]
fn only_target_remains_selected(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2DSketches"]);
    expect_target_selection(world, "2DSketches");
}

#[then("its Large and Text checkboxes remain unchecked")]
fn only_target_display_is_unchanged(world: &mut ApplicationWorld) {
    for option in ["Large", "Text"] {
        assert!(!row_option_checked(world, "2DSketches", option));
    }
}

#[then("the customization dialog closes without applying the staged commands")]
#[then("the customization dialog closes without applying the staged entries")]
fn customization_closed_without_apply(world: &mut ApplicationWorld) {
    customization_closed(world);
    no_user_commands(world);
}

#[when("the user selects 2D Sketches in the available command list")]
fn select_sketches_after_capturing_plain_rows(world: &mut ApplicationWorld) {
    capture_plain_source_rows(world);
    click_in_customize(world, "2DSketches");
}

pub(super) fn capture_plain_source_rows(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}

#[when("the user presses Ctrl+A while the available command list has focus")]
#[when("the user presses Ctrl+A while the source list has focus")]
fn select_all_visible_source_rows(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::A, true);
}

#[then("every currently visible available-command row is highlighted")]
fn all_visible_source_rows_highlighted(world: &mut ApplicationWorld) {
    expect_source_selection(world, None, &["<Separator>", "3DAnnotations"]);
}

#[when("the user clicks 3D Annotations in the available command list")]
fn select_annotations_in_source(world: &mut ApplicationWorld) {
    click_in_customize(world, "3DAnnotations");
}

#[then("3D Annotations is highlighted")]
#[then("the other currently visible available-command rows are not highlighted")]
fn only_annotations_is_highlighted_in_source(world: &mut ApplicationWorld) {
    expect_source_selection(
        world,
        Some("3DAnnotations"),
        &["<Separator>", "3DAnnotations"],
    );
}

pub(super) fn expect_source_selection(
    world: &mut ApplicationWorld,
    only: Option<&str>,
    required: &[&str],
) {
    world.hover([800, 900]);
    let x = world.run_in_window("Customize", ">").minimum[0] - 45.;
    let rows = sample_rows(source_rows(world), x);
    for expected in required {
        assert!(rows.iter().any(|(label, _)| label == expected));
    }
    let current = super::super::capture_framebuffer(&mut world.app().window.native);
    let baseline = world
        .last_capture
        .as_ref()
        .expect("expected plain source rows");
    assert_eq!(current.extent, baseline.extent);
    for (label, point) in rows {
        assert_eq!(
            pixel(&current, point) != pixel(baseline, point),
            only.is_none_or(|selected| label == selected),
            "source highlight for {label:?}"
        );
    }
}

pub(super) fn expect_target_selection(world: &mut ApplicationWorld, selected: &str) {
    expect_target_selections(world, &[selected]);
}

pub(super) fn expect_target_selections(world: &mut ApplicationWorld, selected: &[&str]) {
    world.hover([800, 900]);
    let rows = target_row_runs(world);
    let last = rows.last().expect("expected staged rows");
    let x = last.minimum[0] + 105.;
    let plain = point_at([x, last.center()[1] + 30.]);
    let rows = sample_rows(rows, x);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    for (label, point) in rows {
        assert_eq!(
            pixel(&capture, point) != pixel(&capture, plain),
            selected.contains(&label.as_str()),
            "target highlight for {label:?}"
        );
    }
}

fn sample_rows(rows: Vec<&diagnostics::RenderedRun>, x: f32) -> Vec<(String, [usize; 2])> {
    rows.into_iter()
        .map(|row| (row.text.clone(), point_at([x, row.center()[1]])))
        .collect()
}

fn point_at(position: [f32; 2]) -> [usize; 2] {
    diagnostics::pointer_pixels(position).map(|axis| axis as usize)
}

fn pixel(capture: &ViewportCapture, [x, y]: [usize; 2]) -> &[u8] {
    let start = (y * capture.extent[0] as usize + x) * 4;
    &capture.rgba[start..start + 4]
}
