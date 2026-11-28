// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Surface Boundary Patch scenarios (boundary-patch-01..06, #3918).
use super::super::super::*;
use oblikovati_gui::GuiBoundaryPatchPreview;
use oblikovati_viewport::{ViewportCamera, ViewportNavigation, ViewportPhysicalKey};

pub(super) const PATCH_WINDOW: &str = "Boundary Patch";
pub(super) const THICKEN_WINDOW: &str = "Thicken/Offset";

pub(super) fn open_fixture_copy(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().expect("expected fixture filename"));
    std::fs::copy(&source, &copy).unwrap_or_else(|e| {
        panic!(
            "copy fixture {} -> {}: {e}",
            source.display(),
            copy.display()
        )
    });
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

pub(super) fn open_boundary_patch_ribbon(world: &mut ApplicationWorld) {
    if world.frame.runs.iter().any(|r| r.text == "BoundaryPatch") {
        let center = world.run("BoundaryPatch").center();
        world.click(super::super::super::diagnostics::pointer_pixels(center));
    } else {
        // When Surface panel is collapsed on narrow ribbon, open Surface group first.
        let surface = world.run("Surface").center();
        world.click(super::super::super::diagnostics::pointer_pixels(surface));
        world.pump(Vec::new());
        let center = world.run("BoundaryPatch").center();
        world.click(super::super::super::diagnostics::pointer_pixels(center));
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(PATCH_WINDOW).is_some(),
        "expected Boundary Patch panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_patch_control(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let center = world.run_in_window(PATCH_WINDOW, &squeezed).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_condition_icon(world: &mut ApplicationWorld, ordinal: u8) {
    let label = match ordinal {
        1 => "G0",
        2 => "G1",
        3 => "G2",
        other => panic!("unexpected condition ordinal {other}"),
    };
    click_patch_control(world, label);
}

pub(super) fn enter_weight_value(world: &mut ApplicationWorld, value: &str) {
    let row = world.run_in_window(PATCH_WINDOW, "Weight").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(PATCH_WINDOW)
        .expect("expected Boundary Patch window geometry");
    let field = super::super::super::diagnostics::pointer_pixels([
        (row.maximum[0] + origin[0] + size[0] - 12.0) * 0.5,
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

pub(super) fn frame_top_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.30 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([340, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn frame_sketch9_clear_of_panel(world: &mut ApplicationWorld) {
    let camera = ViewportCamera {
        eye: [-1.5, 0.25, 300.0],
        target: [-1.5, 0.25, 0.0],
        up: [0.0, 1.0, 0.0],
        scale: 12.0,
        near: 1.0,
        far: 600.0,
        perspective: false,
    };
    world.app().window.set_view_camera(camera).unwrap();
    world.pump(Vec::new());
}

pub(super) fn frame_block_front_iso(world: &mut ApplicationWorld) {
    let camera = ViewportCamera {
        eye: [300.0, -300.0, 300.0],
        target: [-2.0, -1.25, 1.0],
        up: [0.0, 0.0, 1.0],
        scale: 7.5,
        near: 1.0,
        far: 1200.0,
        perspective: false,
    };
    world.app().window.set_view_camera(camera).unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([340, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn frame_block_back_iso(world: &mut ApplicationWorld) {
    let camera = ViewportCamera {
        eye: [-300.0, 300.0, 300.0],
        target: [-2.0, -1.25, 1.0],
        up: [0.0, 0.0, 1.0],
        scale: 7.5,
        near: 1.0,
        far: 1200.0,
        perspective: false,
    };
    world.app().window.set_view_camera(camera).unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([340, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point)
        .unwrap_or_else(|e| panic!("project model point {point:?}: {e}"));
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn ensure_sketch_visible(world: &mut ApplicationWorld, sketch_name: &str) {
    let doc = world.state().document.expect("active part document");
    let sketch_id = world
        .state()
        .browser
        .expect("model browser")
        .sketches
        .iter()
        .find(|s| s.name == sketch_name)
        .unwrap_or_else(|| panic!("expected sketch {sketch_name}"))
        .id;
    world
        .app()
        .owner
        .set_sketch_visibility_from_gui(doc, sketch_id, true)
        .unwrap();
    world.pump(Vec::new());
}

pub(super) fn add_visible_diagonal_to_sketch1(world: &mut ApplicationWorld) {
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .add_entity(&oblikovati_api::wire::AddSketchEntityArgs {
            sketch_index: 0,
            kind: "line".into(),
            points: Some(vec![Some(vec![-0.6, 0.0]), Some(vec![0.6, 0.5])]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

pub(super) fn create_zero_offset_surface_on_face(world: &mut ApplicationWorld, point: [f64; 3]) {
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
        "expected Thicken/Offset panel"
    );
    let surface_btn = world.run_in_window(THICKEN_WINDOW, "Surface").center();
    world.click(super::super::super::diagnostics::pointer_pixels(
        surface_btn,
    ));
    world.pump(Vec::new());
    let row = world.run_in_window(THICKEN_WINDOW, "Distance").clone();
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
    world.type_text("0");
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
    click_model_point(world, point);
    let ok = world.run_in_window(THICKEN_WINDOW, "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none(),
        "expected Thicken/Offset panel to close after zero-offset surface commit"
    );
}

pub(super) fn double_click_boundary_patch1(world: &mut ApplicationWorld) {
    let run = world.run_in_window("Model", "BoundaryPatch1").center();
    let pixel = super::super::super::diagnostics::pointer_pixels(run);
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        pixel, 2,
    ));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(PATCH_WINDOW).is_some(),
        "expected Boundary Patch panel to reopen for Boundary Patch1"
    );
}

pub(super) fn has_ready_preview(world: &mut ApplicationWorld) -> bool {
    world
        .state()
        .boundary_patch_preview
        .as_ref()
        .is_some_and(|p| p.result == GuiBoundaryPatchPreview::Ready)
}

pub(super) fn has_warm_amber_preview_triangles(world: &mut ApplicationWorld) -> bool {
    world.app().window.scene_triangles.iter().any(|tri| {
        let [r, g, b, a] = tri.color;
        r > 0.85 && g > 0.50 && g < 0.75 && b < 0.20 && a < 0.9
    })
}
