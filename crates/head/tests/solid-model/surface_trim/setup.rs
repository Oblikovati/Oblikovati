// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Surface Trim scenarios (surface-trim-01..05, #3918).
use super::super::super::*;
use oblikovati_gui::GuiSurfaceTrimPreview;
use oblikovati_model::feature::surface_trim::SurfaceTrimSide;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const TRIM_WINDOW: &str = "Trim Surface";
const THICKEN_WINDOW: &str = "Thicken/Offset";
const REMOVE_HIGHLIGHT: [f32; 4] = [0.95, 0.26, 0.18, 0.55];

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([340, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn create_zero_offset_vertical_surface(world: &mut ApplicationWorld) {
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

    // Switch output mode to Surface (OffsetSrf).
    let surf_btn = world.run_in_window(THICKEN_WINDOW, "Surface").center();
    world.click(super::super::super::diagnostics::pointer_pixels(surf_btn));
    world.pump(Vec::new());
    world.pump(Vec::new());

    // Select the vertical side face at x = 0 crossing XZ Plane (y in [-1.25, +1.25], z in [0, 2.0]).
    pan_clear_of_panel(world);
    let candidates = [[0.0, 0.5, 1.0], [0.0, -0.5, 1.0], [0.0, 0.0, 1.0]];
    let mut picked = false;
    for pt in candidates {
        if let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) {
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world
                .state()
                .thicken_offset_preview
                .as_ref()
                .is_some_and(|preview| preview.request.faces.len() == 1)
            {
                picked = true;
                break;
            }
        }
    }
    assert!(
        picked,
        "expected vertical side face at x=0 to be selected in Thicken/Offset, preview={:?}",
        world.state().thicken_offset_preview
    );

    // Set Distance to 0 mm (zero-offset surface).
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
    world.type_text("0 mm");
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
    world.pump(Vec::new());

    // Accept OffsetSrf1.
    let ok = world.run_in_window(THICKEN_WINDOW, "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none(),
        "expected Thicken/Offset panel to close after creating OffsetSrf1"
    );
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected 1 surface body after zero-offset surface creation, saw {:?}",
        browser.surface_bodies
    );
}

pub(super) fn hide_original_solid(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "Solid1") {
        let folder = world.run("SolidBodies(1)").clone();
        let arrow = super::super::super::diagnostics::pointer_pixels([
            folder.minimum[0] - 10.0,
            folder.center()[1],
        ]);
        world.click(arrow);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    let solid_row = world.run("Solid1").center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(solid_row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let vis = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "Visibility")
        .unwrap_or_else(|| {
            panic!(
                "expected Visibility menu item on Solid1, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(vis));
    world.pump(Vec::new());
    world.pump(Vec::new());
    // Verify Solid1 is hidden in the browser and only the planar vertical surface remains visible.
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser.bodies[0].visible,
        "expected Solid1 visibility to be false, saw {:?}",
        browser.bodies[0]
    );
    let mut min_x = f32::INFINITY;
    let mut max_x = f32::NEG_INFINITY;
    for tri in &world.app().window.scene_triangles {
        for p in tri.positions {
            min_x = min_x.min(p[0]);
            max_x = max_x.max(p[0]);
        }
    }
    assert!(
        (max_x - min_x) < 0.02,
        "expected original 4cm solid hidden so only planar vertical surface remains, saw x=[{min_x}, {max_x}]"
    );
}

pub(super) fn open_trim_surface_ribbon(world: &mut ApplicationWorld) {
    if let Some(trim) = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Trim")
        .map(|run| run.center())
    {
        world.click(super::super::super::diagnostics::pointer_pixels(trim));
        world.pump(Vec::new());
        world.pump(Vec::new());
    } else {
        // When narrow layout collapses the Surface panel, open its flyout first.
        let surface = world.run("Surface").center();
        world.click(super::super::super::diagnostics::pointer_pixels(surface));
        world.pump(Vec::new());
        world.pump(Vec::new());
        let trim = world.run("Trim").center();
        world.click(super::super::super::diagnostics::pointer_pixels(trim));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(TRIM_WINDOW).is_some(),
        "expected Trim Surface dialog to open, saw {}",
        world.text()
    );
}

pub(super) fn pick_browser_xz_origin_plane(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "XZPlane") {
        let origin = world.run("Origin").clone();
        let arrow = super::super::super::diagnostics::pointer_pixels([
            origin.minimum[0] - 10.0,
            origin.center()[1],
        ]);
        world.click(arrow);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    let xz = world.run("XZPlane").center();
    world.click(super::super::super::diagnostics::pointer_pixels(xz));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_positive_y_surface_region(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // Use the actual x plane of the visible vertical surface (x=0 or x=-4), spanning y in [-1.25, +1.25], z in [0, 2.0].
    let surface_x = f64::from(
        world
            .app()
            .window
            .scene_triangles
            .first()
            .expect("expected visible vertical surface triangles")
            .positions[0][0],
    );
    let candidates = [
        [surface_x, 0.7, 1.0],
        [surface_x, 0.5, 1.0],
        [surface_x, 0.85, 1.2],
    ];
    for pt in candidates {
        if let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) {
            world.click(pixel);
            world.pump(Vec::new());
            world.pump(Vec::new());
            world.pump(Vec::new());
            if world
                .state()
                .surface_trim_preview
                .as_ref()
                .is_some_and(|preview| preview.result == GuiSurfaceTrimPreview::Ready)
            {
                return;
            }
        }
    }
    panic!(
        "expected click on y > 0 surface region to produce Ready Surface Trim preview, saw {:?}",
        world.state().surface_trim_preview
    );
}

pub(super) fn click_trim_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(TRIM_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn trim_control_enabled(world: &ApplicationWorld, label: &str) -> bool {
    world.run_in_window(TRIM_WINDOW, label).alpha == u8::MAX
}

pub(super) fn highlighted_region_mean_y(world: &mut ApplicationWorld) -> Option<f32> {
    let mut sum = 0.0_f32;
    let mut count = 0_usize;
    for tri in &world.app().window.scene_triangles {
        if colors_close(tri.color, REMOVE_HIGHLIGHT) {
            for pt in tri.positions {
                sum += pt[1];
                count += 1;
            }
        }
    }
    (count > 0).then(|| sum / count as f32)
}

pub(super) fn current_surface_area(world: &mut ApplicationWorld) -> f64 {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    eval.ordered_bodies()
        .filter(|body| body.is_surface())
        .flat_map(|body| {
            body.face_reference_keys(session.history(), guid)
                .unwrap_or_default()
                .into_iter()
                .map(|face| body.face_area(session.history(), guid, &face).unwrap())
        })
        .sum()
}

pub(super) fn edit_trim1_feature(world: &mut ApplicationWorld) {
    let row = world.run("Trim1").center();
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
                "expected EditFeature menu item on Trim1, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(TRIM_WINDOW).is_some(),
        "expected Trim Surface dialog to reopen for Trim1, saw {}",
        world.text()
    );
}

pub(super) fn expect_remove_side(world: &mut ApplicationWorld, expected: SurfaceTrimSide) {
    let preview = world
        .state()
        .surface_trim_preview
        .expect("expected Surface Trim preview");
    assert_eq!(preview.result, GuiSurfaceTrimPreview::Ready);
    assert_eq!(preview.request.remove_side, Some(expected));
}

fn colors_close(a: [f32; 4], b: [f32; 4]) -> bool {
    (0..4).all(|i| (a[i] - b[i]).abs() < 0.03)
}
