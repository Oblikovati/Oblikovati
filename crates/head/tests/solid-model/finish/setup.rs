// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Finish scenarios (finish-01..05, #3918).
use super::super::super::*;
use oblikovati_martensite::finish_appearance_rgba;
use oblikovati_viewport::ViewportNavigation;

pub(super) const FINISH_WINDOW: &str = "Finish";

pub(super) fn click_finish_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Finish")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(FINISH_WINDOW).is_some(),
        "expected Finish panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_panel_control(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let center = world.run_in_window(FINISH_WINDOW, &squeezed).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_finish_type(world: &mut ApplicationWorld, ordinal: u8) {
    click_panel_control(world, &format!("T{ordinal}"));
}

pub(super) fn select_process_choice(world: &mut ApplicationWorld, choice: &str) {
    let current = world
        .state()
        .finish_preview
        .as_ref()
        .map(|p| p.request.process.clone())
        .unwrap_or_else(|| "Default Coating".to_owned());
    let squeezed_current = super::super::super::world::squeezed(&current);
    let combo_center = world
        .run_in_window(FINISH_WINDOW, &squeezed_current)
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(
        combo_center,
    ));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let squeezed = super::super::super::world::squeezed(choice);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected Process option {choice:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn frame_part_clear_of_panel(world: &mut ApplicationWorld) {
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

pub(super) fn included_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .finish_preview
        .as_ref()
        .map_or(0, |p| p.request.included_faces.len())
}

pub(super) fn excluded_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .finish_preview
        .as_ref()
        .map_or(0, |p| p.request.excluded_faces.len())
}

pub(super) fn click_top_planar_face(world: &mut ApplicationWorld) {
    frame_part_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let before = included_count(world);
    for offset in [[0.0, 0.0], [0.2, 0.0], [-0.2, 0.0], [0.0, 0.2]] {
        let point = [cx + offset[0], cy + offset[1], z_top];
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if included_count(world) > before {
            return;
        }
    }
    panic!(
        "expected top planar face click to select face; preview={:?}",
        world.state().finish_preview
    );
}

pub(super) fn click_cylindrical_body_face(world: &mut ApplicationWorld) {
    frame_part_clear_of_panel(world);
    let doc = world.active_id();
    let target_point = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        eval.ordered_bodies()
            .find_map(|body| {
                let faces = body.face_reference_keys(session.history(), guid).ok()?;
                let has_cyl = faces.iter().any(|face| {
                    body.face_topology(session.history(), guid, face)
                        .is_ok_and(|t| {
                            t.surface == oblikovati_martensite::PrismSurfaceKind::Cylinder
                        })
                });
                if !has_cyl {
                    return None;
                }
                // Pick the highest planar end face of the cylindrical body so it is unobstructed in Home view.
                faces
                    .iter()
                    .filter_map(|face| body.face_bounds_center(session.history(), guid, face).ok())
                    .max_by(|a, b| a[2].total_cmp(&b[2]))
            })
            .expect("expected exposed cylindrical body face in two-bodies.opd")
    };
    let before = included_count(world);
    for offset in [[0.0, 0.0], [0.08, 0.0], [-0.08, 0.0], [0.0, 0.08]] {
        let pt = [
            target_point[0] + offset[0],
            target_point[1] + offset[1],
            target_point[2],
        ];
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if included_count(world) > before {
            return;
        }
    }
    panic!(
        "expected cylindrical body face click at {target_point:?} to select face; preview={:?}",
        world.state().finish_preview
    );
}

pub(super) fn click_visible_block_face(world: &mut ApplicationWorld) -> [f64; 3] {
    frame_part_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let point = [cx, cy, z_top];
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point)
        .expect("expected visible block face pixel");
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    point
}

pub(super) fn has_face_color(world: &mut ApplicationWorld, appearance: &str) -> bool {
    let target = finish_appearance_rgba(appearance);
    world.app().window.scene_triangles.iter().any(|tri| {
        (tri.color[0] - target[0]).abs() < 0.08
            && (tri.color[1] - target[1]).abs() < 0.08
            && (tri.color[2] - target[2]).abs() < 0.08
    })
}

pub(super) fn expand_finishes_browser_folder(world: &mut ApplicationWorld) {
    let model = super::super::super::world::squeezed(&world.window_text("Model"));
    if !model.contains("Finish1") {
        let folder = world.run_in_window("Model", "Finishes").center();
        world.click(super::super::super::diagnostics::pointer_pixels(folder));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}

pub(super) fn double_click_finish1_in_browser(world: &mut ApplicationWorld) {
    expand_finishes_browser_folder(world);
    let label = if super::super::super::world::squeezed(&world.window_text("Model"))
        .contains("Finish1[!]")
    {
        "Finish1[!]"
    } else {
        "Finish1"
    };
    let row = world.run_in_window("Model", label).center();
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
        super::super::super::diagnostics::window_geometry(FINISH_WINDOW).is_some(),
        "expected Finish panel to reopen for Finish1"
    );
}
