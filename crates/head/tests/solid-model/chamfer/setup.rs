// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Chamfer scenarios (chamfer-01..chamfer-10).
use super::super::super::*;
use oblikovati_gui::GuiChamferV2Preview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const CHAMFER_WINDOW: &str = "Chamfer";

pub(super) fn open_chamfer_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("chamfer fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    pan_clear_of_panel(world);
}

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

pub(super) fn open_chamfer_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Chamfer");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_some(),
        "expected Chamfer panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_chamfer_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(CHAMFER_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_chamfer_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window(CHAMFER_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW).unwrap();
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

pub(super) fn accept_chamfer_and_wait(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_none(),
        "expected Chamfer window to close after OK, preview={:?}",
        world.state().chamfer_v2_preview
    );
}

pub(super) fn reopen_chamfer(world: &mut ApplicationWorld, name: &str) {
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
        super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_some(),
        "expected Chamfer panel to reopen for {name}"
    );
}

pub(super) fn assert_chamfer_ready(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .chamfer_v2_preview
        .expect("expected Chamfer v2 preview state");
    assert_eq!(
        preview.result,
        GuiChamferV2Preview::Ready,
        "expected Ready Chamfer preview, got {:?}",
        preview.result
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
        .unwrap()
        .volume(session.history(), guid)
        .unwrap()
}

pub(super) type EdgeCandidate = (Vec<u8>, [f64; 3], [f64; 3], [f64; 3], f64, bool);

pub(super) fn collect_edge_candidates(
    world: &mut ApplicationWorld,
    body_index: usize,
) -> Vec<EdgeCandidate> {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let bodies: Vec<_> = eval.ordered_bodies().collect();
    let body = bodies.get(body_index).copied().unwrap_or(bodies[0]);
    let mut out = Vec::new();
    for key in body.edge_reference_keys(session.history(), guid).unwrap() {
        let Ok(segs) = body.edge_display_segments(session.history(), guid, &key, 0.02) else {
            continue;
        };
        if segs.is_empty() {
            continue;
        }
        let a = segs[0][0];
        let b = segs[segs.len() - 1][1];
        let mid_seg = segs[segs.len() / 2];
        let mid = [
            (mid_seg[0][0] + mid_seg[1][0]) * 0.5,
            (mid_seg[0][1] + mid_seg[1][1]) * 0.5,
            (mid_seg[0][2] + mid_seg[1][2]) * 0.5,
        ];
        let len: f64 = segs
            .iter()
            .map(|[p, q]| {
                ((p[0] - q[0]).powi(2) + (p[1] - q[1]).powi(2) + (p[2] - q[2]).powi(2)).sqrt()
            })
            .sum();
        let closed =
            ((a[0] - b[0]).powi(2) + (a[1] - b[1]).powi(2) + (a[2] - b[2]).powi(2)).sqrt() < 1e-3;
        out.push((key, a, b, mid, len, closed));
    }
    out
}

pub(super) fn click_matching_edge(
    world: &mut ApplicationWorld,
    body_index: usize,
    predicate: impl Fn(&[u8], [f64; 3], [f64; 3], f64, bool) -> bool,
) -> [f64; 3] {
    pan_clear_of_panel(world);
    let edges = collect_edge_candidates(world, body_index);
    for (key, a, b, mid, len, closed) in &edges {
        if !predicate(key, *a, *b, *len, *closed) {
            continue;
        }
        let samples = if *closed {
            vec![*mid]
        } else {
            vec![
                *mid,
                [
                    a[0] + (b[0] - a[0]) * 0.35,
                    a[1] + (b[1] - a[1]) * 0.35,
                    a[2] + (b[2] - a[2]) * 0.35,
                ],
                [
                    a[0] + (b[0] - a[0]) * 0.65,
                    a[1] + (b[1] - a[1]) * 0.65,
                    a[2] + (b[2] - a[2]) * 0.65,
                ],
            ]
        };
        for pt in samples {
            let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
                continue;
            };
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world.state().selected_edge_count > 0 {
                return pt;
            }
        }
    }
    // Try opposite orbit so back-facing vertical/side edges become directly clickable.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Orbit {
            yaw: std::f64::consts::PI * 0.75,
            pitch: 0.0,
        },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([380, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
    for (key, a, b, mid, len, closed) in &edges {
        if !predicate(key, *a, *b, *len, *closed) {
            continue;
        }
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *mid) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world.state().selected_edge_count > 0 {
            return *mid;
        }
    }
    panic!("expected to pick a matching edge on body {body_index}");
}

pub(super) fn click_body_face_for_chamfer(world: &mut ApplicationWorld, body_index: usize) {
    pan_clear_of_panel(world);
    let doc = world.active_id();
    let centers = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        let bodies: Vec<_> = eval.ordered_bodies().collect();
        let body = bodies.get(body_index).copied().unwrap_or(bodies[0]);
        let mut pts = Vec::new();
        for fkey in body.face_reference_keys(session.history(), guid).unwrap() {
            if let Ok(c) = body.face_bounds_center(session.history(), guid, &fkey) {
                pts.push(c);
            }
        }
        pts
    };
    let before = world
        .state()
        .chamfer_v2_pick
        .as_ref()
        .map_or(0, |p| p.sequence);
    for pt in centers {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .chamfer_v2_pick
            .as_ref()
            .is_some_and(|p| p.sequence > before)
        {
            return;
        }
    }
    panic!("expected clicking a body face to satisfy Chamfer face pick");
}

pub(super) fn pick_valid_unmodified_edge(
    world: &mut ApplicationWorld,
    vertical_only: bool,
) -> [f64; 3] {
    let edges = collect_edge_candidates(world, 0);
    // Try candidate edges and verify the live chamfer preview is Ready.
    for orbit_yaw in [
        0.0_f64,
        std::f64::consts::PI * 0.65,
        std::f64::consts::PI * 1.3,
    ] {
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Home { margin: 0.35 },
        )
        .unwrap();
        if orbit_yaw > 0.0 {
            crate::HeadWindow::navigate_view(
                &mut world.app().window.native,
                ViewportNavigation::Orbit {
                    yaw: orbit_yaw,
                    pitch: 0.0,
                },
            )
            .unwrap();
        }
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Pan([380, 0]),
        )
        .unwrap();
        world.pump(Vec::new());
        for (_, a, b, mid, len, closed) in &edges {
            if *closed || *len < 0.3 {
                continue;
            }
            if vertical_only {
                let dz = (b[2] - a[2]).abs();
                let dxy = (b[0] - a[0]).hypot(b[1] - a[1]);
                if dz < len * 0.8 || dxy > 0.05 {
                    continue;
                }
            }
            let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *mid) else {
                continue;
            };
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world
                .state()
                .chamfer_v2_preview
                .as_ref()
                .is_some_and(|p| p.result == GuiChamferV2Preview::Ready)
            {
                return *mid;
            }
        }
    }
    panic!("expected to find an unmodified edge with Ready Chamfer preview");
}
