// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Thread scenarios (thread-01..thread-10a).
use super::super::super::*;
use oblikovati_gui::GuiThreadPreview;
use oblikovati_martensite::ThreadableFaceInfo;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey, ViewportTriangle};

pub(super) const THREAD_WINDOW: &str = "Thread";
pub(super) const EDIT_FAILED_MODAL: &str = "Edit Thread Feature Failed";

pub(super) fn click_thread_ribbon_raw(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Thread")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_thread_panel(world: &mut ApplicationWorld) {
    click_thread_ribbon_raw(world);
    assert!(
        super::super::super::diagnostics::window_geometry(THREAD_WINDOW).is_some(),
        "expected Thread panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_thread_control(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let center = world.run_in_window(THREAD_WINDOW, &squeezed).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn reopen_thread1(world: &mut ApplicationWorld) {
    let row = world.run("Thread1").center();
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
        super::super::super::diagnostics::window_geometry(THREAD_WINDOW).is_some(),
        "expected Thread panel to reopen for Thread1"
    );
}

pub(super) fn replace_thread_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let row = world.run_in_window(THREAD_WINDOW, &squeezed).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(THREAD_WINDOW).unwrap();
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

pub(super) fn open_thread_combo(world: &mut ApplicationWorld, row_label: &str) {
    let squeezed = super::super::super::world::squeezed(row_label);
    let row = world.run_in_window(THREAD_WINDOW, &squeezed).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(THREAD_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_thread_combo(world: &mut ApplicationWorld, row_label: &str, choice: &str) {
    open_thread_combo(world, row_label);
    let squeezed = super::super::super::world::squeezed(choice);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected combo option {choice:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn popup_option_texts(world: &ApplicationWorld) -> Vec<String> {
    world
        .frame
        .runs
        .iter()
        .filter(|run| world.frame.is_popup_row(run))
        .map(|run| run.text.clone())
        .collect()
}

pub(super) fn accept_thread_and_wait(world: &mut ApplicationWorld) {
    click_thread_control(world, "OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THREAD_WINDOW).is_none(),
        "expected Thread window to close after OK, preview={:?}",
        world.state().thread_preview
    );
}

pub(super) fn set_checkbox(world: &mut ApplicationWorld, label: &str, wanted: bool) {
    let squeezed = super::super::super::world::squeezed(label);
    let current = world.checked_run(world.run_in_window(THREAD_WINDOW, &squeezed));
    if current != wanted {
        click_thread_control(world, label);
    }
}

pub(super) fn thread_overlay_triangles(world: &mut ApplicationWorld) -> Vec<ViewportTriangle> {
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .copied()
        .filter(|tri| tri.color == crate::part_scene::THREAD_OVERLAY_COLOR)
        .collect()
}

pub(super) fn overlay_span_extent(triangles: &[ViewportTriangle]) -> f32 {
    if triangles.is_empty() {
        return 0.0;
    }
    let mut min = [f32::INFINITY; 3];
    let mut max = [f32::NEG_INFINITY; 3];
    for tri in triangles {
        for p in tri.positions {
            for axis in 0..3 {
                min[axis] = min[axis].min(p[axis]);
                max[axis] = max[axis].max(p[axis]);
            }
        }
    }
    (0..3).map(|axis| max[axis] - min[axis]).fold(0.0, f32::max)
}

pub(super) fn overlay_min_coord(triangles: &[ViewportTriangle], axis: usize) -> f32 {
    triangles
        .iter()
        .flat_map(|tri| tri.positions)
        .map(|p| p[axis])
        .fold(f32::INFINITY, f32::min)
}

pub(super) fn inspect_matching_faces(
    world: &mut ApplicationWorld,
    predicate: impl Fn(&ThreadableFaceInfo) -> bool,
) -> Vec<(Vec<u8>, ThreadableFaceInfo, Vec<[f64; 3]>)> {
    let document = world.active_id();
    let app = world.app();
    let workspace = app.owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let evaluation = app.owner.part_scenes_current(document).unwrap();
    let mut out = Vec::new();
    for body in evaluation.ordered_bodies() {
        let Ok(meshes) =
            body.named_face_meshes(session.history(), guid, crate::part_scene::DISPLAY_MESH)
        else {
            continue;
        };
        for mesh in meshes {
            let Ok(Some(info)) = body.threadable_face_info(session.history(), guid, &mesh.key)
            else {
                continue;
            };
            if !predicate(&info) {
                continue;
            }
            let points: Vec<[f64; 3]> = mesh
                .triangles
                .iter()
                .map(|tri| {
                    [
                        (tri[0][0] + tri[1][0] + tri[2][0]) / 3.0,
                        (tri[0][1] + tri[1][1] + tri[2][1]) / 3.0,
                        (tri[0][2] + tri[1][2] + tri[2][2]) / 3.0,
                    ]
                })
                .collect();
            out.push((mesh.key, info, points));
        }
    }
    out
}

pub(super) fn click_threadable_face(
    world: &mut ApplicationWorld,
    predicate: impl Fn(&ThreadableFaceInfo) -> bool + Copy,
) {
    let matches = inspect_matching_faces(world, predicate);
    assert!(
        !matches.is_empty(),
        "expected at least one matching threadable face in active part"
    );
    let target_key = matches[0].0.clone();
    let sample_points: Vec<[f64; 3]> = matches[0]
        .2
        .iter()
        .step_by((matches[0].2.len() / 24).max(1))
        .copied()
        .collect();
    let views: [(f64, f64); 6] = [
        (0.0, 0.0),
        (0.7, 0.4),
        (-0.7, 0.4),
        (1.4, -0.4),
        (-1.4, -0.4),
        (0.0, 0.9),
    ];
    let mut trace = Vec::new();
    for (yaw, pitch) in views {
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Home { margin: 0.35 },
        )
        .unwrap();
        if yaw != 0.0 || pitch != 0.0 {
            crate::HeadWindow::navigate_view(
                &mut world.app().window.native,
                ViewportNavigation::Orbit { yaw, pitch },
            )
            .unwrap();
        }
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Pan([340, 0]),
        )
        .unwrap();
        world.pump(Vec::new());
        let panel_rect = super::super::super::diagnostics::window_geometry(THREAD_WINDOW);
        for point in &sample_points {
            let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *point)
            else {
                continue;
            };
            if let Some((origin, size)) = panel_rect {
                let p0 = super::super::super::diagnostics::pointer_pixels(origin);
                let p1 = super::super::super::diagnostics::pointer_pixels([
                    origin[0] + size[0],
                    origin[1] + size[1],
                ]);
                if pixel[0] >= p0[0] - 8
                    && pixel[0] <= p1[0] + 8
                    && pixel[1] >= p0[1] - 8
                    && pixel[1] <= p1[1] + 8
                {
                    continue;
                }
            }
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            world.pump(Vec::new());
            let picked_match = world
                .state()
                .thread_pick
                .as_ref()
                .is_some_and(|pick| pick.face == target_key);
            let preview_ready = world
                .state()
                .thread_preview
                .as_ref()
                .is_some_and(|p| p.result == GuiThreadPreview::Ready);
            if picked_match && preview_ready {
                return;
            }
            trace.push(format!(
                "orbit=({yaw},{pitch}) pt={point:?} px={pixel:?} pick={:?}",
                world.state().thread_pick.map(|p| p.diameter_mm)
            ));
        }
    }
    panic!("failed to pick target threadable face via viewport click; trace={trace:#?}");
}

pub(super) fn create_tapered_thread1(world: &mut ApplicationWorld) {
    open_thread_panel(world);
    click_threadable_face(world, |info| info.conical && !info.internal);
    accept_thread_and_wait(world);
}

pub(super) fn create_internal_9mm_full_depth_thread1(world: &mut ApplicationWorld) {
    open_thread_panel(world);
    click_threadable_face(world, |info| {
        info.internal
            && !info.conical
            && (info.diameter_cm * 10.0 - 9.0).abs() < 0.3
            && (info.full_depth_cm * 10.0 - 2.0).abs() < 0.3
    });
    select_thread_combo(world, "Thread Type", "ANSI Metric M Profile");
    select_thread_combo(world, "Size", "9");
    select_thread_combo(world, "Designation", "M9x1.25");
    select_thread_combo(world, "Class", "6H");
    set_checkbox(world, "Full Depth", true);
    accept_thread_and_wait(world);
}
