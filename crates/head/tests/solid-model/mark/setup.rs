// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers and real zero-volume verification for Part Modify Mark (#3918).
use super::super::super::*;
use oblikovati_viewport::ViewportNavigation;

pub(super) const MARK_WINDOW: &str = "Mark";

pub(super) fn click_mark_ribbon_raw(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Mark")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_mark_panel(world: &mut ApplicationWorld) {
    click_mark_ribbon_raw(world);
    assert!(
        super::super::super::diagnostics::window_geometry(MARK_WINDOW).is_some(),
        "expected Mark panel to open, saw {}",
        world.text()
    );
    assert!(
        world.state().mark_preview.is_some(),
        "expected Mark Properties panel preview state, saw {}",
        world.window_text(MARK_WINDOW)
    );
}

pub(super) fn click_mark_control(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let center = world.run_in_window(MARK_WINDOW, &squeezed).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_first_mark_type_combo(world: &mut ApplicationWorld) {
    let row = world.run_in_window(MARK_WINDOW, "MarkType").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(MARK_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_mark_popup_item(world: &mut ApplicationWorld, choice: &str) {
    let squeezed = super::super::super::world::squeezed(choice);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected Mark dropdown option {choice:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_add_geometry_set_plus(world: &mut ApplicationWorld) {
    let header = world.run_in_window(MARK_WINDOW, "MarkGeometrySets").clone();
    let plus = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "+"
                && (run.center()[1] - header.center()[1]).abs() < 16.0
                && run.center()[0] > header.center()[0]
        })
        .unwrap_or_else(|| {
            panic!(
                "expected plus button beside Mark Geometry Sets, saw {}",
                world.window_text(MARK_WINDOW)
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(plus));
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

pub(super) fn click_planar_prism_face(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([300, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let rx = f64::from((max[0] - min[0]) * 0.5);
    let ry = f64::from((max[1] - min[1]) * 0.5);
    let rz = f64::from((max[2] - min[2]) * 0.5);
    let panel_rect = super::super::super::diagnostics::window_geometry(MARK_WINDOW);
    let candidates = [
        [cx, cy, cz + rz * 0.98],
        [cx + rx * 0.85, cy, cz],
        [cx, cy - ry * 0.85, cz],
        [cx + rx * 0.5, cy + ry * 0.5, cz + rz * 0.98],
        [cx, cy, cz],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
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
        if world
            .state()
            .mark_preview
            .as_ref()
            .is_some_and(|preview| !preview.request.faces.is_empty())
        {
            return;
        }
    }
    panic!(
        "expected planar face click to populate Mark Face selector, saw {:?}",
        world.state().mark_preview
    );
}

#[test]
fn mark_on_empty_zero_volume_part_shows_no_volume_tip_and_returns_to_part() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.pump(Vec::new());
    assert_eq!(
        world.state().mark_eligibility,
        oblikovati_gui::GuiMarkEligibility::NoVolume
    );
    click_mark_ribbon_raw(&mut world);
    assert!(
        super::super::super::diagnostics::window_geometry(MARK_WINDOW).is_some(),
        "expected Mark no-volume modal popup"
    );
    let text = super::super::super::world::squeezed(&world.window_text(MARK_WINDOW));
    assert!(
        text.contains("Markfailed.Theselectedcomponenthasnovolume."),
        "expected no-volume failure notice, saw {text}"
    );
    assert!(
        world.state().mark_preview.is_none(),
        "Mark panel must not open on a zero-volume part"
    );
    let ok = world.run_in_window(MARK_WINDOW, "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(MARK_WINDOW).is_none(),
        "expected Mark notice modal to dismiss cleanly"
    );
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.close();
}
