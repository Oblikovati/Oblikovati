// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Thicken/Offset scenarios (thicken-offset-01..09).
use super::super::super::*;
use oblikovati_gui::GuiThickenOffsetPreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const THICKEN_WINDOW: &str = "Thicken/Offset";
pub(super) const EDIT_ERROR_WINDOW: &str = "Edit Thicken";

pub(super) fn open_thicken_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Thicken/Offset")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_some(),
        "expected Thicken/Offset panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_thicken_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(THICKEN_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_thicken_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window(THICKEN_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.68,
        row.center()[1],
    ]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text(value);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_type_combo(world: &mut ApplicationWorld) {
    let row = world.run_in_window(THICKEN_WINDOW, "Type").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.70,
        row.center()[1],
    ]);
    world.click(combo);
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

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) -> bool {
    let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
        return false;
    };
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    true
}

pub(super) fn select_exposed_circular_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // In two-bodies.opd, cylinder extends to z = 3.0 cm centered at (1.0, 1.5).
    assert!(click_model_point(world, [1.0, 1.5, 3.0]));
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected thicken preview after selecting circular end face");
    assert_eq!(
        preview.request.faces.len(),
        1,
        "expected 1 Face selected, saw {preview:?}"
    );
}

pub(super) fn select_block_planar_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // In body-properties-start.opd, block spans [-4..0, -2.5..0, 0..2.0].
    assert!(click_model_point(world, [-2.0, -1.25, 2.0]));
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected thicken preview after selecting planar face");
    assert_eq!(
        preview.request.faces.len(),
        1,
        "expected 1 Face selected on block, saw {preview:?}"
    );
}

pub(super) fn select_cylindrical_side_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // In fillet-circular-edge-candidate.opd, cylinder center (2.0, 0.0), r=0.6, z=0..2.0.
    let candidates = [
        [2.6, 0.0, 1.0],
        [2.42, -0.42, 1.0],
        [2.42, 0.42, 1.0],
        [2.0, -0.6, 1.0],
        [2.0, 0.6, 1.0],
        [1.4, 0.0, 1.0],
    ];
    for point in candidates {
        if !click_model_point(world, point) {
            continue;
        }
        if world
            .state()
            .thicken_offset_preview
            .as_ref()
            .is_some_and(|preview| preview.request.faces.len() == 1)
        {
            return;
        }
    }
    panic!(
        "expected cylindrical side face click to select 1 Face, saw {:?}",
        world.state().thicken_offset_preview
    );
}

pub(super) fn select_rounded_cylinder_side_face(world: &mut ApplicationWorld) {
    // In rounded-cylinder-two-surfaces-candidate.opd, SweepSrf2 is at x=-20, SweepSrf1 at x=-10,
    // and Extrusion2+Fillet1 is on the far right at (8.0, 0.0), r=1.0, z=0..2.0 — already clear
    // of the left-side browser and Thicken/Offset panel in Home view.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.20 },
    )
    .unwrap();
    world.pump(Vec::new());
    let candidates = [
        [9.0, 0.0, 0.9],
        [8.707, -0.707, 0.9],
        [8.707, 0.707, 0.9],
        [8.0, -1.0, 0.9],
        [8.0, 1.0, 0.9],
        [7.0, 0.0, 0.9],
    ];
    for point in candidates {
        if !click_model_point(world, point) {
            continue;
        }
        if world
            .state()
            .thicken_offset_preview
            .as_ref()
            .is_some_and(|preview| preview.request.faces.len() == 2)
        {
            return;
        }
    }
    panic!(
        "expected rounded cylinder side face click to chain 2 Faces, saw {:?}",
        world.state().thicken_offset_preview
    );
}

pub(super) fn accept_thicken_and_wait(world: &mut ApplicationWorld) {
    click_thicken_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none(),
        "expected Thicken/Offset panel to close after OK, preview={:?}",
        world.state().thicken_offset_preview
    );
}

pub(super) fn double_click_feature(world: &mut ApplicationWorld, name: &str) {
    let center = world.run(name).center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    if super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none() {
        let row = world.run(name).center();
        world.right_click(super::super::super::diagnostics::pointer_pixels(row));
        world.pump(Vec::new());
        world.pump(Vec::new());
        let edit = world
            .frame
            .runs
            .iter()
            .rev()
            .find(|run| world.frame.is_popup_row(run) && run.text == "EditFeature")
            .unwrap_or_else(|| {
                panic!(
                    "expected EditFeature popup item for {name}, saw {}",
                    world.text()
                )
            })
            .center();
        world.click(super::super::super::diagnostics::pointer_pixels(edit));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_some(),
        "expected Thicken/Offset panel after reopening {name}, saw {}",
        world.text()
    );
}

pub(super) fn set_checkbox(world: &mut ApplicationWorld, label: &str, current: bool, wanted: bool) {
    if current != wanted {
        click_thicken_control(world, label);
    }
}

pub(super) fn expect_ready_preview(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected thicken/offset preview");
    assert_eq!(
        preview.result,
        GuiThickenOffsetPreview::Ready,
        "expected Ready preview, got {preview:?}"
    );
}
