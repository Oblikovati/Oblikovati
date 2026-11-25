// SPDX-License-Identifier: GPL-2.0-only
//! Switch the two published sketch Parts with their original saved geometry.
use super::*;
use oblikovati_api::{client::Client, wire::SketchArgs};
use oblikovati_model::sketch::SketchPlane;
use std::path::Path;

const EMPTY: &str = "fixtures/cad/planar-sketch-empty-xy.opd";
const PROFILE: &str = "fixtures/cad/revolve/revolve-profile.opd";

#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` and `fixtures/cad/revolve/revolve-profile.opd` are open in that order"
)]
fn open_saved_sketch_parts(world: &mut ApplicationWorld) {
    let root = super::super::scenario_root();
    let ids = [EMPTY, PROFILE].map(|fixture| {
        let path = root.join(fixture);
        world.open_fixture(&path);
        let id = world.active_id();
        let state = world.state();
        let storage = &state.document_storage[&id];
        assert_eq!(storage.path.as_deref().map(Path::new), Some(path.as_path()));
        assert!(!storage.unsaved_edits);
        id
    });
    assert!(world.state().documents.iter().map(|row| row.id).eq(ids));
    world.parts.extend(ids);
}

pub(super) fn is_saved_sketch_part(world: &mut ApplicationWorld, id: u64) -> bool {
    world.state().document_storage[&id]
        .path
        .as_deref()
        .is_some_and(|path| {
            let path = Path::new(path);
            [EMPTY, PROFILE]
                .into_iter()
                .any(|fixture| path.file_name() == Path::new(fixture).file_name())
        })
}

pub(super) fn expect_sketch_lines(world: &mut ApplicationWorld, count: usize) {
    world.expect_browser_sketch();
    let entities = Client::new(world.app().owner.document_host())
        .sketch()
        .entities(&SketchArgs::default())
        .unwrap()
        .entities
        .unwrap_or_default();
    let lines = entities
        .iter()
        .filter(|entity| entity.kind == "line")
        .count();
    assert_eq!(lines, count);
    assert!(
        entities
            .iter()
            .all(|entity| matches!(entity.kind.as_str(), "line" | "point"))
    );
}

#[given("the second part's sketch is visible")]
fn second_sketch_is_visible(world: &mut ApplicationWorld) {
    world.expect_active(second(world));
    let browser = world.state().browser.unwrap();
    let [sketch] = browser.sketches.as_slice() else {
        panic!("expected one saved sketch in the second Part");
    };
    if !sketch.visible {
        let name = &sketch.name;
        use super::super::workspace_layout::browser_feature::{choose_menu_row, right_click_row};
        right_click_row(world, name);
        choose_menu_row(world, "Visibility");
    }
    assert!(world.state().browser.unwrap().sketches[0].visible);
}

pub(super) fn published_rectangle_visible(world: &mut ApplicationWorld) -> bool {
    use super::super::{capture_framebuffer, diagnostics::window_occlusions, rectangle_visibility};
    let [a, b] = [[-0.6, 0.], [0.6, 0.5]].map(|point| {
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), point).unwrap()
    });
    let corners = [
        [a[0].min(b[0]), a[1].min(b[1])],
        [a[0].max(b[0]), a[1].max(b[1])],
    ];
    let area = world.viewport_bounds();
    let capture = capture_framebuffer(&mut world.app().window.native);
    rectangle_visibility::rectangle_visible_in_view(&capture, corners, area, &window_occlusions())
        .expect("expected the published profile to project inside the clear viewport")
}
