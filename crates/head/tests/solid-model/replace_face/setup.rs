// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Surface Replace Face scenarios (replace-face-01..04, #3918).
use super::super::super::*;
use super::viewport::{
    click_face_around_point, click_model_point, ensure_point_visible_and_unoccluded,
    find_body_face_center, find_highest_planar_solid_face_center, find_torus_face_candidates,
    pan_clear_of_panel,
};
use oblikovati_gui::{FeatureOutputMode, GuiReplaceFacePreview, GuiRevolvePreview};
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const REPLACE_FACE_WINDOW: &str = "Replace Face";
pub(super) const FAILURE_MODAL_WINDOW: &str = "Create Replace Face feature failed";

pub(super) fn open_replace_face_ribbon(world: &mut ApplicationWorld) {
    if world.frame.runs.iter().any(|run| run.text == "ReplaceFace") {
        world.click_text("Replace Face");
    } else {
        world.click_text("Surface");
        world.pump(Vec::new());
        world.pump(Vec::new());
        world.click_text("Replace Face");
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(REPLACE_FACE_WINDOW).is_some(),
        "expected Replace Face panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_replace_face_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(REPLACE_FACE_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_modal_control(world: &mut ApplicationWorld, label: &str) {
    world.pump(Vec::new());
    let center = world.run_in_window(FAILURE_MODAL_WINDOW, label).center();
    let point = super::super::super::diagnostics::pointer_pixels(center);
    let motion =
        oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Motion {
            position: point,
            modifiers: Default::default(),
        });
    let press =
        oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
            button: oblikovati_viewport::ViewportButton::Left,
            pressed: true,
            position: point,
            modifiers: Default::default(),
        });
    let release =
        oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
            button: oblikovati_viewport::ViewportButton::Left,
            pressed: false,
            position: point,
            modifiers: Default::default(),
        });
    world.pump(vec![motion]);
    world.pump(vec![motion, press]);
    world.pump(vec![motion, release]);
    world.pump(Vec::new());
}

pub(super) fn create_offset_surface_1mm_on_block_top(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    click_thicken_control(world, "Surface");
    pan_clear_of_panel(world);
    // In body-properties-start.opd, block spans [-4..0, -2.5..0, 0..2.0].
    assert!(click_model_point(world, [-2.0, -1.25, 2.0]));
    replace_thicken_field(world, "Distance", "1 mm");
    click_thicken_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Thicken/Offset").is_none(),
        "expected Thicken/Offset panel to close after creating OffsetSrf1"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "OffsetSrf1"),
        "expected OffsetSrf1 in browser: {:?}",
        browser.features
    );
}

pub(super) fn create_offset_surface_1mm_on_exposed_planar_face(world: &mut ApplicationWorld) {
    let target_point = find_highest_planar_solid_face_center(world);
    open_thicken_ribbon(world);
    click_thicken_control(world, "Surface");
    pan_clear_of_panel(world);
    assert!(
        click_face_around_point(world, target_point),
        "expected click on exposed planar block face near {target_point:?}"
    );
    replace_thicken_field(world, "Distance", "1 mm");
    click_thicken_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Thicken/Offset").is_none(),
        "expected Thicken/Offset panel to close after creating offset surface"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser.surface_bodies.is_empty(),
        "expected at least one surface body after offset surface"
    );
}

pub(super) fn revolve_sketch9_full_turn_torus(world: &mut ApplicationWorld) {
    world.click_text("Revolve");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Revolution").is_some(),
        "expected Revolution panel to open for Sketch9, saw {}",
        world.text()
    );
    if world
        .state()
        .revolve_preview
        .as_ref()
        .is_some_and(|p| p.result == GuiRevolvePreview::MissingAxis)
    {
        pick_sketch9_centerline_axis(world);
    }
    // Choose New Solid so Sketch9 revolves into a separate toroidal solid body.
    if world
        .state()
        .revolve_preview
        .as_ref()
        .is_some_and(|p| p.request.output != FeatureOutputMode::NewSolid)
    {
        let center = world.run_in_window("Revolution", "NewSolid").center();
        world.click(super::super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    let preview = world
        .state()
        .revolve_preview
        .expect("expected Revolution preview for Sketch9");
    assert_eq!(
        preview.result,
        GuiRevolvePreview::Ready,
        "expected Sketch9 Revolution preview Ready, got {preview:?}"
    );
    let ok = world.run_in_window("Revolution", "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Revolution").is_none(),
        "expected Revolution panel to close after creating torus solid"
    );
}

