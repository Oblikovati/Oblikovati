// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Draft scenarios (draft-01..draft-06, #3918).
use super::super::super::*;
use oblikovati_gui::GuiDraftPreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const DRAFT_WINDOW: &str = "Draft";
pub(super) const CREATE_FAILURE_WINDOW: &str = "Create Face Draft";
pub(super) const EDIT_FAILURE_WINDOW: &str = "Edit Face Draft";

pub(super) fn click_draft_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Draft")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_some(),
        "expected Draft panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_draft_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(DRAFT_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_modal_control(world: &mut ApplicationWorld, window: &str, label: &str) {
    let center = world.run_in_window(window, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_draft_angle(world: &mut ApplicationWorld, value: &str) {
    let row = world.run_in_window(DRAFT_WINDOW, "DraftAngle").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(DRAFT_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
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

pub(super) fn reopen_face_draft1(world: &mut ApplicationWorld) {
    let row = world.run("FaceDraft1").center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let edit = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "EditFeature")
        .unwrap_or_else(|| panic!("expected EditFeature popup item, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_some(),
        "expected Draft panel to reopen for FaceDraft1"
    );
}

pub(super) fn pick_xz_origin_plane(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "XZPlane") {
        let origin = world.run("Origin").clone();
        let arrow = super::super::super::diagnostics::pointer_pixels([
            origin.minimum[0] - 10.,
            origin.center()[1],
        ]);
        world.click(arrow);
        world.pump(Vec::new());
        if !world.frame.runs.iter().any(|run| run.text == "XZPlane") {
            world.click_text("Origin");
            world.pump(Vec::new());
        }
    }
    let xz = super::super::super::diagnostics::pointer_pixels(world.run("XZPlane").center());
    world.click(xz);
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

pub(super) fn frame_solid_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([400, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn click_top_face_for_pull_direction(world: &mut ApplicationWorld) {
    frame_solid_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [
        [cx, cy, z_top],
        [cx + 0.15, cy, z_top],
        [cx - 0.15, cy, z_top],
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
            .draft_preview
            .as_ref()
            .is_some_and(|p| p.request.pull_direction.is_some())
        {
            return;
        }
    }
    panic!(
        "expected top-face click to set Draft pull direction; preview={:?}",
        world.state().draft_preview
    );
}

pub(super) fn click_shelled_rim_and_outer_y0_face(world: &mut ApplicationWorld) {
    frame_solid_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let z_top = f64::from(max[2]);
    // 1. Pick the 2 mm planar rim at Z = 20 mm (2.0 cm) for Pull Direction.
    let rim_points = [
        [cx, f64::from(min[1]) + 0.1, z_top],
        [f64::from(max[0]) - 0.1, cy, z_top],
        [cx, f64::from(max[1]) - 0.1, z_top],
        [f64::from(min[0]) + 0.1, cy, z_top],
    ];
    let mut picked_rim = false;
    for point in rim_points {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .draft_preview
            .as_ref()
            .is_some_and(|p| p.request.pull_direction.is_some())
        {
            picked_rim = true;
            break;
        }
    }
    assert!(
        picked_rim,
        "expected rim click at Z=20mm to populate Pull Direction; preview={:?}",
        world.state().draft_preview
    );

    // 2. Pick the outer face at Y = 0 mm for Faces.
    let y0_points = [
        [cx, f64::from(min[1]), cz],
        [cx + 0.4, f64::from(min[1]), cz],
        [cx - 0.4, f64::from(min[1]), cz],
    ];
    for point in y0_points {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .draft_preview
            .as_ref()
            .is_some_and(|p| !p.request.faces.is_empty())
        {
            return;
        }
    }
    // Orbit to show the -Y face directly if hidden from Home view.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Orbit {
            yaw: std::f64::consts::FRAC_PI_2,
            pitch: 0.0,
        },
    )
    .unwrap();
    world.pump(Vec::new());
    for point in y0_points {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .draft_preview
            .as_ref()
            .is_some_and(|p| !p.request.faces.is_empty())
        {
            return;
        }
    }
    panic!(
        "expected outer Y=0 face click to populate Faces; preview={:?}",
        world.state().draft_preview
    );
}

pub(super) fn click_side_face_for_draft(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let side_points = [
        [f64::from(max[0]), cy, cz],
        [cx, f64::from(min[1]), cz],
        [cx, f64::from(max[1]), cz],
        [f64::from(min[0]), cy, cz],
    ];
    for point in side_points {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .draft_preview
            .as_ref()
            .is_some_and(|p| !p.request.faces.is_empty())
        {
            return;
        }
    }
    panic!(
        "expected side-face click to populate Draft Faces; preview={:?}",
        world.state().draft_preview
    );
}

pub(super) fn accept_draft_and_wait(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().draft_preview.as_ref().map(|p| &p.result),
        Some(&GuiDraftPreview::Ready),
        "expected Ready Draft preview before OK, got {:?}",
        world.state().draft_preview
    );
    click_draft_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(DRAFT_WINDOW).is_none(),
        "expected Draft window to close after OK, preview={:?}",
        world.state().draft_preview
    );
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
