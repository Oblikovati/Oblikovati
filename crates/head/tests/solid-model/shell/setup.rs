// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Shell scenarios (shell-01..shell-07).
use super::super::super::*;
use oblikovati_gui::GuiShellV2Preview;
use oblikovati_martensite::PrismSurfaceKind;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const SHELL_WINDOW: &str = "Shell";

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([380, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn open_shell_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Shell");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SHELL_WINDOW).is_some(),
        "expected Shell panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_shell_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(SHELL_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_shell_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window(SHELL_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(SHELL_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.65,
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

pub(super) fn replace_override_thickness(world: &mut ApplicationWorld, value: &str) {
    let pill = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| {
            let (origin, size) =
                super::super::super::diagnostics::window_geometry(SHELL_WINDOW).unwrap();
            let c = run.center();
            c[0] >= origin[0]
                && c[0] <= origin[0] + size[0]
                && c[1] >= origin[1]
                && c[1] <= origin[1] + size[1]
                && (run.text == "1Face" || run.text == "SelectFace" || run.text == "PickFace...")
        })
        .unwrap_or_else(|| {
            panic!(
                "expected override face pill in Shell window, saw {}",
                world.window_text(SHELL_WINDOW)
            )
        })
        .clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(SHELL_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        pill.center()[1],
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

pub(super) fn accept_shell_and_wait(world: &mut ApplicationWorld) {
    click_shell_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SHELL_WINDOW).is_none(),
        "expected Shell window to close after OK, preview={:?}",
        world.state().shell_v2_preview
    );
}

pub(super) fn reopen_shell(world: &mut ApplicationWorld, name: &str) {
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
        .unwrap_or_else(|| panic!("expected EditFeature popup item, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SHELL_WINDOW).is_some(),
        "expected Shell panel to reopen for {name}"
    );
}

pub(super) fn assert_shell_ready(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .shell_v2_preview
        .expect("expected Shell v2 preview state");
    assert_eq!(
        preview.result,
        GuiShellV2Preview::Ready,
        "expected Ready Shell preview, got {:?}",
        preview.result
    );
}

pub(super) fn solid_volume_and_faces(world: &mut ApplicationWorld) -> (f64, usize) {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let body = eval.ordered_bodies().next().unwrap();
    let vol = body.volume(session.history(), guid).unwrap();
    let faces = body
        .face_reference_keys(session.history(), guid)
        .unwrap()
        .len();
    (vol, faces)
}

pub(super) fn collect_faces(
    world: &mut ApplicationWorld,
) -> Vec<(Vec<u8>, [f64; 3], PrismSurfaceKind)> {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let body = eval.ordered_bodies().next().unwrap();
    let mut out = Vec::new();
    for key in body.face_reference_keys(session.history(), guid).unwrap() {
        if let Ok(topo) = body.face_topology(session.history(), guid, &key) {
            out.push((key, topo.point, topo.surface));
        }
    }
    out
}

pub(super) fn click_top_planar_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let mut planar: Vec<_> = collect_faces(world)
        .into_iter()
        .filter(|(_, _, kind)| *kind == PrismSurfaceKind::Plane)
        .collect();
    planar.sort_by(|a, b| b.1[2].total_cmp(&a.1[2]));
    let before = world
        .state()
        .shell_v2_pick
        .as_ref()
        .map_or(0, |p| p.sequence);
    for (_, center, _) in planar {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, center) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .shell_v2_pick
            .as_ref()
            .is_some_and(|p| p.sequence > before)
        {
            return;
        }
    }
    panic!("expected clicking the top planar face to register a Shell pick");
}

pub(super) fn click_side_planar_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let faces = collect_faces(world);
    let (min_x, max_x, min_y, max_y, min_z, max_z) = faces.iter().fold(
        (
            f64::INFINITY,
            f64::NEG_INFINITY,
            f64::INFINITY,
            f64::NEG_INFINITY,
            f64::INFINITY,
            f64::NEG_INFINITY,
        ),
        |(x0, x1, y0, y1, z0, z1), (_, p, _)| {
            (
                x0.min(p[0]),
                x1.max(p[0]),
                y0.min(p[1]),
                y1.max(p[1]),
                z0.min(p[2]),
                z1.max(p[2]),
            )
        },
    );
    let (cx, cy, mid_z) = (
        (min_x + max_x) * 0.5,
        (min_y + max_y) * 0.5,
        (min_z + max_z) * 0.5,
    );
    let mut sides: Vec<_> = faces
        .into_iter()
        .filter(|(_, p, kind)| {
            *kind == PrismSurfaceKind::Plane && (p[2] - mid_z).abs() < (max_z - min_z) * 0.35
        })
        .collect();
    // Prefer outer side faces furthest from the block's actual XY center (front-most first).
    sides.sort_by(|a, b| {
        let da = (a.1[0] - cx).hypot(a.1[1] - cy);
        let db = (b.1[0] - cx).hypot(b.1[1] - cy);
        db.total_cmp(&da)
    });
    let before = world
        .state()
        .shell_v2_pick
        .as_ref()
        .map_or(0, |p| p.sequence);
    for (_, center, _) in &sides {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *center) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .shell_v2_pick
            .as_ref()
            .is_some_and(|p| p.sequence > before)
        {
            return;
        }
    }
    panic!("expected clicking a side planar face to register a Shell override pick");
}

pub(super) fn click_rounded_face(world: &mut ApplicationWorld) {
    let rounded: Vec<_> = collect_faces(world)
        .into_iter()
        .filter(|(_, _, kind)| *kind != PrismSurfaceKind::Plane)
        .collect();
    assert!(
        !rounded.is_empty(),
        "expected at least one rounded face on the filleted fixture"
    );
    let before = world
        .state()
        .shell_v2_pick
        .as_ref()
        .map_or(0, |p| p.sequence);
    for yaw in [
        0.0,
        std::f64::consts::PI * 0.5,
        std::f64::consts::PI,
        std::f64::consts::PI * 1.5,
    ] {
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Home { margin: 0.35 },
        )
        .unwrap();
        if yaw > 0.0 {
            crate::HeadWindow::navigate_view(
                &mut world.app().window.native,
                ViewportNavigation::Orbit { yaw, pitch: 0.0 },
            )
            .unwrap();
        }
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Pan([380, 0]),
        )
        .unwrap();
        world.pump(Vec::new());
        for (_, center, _) in &rounded {
            let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *center)
            else {
                continue;
            };
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world
                .state()
                .shell_v2_pick
                .as_ref()
                .is_some_and(|p| p.sequence > before)
            {
                return;
            }
        }
    }
    panic!("expected clicking the rounded fillet face to register a Shell pick");
}