fn pick_sketch9_centerline_axis(world: &mut ApplicationWorld) {
    let doc = world.active_id();
    let axis_point = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let snapshot = session.snapshot();
        let eval = app.owner.part_scenes_current(doc).unwrap();
        let datums = eval.datum_evaluation();
        let resolved = snapshot
            .require_converged_resolved_sketch_with_datums(
                oblikovati_model::document_sketch::DocumentSketchId(9),
                &datums,
            )
            .expect("expected resolved Sketch9");
        let geom = resolved.sketch.geometry();
        let line_id = geom
            .centerline_ids()
            .into_iter()
            .next()
            .unwrap_or(oblikovati_model::sketch::SketchEntityId(60));
        let line = geom
            .lines()
            .iter()
            .find(|l| l.id == line_id)
            .expect("expected Sketch9 axis line");
        let a = resolved
            .plane
            .to_model(geom.point(line.endpoints[0]).unwrap().position)
            .unwrap();
        let b = resolved
            .plane
            .to_model(geom.point(line.endpoints[1]).unwrap().position)
            .unwrap();
        [
            0.5 * (a[0] + b[0]),
            0.5 * (a[1] + b[1]),
            0.5 * (a[2] + b[2]),
        ]
    };
    let pixel = ensure_point_visible_and_unoccluded(world, axis_point);
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    let state = world.state().revolve_preview;
    assert!(
        state.as_ref().is_some_and(|p| p.request.axis.is_some()),
        "expected Sketch9 axis click at {axis_point:?} -> pixel {pixel:?} to pick axis; occlusions={:?}, preview={state:?}",
        super::super::super::diagnostics::window_occlusions()
    );
}

pub(super) fn select_block_top_and_offset_surface(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let _ = crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Orbit {
            yaw: 0.45,
            pitch: 0.22,
        },
    );
    world.pump(Vec::new());
    let solid_top = find_body_face_center(world, false, |c| (c[2] - 2.0).abs() < 0.05);
    let surface_face = find_body_face_center(world, true, |_| true);
    select_face_for_slot(world, &[solid_top], true);
    click_replace_face_control(world, "SelectNewFaces");
    select_face_for_slot(world, &[surface_face], false);
}

pub(super) fn select_torus_and_offset_surface(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let torus_points = find_torus_face_candidates(world);
    let surface_face = find_body_face_center(world, true, |_| true);
    select_face_for_slot(world, &torus_points, true);
    click_replace_face_control(world, "SelectNewFaces");
    select_face_for_slot(world, &[surface_face], false);
    let preview = world
        .state()
        .replace_face_preview
        .expect("expected Replace Face preview state");
    assert_eq!(preview.request.existing_faces.len(), 1);
    assert_eq!(preview.request.new_faces.len(), 1);
    assert!(
        matches!(preview.result, GuiReplaceFacePreview::Failed(_)),
        "expected torus -> separate surface preview to fail, got {:?}",
        preview.result
    );
}

fn select_face_for_slot(world: &mut ApplicationWorld, points: &[[f64; 3]], existing: bool) {
    let before = if existing {
        existing_count(world)
    } else {
        new_count(world)
    };
    let offsets = [
        [0.0, 0.0, 0.0],
        [0.05, 0.0, 0.0],
        [-0.05, 0.0, 0.0],
        [0.0, 0.05, 0.0],
        [0.0, -0.05, 0.0],
    ];
    for &point in points {
        for d in offsets {
            let candidate = [point[0] + d[0], point[1] + d[1], point[2] + d[2]];
            if !click_model_point(world, candidate) {
                continue;
            }
            let after = if existing {
                existing_count(world)
            } else {
                new_count(world)
            };
            if after > before {
                return;
            }
        }
    }
    panic!(
        "expected viewport click near {:?} to populate {} selector; preview={:?}",
        points.first(),
        if existing {
            "Existing Faces"
        } else {
            "New Faces"
        },
        world.state().replace_face_preview
    );
}

pub(super) fn existing_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .replace_face_preview
        .as_ref()
        .map_or(0, |p| p.request.existing_faces.len())
}

pub(super) fn new_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .replace_face_preview
        .as_ref()
        .map_or(0, |p| p.request.new_faces.len())
}

fn open_thicken_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Thicken/Offset")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn click_thicken_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window("Thicken/Offset", label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn replace_thicken_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window("Thicken/Offset", label).clone();
    let (origin, size) =
        super::super::super::diagnostics::window_geometry("Thicken/Offset").unwrap();
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
