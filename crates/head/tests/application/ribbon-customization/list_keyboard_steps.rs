// SPDX-License-Identifier: GPL-2.0-only
//! Inspect keyboard selection in the two rendered ribbon command lists.
use super::list_steps::source_rows;
use super::transaction_steps::{target_row_runs, target_rows};
use super::*;
use oblikovati_viewport::{ViewportCapture, ViewportPhysicalKey};

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

#[when("the user clicks the command name in the target row")]
fn click_only_target_name(world: &mut ApplicationWorld) {
    let rows = target_row_runs(world);
    let [row] = rows.as_slice() else {
        panic!("expected one staged command name");
    };
    world.click(diagnostics::pointer_pixels(row.center()));
}

#[when("the user presses Space")]
fn press_space(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Space, false);
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
