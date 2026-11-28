// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Combine scenarios (combine-01..combine-08).
use super::super::super::*;
use oblikovati_gui::GuiCombinePreview;
use oblikovati_model::feature::combine::CombineOperation;
use oblikovati_viewport::ViewportNavigation;

pub(super) const COMBINE_WINDOW: &str = "Combine";
pub(super) const EDIT_ERROR_WINDOW: &str = "Edit Combine Feature";

pub(super) fn open_combine_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("combine fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    world.pump(Vec::new());
}

pub(super) fn open_combine_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Combine")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_some(),
        "expected Combine panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_combine_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(COMBINE_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_clear_on_row(world: &mut ApplicationWorld, row_label: &str) {
    let row_y = world.run_in_window(COMBINE_WINDOW, row_label).center()[1];
    let (origin, size) = super::super::super::diagnostics::window_geometry(COMBINE_WINDOW).unwrap();
    let clear = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "Clear"
                && run.center()[0] >= origin[0]
                && run.center()[0] <= origin[0] + size[0]
                && (run.center()[1] - row_y).abs() <= 14.0
        })
        .min_by(|a, b| {
            (a.center()[1] - row_y)
                .abs()
                .total_cmp(&(b.center()[1] - row_y).abs())
        })
        .unwrap_or_else(|| {
            panic!(
                "expected Clear button on {row_label} row in Combine panel, saw {}",
                world.window_text(COMBINE_WINDOW)
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(clear));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([360, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn click_solid_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point)
        .unwrap_or_else(|error| panic!("model pixel for {point:?}: {error}"));
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_block_and_first_cylinder(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // Block top face at (2.0, 1.5, 2.0) is clear of both cylinders (r=0.5 at x=1 and x=3).
    click_solid_point(world, [2.0, 1.5, 2.0]);
    // First cylinder top face protrudes above block at (1.0, 1.5, 3.0).
    click_solid_point(world, [1.0, 1.5, 3.0]);
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview after selecting base and tool");
    assert!(
        preview.request.base_body.is_some() && preview.request.tool_bodies.len() == 1,
        "expected 1 base and 1 tool after picking block and cyl1, saw {preview:?}"
    );
}

pub(super) fn select_block_and_both_cylinders(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    click_solid_point(world, [2.0, 1.5, 2.0]);
    click_solid_point(world, [1.0, 1.5, 3.0]);
    click_solid_point(world, [3.0, 1.5, 3.0]);
    let preview = world
        .state()
        .combine_preview
        .expect("expected combine preview after selecting base and two tools");
    assert!(
        preview.request.base_body.is_some() && preview.request.tool_bodies.len() == 2,
        "expected 1 base and 2 tools after picking block and both cylinders, saw {preview:?}"
    );
}

pub(super) fn set_keep_toolbodies(world: &mut ApplicationWorld, wanted: bool) {
    let current = world
        .state()
        .combine_preview
        .is_some_and(|preview| preview.request.keep_toolbodies);
    if current != wanted {
        click_combine_control(world, "KeepToolbodies");
    }
    let after = world
        .state()
        .combine_preview
        .expect("expected combine preview")
        .request
        .keep_toolbodies;
    assert_eq!(after, wanted, "expected Keep Toolbodies == {wanted}");
}

pub(super) fn accept_combine_and_wait(world: &mut ApplicationWorld) {
    click_combine_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_none(),
        "expected Combine panel to close after OK, preview={:?}",
        world.state().combine_preview
    );
}

pub(super) fn double_click_combine1(world: &mut ApplicationWorld) {
    let center = world.run("Combine1").center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(COMBINE_WINDOW).is_some(),
        "expected Combine panel after double-clicking Combine1, saw {}",
        world.text()
    );
}

pub(super) fn create_combine1_join_keep_tool(world: &mut ApplicationWorld) {
    open_combine_ribbon(world);
    select_block_and_first_cylinder(world);
    click_combine_control(world, "Join");
    set_keep_toolbodies(world, true);
    assert_eq!(
        world.state().combine_preview.unwrap().result,
        GuiCombinePreview::Ready
    );
    accept_combine_and_wait(world);
}

pub(super) fn create_combine1_cut_two_tools(world: &mut ApplicationWorld) {
    open_combine_ribbon(world);
    select_block_and_both_cylinders(world);
    set_keep_toolbodies(world, false);
    click_combine_control(world, "Cut");
    let preview = world.state().combine_preview.unwrap();
    assert_eq!(preview.request.operation, CombineOperation::Cut);
    assert_eq!(preview.result, GuiCombinePreview::Ready);
    accept_combine_and_wait(world);
}
