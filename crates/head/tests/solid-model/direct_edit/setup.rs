// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Direct Edit scenarios (direct-edit-01..08, #3918).
use super::super::super::*;
use oblikovati_model::feature::direct_edit::{DirectEditMode, DirectEditSelectionFilter};
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const DIRECT_EDIT_WINDOW: &str = "Direct Edit";
pub(super) const MANIPULATOR_WINDOW: &str = "##direct-edit-manipulator";

pub(super) fn open_direct_edit_ribbon(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "DirectEdit")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_some(),
        "expected Direct Edit floating toolbar to open, saw {}",
        world.text()
    );
}

pub(super) fn click_toolbar_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(DIRECT_EDIT_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_popup_item(world: &mut ApplicationWorld, label: &str) {
    let center = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == label)
        .unwrap_or_else(|| panic!("expected popup menu item {label:?}, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
    let mut min = [f32::INFINITY; 3];
    let mut max = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for vertex in tri.positions {
            for axis in 0..3 {
                min[axis] = min[axis].min(vertex[axis]);
                max[axis] = max[axis].max(vertex[axis]);
            }
        }
    }
    (min, max)
}

pub(super) fn scene_extent_max(world: &mut ApplicationWorld) -> f32 {
    let (min, max) = scene_bounds(world);
    (max[0] - min[0]).max(max[1] - min[1]).max(max[2] - min[2])
}

pub(super) fn frame_cylinder_clear_of_toolbar(world: &mut ApplicationWorld) {
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

pub(super) fn click_planar_circular_end_face(world: &mut ApplicationWorld) {
    frame_cylinder_clear_of_toolbar(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [
        [cx, cy, z_top],
        [cx + 0.1, cy, z_top],
        [cx - 0.1, cy, z_top],
        [cx, cy + 0.1, z_top],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .direct_edit_preview
            .as_ref()
            .is_some_and(|p| !p.request.faces.is_empty())
        {
            return;
        }
    }
    panic!(
        "expected click on planar circular end face to select face; win={:?}, pick={:?}, preview={:?}, face_chosen={}, bounds=({min:?}, {max:?})",
        super::super::super::diagnostics::window_geometry(DIRECT_EDIT_WINDOW),
        world.state().direct_edit_pick,
        world.state().direct_edit_preview,
        world.state().model_face_chosen,
    );
}

pub(super) fn click_cylindrical_solid(world: &mut ApplicationWorld) {
    frame_cylinder_clear_of_toolbar(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [[cx, cy, z_top], [cx, cy, cz], [f64::from(max[0]), cy, cz]];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world.state().direct_edit_preview.as_ref().is_some_and(|p| {
            p.request.filter == DirectEditSelectionFilter::Solids && p.request.body.is_some()
        }) {
            return;
        }
    }
    panic!(
        "expected click on cylindrical solid to select solid; preview={:?}",
        world.state().direct_edit_preview
    );
}

pub(super) fn open_selection_menu_and_choose(world: &mut ApplicationWorld, option: &str) {
    let text = world.window_text(DIRECT_EDIT_WINDOW);
    let trigger = if text.contains("Facesv") {
        "Facesv"
    } else {
        "Solidsv"
    };
    click_toolbar_control(world, trigger);
    click_popup_item(world, option);
}

pub(super) fn replace_field_next_to(world: &mut ApplicationWorld, label: &str, value: &str) {
    let run = world.run_in_window(DIRECT_EDIT_WINDOW, label).clone();
    let field =
        super::super::super::diagnostics::pointer_pixels([run.maximum[0] + 34.0, run.center()[1]]);
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

pub(super) fn solid_volume(world: &mut ApplicationWorld) -> f64 {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    eval.ordered_bodies()
        .next()
        .expect("expected active solid body")
        .volume(session.history(), guid)
        .expect("expected analytic solid volume")
}

pub(super) fn setup_mode_and_solid(world: &mut ApplicationWorld, mode: DirectEditMode) {
    open_direct_edit_ribbon(world);
    if mode != DirectEditMode::Move {
        click_toolbar_control(world, mode.label());
    }
    if mode != DirectEditMode::Scale {
        open_selection_menu_and_choose(world, "Solids");
    }
    click_cylindrical_solid(world);
}
