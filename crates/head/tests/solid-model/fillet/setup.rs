// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Fillet scenarios (fillet-01..fillet-14).
use super::super::super::*;
use oblikovati_gui::GuiFilletV2Preview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const FILLET_WINDOW: &str = "Fillet";

pub(super) fn open_fillet_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("fillet fixture {source:?}: {error}"));
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

pub(super) fn open_fillet_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Fillet");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(FILLET_WINDOW).is_some(),
        "expected Fillet panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_fillet_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(FILLET_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn reopen_fillet1(world: &mut ApplicationWorld) {
    let row = world.run("Fillet1").center();
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
        super::super::super::diagnostics::window_geometry(FILLET_WINDOW).is_some(),
        "expected Fillet panel to reopen for Fillet1"
    );
}

pub(super) fn replace_fillet_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window(FILLET_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(FILLET_WINDOW).unwrap();
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

pub(super) fn accept_fillet_and_wait(world: &mut ApplicationWorld) {
    click_fillet_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(FILLET_WINDOW).is_none(),
        "expected Fillet window to close after OK, preview={:?}",
        world.state().fillet_v2_preview
    );
}

fn ctrl_click(world: &mut ApplicationWorld, position: [i32; 2]) {
    use oblikovati_viewport::{ViewportButton, ViewportInput, ViewportModifiers};
    world.hover(position);
    let mods = ViewportModifiers {
        control: true,
        ..Default::default()
    };
    for pressed in [true, false] {
        world.pump(vec![ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed,
            position,
            modifiers: mods,
        })]);
    }
}

pub(super) fn click_any_solid_edge(world: &mut ApplicationWorld, toggle: bool) {
    pan_clear_of_panel(world);
    let points = candidate_edge_midpoints(world);
    let before = world.state().selected_edge_count;
    for pt in points {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
            continue;
        };
        if toggle {
            ctrl_click(world, pixel);
        } else {
            world.click(pixel);
        }
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world.state().selected_edge_count > before {
            return;
        }
    }
    panic!("expected clicking a solid edge to increase selected_edge_count from {before}");
}

pub(super) fn select_three_origin_edges(world: &mut ApplicationWorld) {
    // Orient the camera to face the (-X, -Y, -Z) octant so all three edges at (0, 0, 0) are unoccluded.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Orbit {
            yaw: std::f64::consts::PI,
            pitch: -1.1,
        },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([380, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
    let doc = world.active_id();
    let edge_segments = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        let body = eval.ordered_bodies().next().unwrap();
        let mut edges = Vec::new();
        for key in body.edge_reference_keys(session.history(), guid).unwrap() {
            let segs = body
                .edge_display_segments(session.history(), guid, &key, 0.02)
                .unwrap();
            let a = segs[0][0];
            let b = segs[segs.len() - 1][1];
            let touches_origin = [a, b].iter().any(|p| p[0].hypot(p[1]).hypot(p[2]) < 1e-3);
            if touches_origin {
                edges.push((a, b));
            }
        }
        edges
    };
    assert_eq!(
        edge_segments.len(),
        3,
        "expected 3 edges incident at (0,0,0)"
    );
    for (idx, (a, b)) in edge_segments.into_iter().enumerate() {
        let before = world.state().selected_edge_count;
        for t in [0.5_f64, 0.35, 0.65] {
            let pt = [
                a[0] + (b[0] - a[0]) * t,
                a[1] + (b[1] - a[1]) * t,
                a[2] + (b[2] - a[2]) * t,
            ];
            let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
                continue;
            };
            if idx == 0 {
                world.click(pixel);
            } else {
                ctrl_click(world, pixel);
            }
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world.state().selected_edge_count > before {
                break;
            }
        }
    }
    assert_eq!(world.state().selected_edge_count, 3);
}

pub(super) fn click_body_face_center(world: &mut ApplicationWorld, body_index: usize) {
    pan_clear_of_panel(world);
    let doc = world.active_id();
    let candidates = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        let bodies: Vec<_> = eval.ordered_bodies().collect();
        let body = bodies.get(body_index).copied().unwrap_or_else(|| bodies[0]);
        let mut pts = Vec::new();
        for fkey in body.face_reference_keys(session.history(), guid).unwrap() {
            if let Ok(c) = body.face_bounds_center(session.history(), guid, &fkey) {
                pts.push(c);
            }
        }
        pts
    };
    let before_seq = world
        .state()
        .fillet_v2_pick
        .as_ref()
        .map_or(0, |p| p.sequence);
    for pt in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .fillet_v2_pick
            .as_ref()
            .is_some_and(|p| p.sequence > before_seq)
        {
            return;
        }
    }
    panic!("expected clicking body {body_index} face to produce a Fillet v2 pick");
}

pub(super) fn candidate_edge_midpoints(world: &mut ApplicationWorld) -> Vec<[f64; 3]> {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let mut pts = Vec::new();
    for body in eval.ordered_bodies() {
        for key in body.edge_reference_keys(session.history(), guid).unwrap() {
            if let Ok(segs) = body.edge_display_segments(session.history(), guid, &key, 0.02)
                && !segs.is_empty()
            {
                let mid = segs[segs.len() / 2];
                pts.push([
                    (mid[0][0] + mid[1][0]) * 0.5,
                    (mid[0][1] + mid[1][1]) * 0.5,
                    (mid[0][2] + mid[1][2]) * 0.5,
                ]);
            }
        }
    }
    pts
}

pub(super) fn assert_v2_ready_or_partial(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected fillet v2 preview");
    assert!(
        matches!(
            preview.result,
            GuiFilletV2Preview::Ready | GuiFilletV2Preview::PartialFailure { .. }
        ),
        "expected Ready or PartialFailure preview, got {:?}",
        preview.result
    );
}
