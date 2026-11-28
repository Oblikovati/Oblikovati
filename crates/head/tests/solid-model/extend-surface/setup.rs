// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Surface Extend scenarios (extend-surface-01..06, #3918).
use super::super::super::*;
use oblikovati_gui::{GuiExtendSurfaceExtentMode, GuiExtendSurfacePreview};
use oblikovati_model::feature::extend_surface::ExtendSurfaceMethod;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const EXTEND_WINDOW: &str = "Extend Surface";
const THICKEN_WINDOW: &str = "Thicken/Offset";

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
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

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) -> bool {
    let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
        return false;
    };
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    true
}

pub(super) fn create_1mm_offset_surface(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let thicken = world
        .run_in_window("##ribbon-additional-PartModify", "Thicken/Offset")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(thicken));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_some(),
        "expected Thicken/Offset panel to open, saw {}",
        world.text()
    );
    let surface_btn = world.run_in_window(THICKEN_WINDOW, "Surface").center();
    world.click(super::super::super::diagnostics::pointer_pixels(
        surface_btn,
    ));
    world.pump(Vec::new());
    world.pump(Vec::new());
    pan_clear_of_panel(world);
    // Top planar face of body-properties-start.opd at z = 2.0 cm, x in [-4, 0], y in [-2.5, 0].
    assert!(click_model_point(world, [-2.0, -1.25, 2.0]));
    let ok_btn = world.run_in_window(THICKEN_WINDOW, "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok_btn));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none(),
        "expected Thicken/Offset panel to close after committing OffsetSrf1"
    );
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected 1 offset surface body (Srf1), saw {:?}",
        browser.surface_bodies
    );
}

pub(super) fn open_extend_surface_ribbon(world: &mut ApplicationWorld) {
    let direct = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Extend")
        .map(|run| run.center());
    if let Some(center) = direct {
        world.click(super::super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
    } else {
        let surface_panel = world.run("Surface").center();
        world.click(super::super::super::diagnostics::pointer_pixels(
            surface_panel,
        ));
        world.pump(Vec::new());
        world.pump(Vec::new());
        let extend = world.run("Extend").center();
        world.click(super::super::super::diagnostics::pointer_pixels(extend));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(EXTEND_WINDOW).is_some(),
        "expected Extend Surface panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_extend_control(world: &mut ApplicationWorld, label: &str) {
    click_extend_control_with_index(world, label, false);
}

pub(super) fn click_extend_control_with_index(
    world: &mut ApplicationWorld,
    label: &str,
    from_end: bool,
) {
    let needle = super::super::super::world::squeezed(label);
    let (origin, size) = super::super::super::diagnostics::window_geometry(EXTEND_WINDOW).unwrap();
    let matches: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == needle
                && run.center()[0] >= origin[0]
                && run.center()[0] <= origin[0] + size[0]
                && run.center()[1] >= origin[1]
                && run.center()[1] <= origin[1] + size[1]
        })
        .collect();
    let run = if from_end {
        matches.last().copied()
    } else {
        matches.first().copied()
    }
    .unwrap_or_else(|| {
        panic!(
            "expected control {label:?} in {EXTEND_WINDOW}, saw {}",
            world.window_text(EXTEND_WINDOW)
        )
    });
    world.click(super::super::super::diagnostics::pointer_pixels(
        run.center(),
    ));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn set_edge_chain(world: &mut ApplicationWorld, wanted: bool) {
    let current = world
        .state()
        .extend_surface_preview
        .as_ref()
        .is_none_or(|p| p.request.edge_chain);
    if current != wanted {
        click_extend_control(world, "Edge Chain");
    }
}

pub(super) fn replace_distance_field(world: &mut ApplicationWorld, value: &str) {
    let (origin, size) = super::super::super::diagnostics::window_geometry(EXTEND_WINDOW).unwrap();
    // Pick the "Distance" property label (below the "Distance" / "To" extent mode row).
    let rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "Distance"
                && run.center()[0] >= origin[0]
                && run.center()[0] <= origin[0] + size[0]
                && run.center()[1] >= origin[1]
                && run.center()[1] <= origin[1] + size[1]
        })
        .collect();
    let row = rows
        .last()
        .expect("expected Distance field label in Extend Surface panel");
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

pub(super) fn select_upper_boundary_edge(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // Offset surface sits at z = 2.1 cm, spanning x in [-4.0, 0.0], y in [-2.5, 0.0].
    assert!(click_model_point(world, [-2.0, 0.0, 2.1]));
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected extend surface preview after clicking upper boundary edge");
    assert_eq!(
        preview.request.edges.len(),
        1,
        "expected 1 boundary edge selected, saw {preview:?}"
    );
}

pub(super) fn select_right_boundary_edge(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // Right edge of the rectangular offset surface at x = 0.0, y in [-2.5, 0.0], z = 2.1 cm.
    assert!(click_model_point(world, [0.0, -1.25, 2.1]));
}

pub(super) fn surface_y_span_at_z21(world: &mut ApplicationWorld) -> f32 {
    let mut min_y = f32::INFINITY;
    let mut max_y = f32::NEG_INFINITY;
    for tri in &world.app().window.scene_triangles {
        let on_offset = tri.positions.iter().all(|p| (p[2] - 2.1).abs() < 0.02);
        if !on_offset {
            continue;
        }
        for p in tri.positions {
            min_y = min_y.min(p[1]);
            max_y = max_y.max(p[1]);
        }
    }
    if min_y.is_finite() && max_y.is_finite() {
        max_y - min_y
    } else {
        0.0
    }
}

pub(super) fn ready_preview_area(world: &mut ApplicationWorld) -> f64 {
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected extend surface preview");
    match preview.result {
        GuiExtendSurfacePreview::Ready { area_mm2 } => area_mm2,
        other => panic!("expected Ready preview, saw {other:?}"),
    }
}

pub(super) fn double_click_extend1_in_browser(world: &mut ApplicationWorld) {
    let center = world.run("Extend1").center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    if super::super::super::diagnostics::window_geometry(EXTEND_WINDOW).is_none() {
        let row = world.run("Extend1").center();
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
                    "expected EditFeature popup item for Extend1, saw {}",
                    world.text()
                )
            })
            .center();
        world.click(super::super::super::diagnostics::pointer_pixels(edit));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(EXTEND_WINDOW).is_some(),
        "expected Extend Surface panel after reopening Extend1, saw {}",
        world.text()
    );
}

pub(super) fn assert_distance_10mm_and_stretch(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .extend_surface_preview
        .expect("expected Extend1 edit preview");
    assert_eq!(
        preview.request.extent_mode,
        GuiExtendSurfaceExtentMode::Distance
    );
    assert_eq!(preview.request.distance, Some(1.0));
    assert_eq!(preview.request.method, ExtendSurfaceMethod::Stretch);
}
