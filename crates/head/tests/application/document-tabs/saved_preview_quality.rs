// SPDX-License-Identifier: GPL-2.0-only
//! Present Windows-style path metadata on a saved Part in the native Linux window.
use super::super::{capture_framebuffer, desktop_input, diagnostics};
use super::*;
use imgui::{DrawCmd, DrawData, internal::RawCast, sys};

const PREVIEW: &str = "##tab-preview";
const RAW_PATH: &str = r"\\?\UNC\Server\Share\oblique-part.opd";
const DISPLAY_PATH: &str = r"\\Server\Share\oblique-part.opd";
const NETWORK_PART: &str = "fixtures/cad/docking-visible-solid-current.opd";

#[given("a saved part document on a mapped or network path is open in the MCAD Software")]
fn open_shared_part(world: &mut ApplicationWorld) {
    world.restore_layout_without_asset_tools();
    let path = super::super::scenario_root().join(NETWORK_PART);
    super::super::viewport_orientation::open_saved_navigation_block(world, &path);
    let saved = world.active_id();
    look_obliquely(world);
    world.previous_document = Some(saved);
    world.other_document = Some(world.add_part());
}

#[when("the user hovers over that document's tab")]
fn hover_shared_tab(world: &mut ApplicationWorld) {
    let (saved, neighbor) = shared_tab_names(world);
    let tab = world.expect_tab(&saved);
    world.strip_left = Some(world.expect_tab(&neighbor).minimum[0]);
    world.observed_counts.push(tab_close_marks(world, &tab));
    hover_tab(world, &tab);
}

#[then("a smooth high-DPI thumbnail preview appears above the tab")]
fn shared_thumbnail(world: &mut ApplicationWorld) {
    let saved = world.previous_document.unwrap();
    let path = world.state().document_storage[&saved].path.clone().unwrap();
    expect_preview(world, &path);
}

#[then(r#"the preview path displays a normalized user-facing path without raw OS prefixes such as "\\?\\" or "\\?\\UNC\\""#)]
fn shared_caption(world: &mut ApplicationWorld) {
    let saved = world.previous_document.unwrap();
    let path = world.state().document_storage[&saved].path.clone().unwrap();
    assert!(
        path.ends_with(NETWORK_PART),
        "expected the published saved path"
    );
    assert_eq!(world.run_in_window(PREVIEW, &path).text, path);
    assert!(!path.starts_with(r"\\?\"));
}

#[then("hovering reveals the tab close control without shifting adjacent tab positions")]
fn shared_close_control(world: &mut ApplicationWorld) {
    let (saved, neighbor) = shared_tab_names(world);
    let tab = world.expect_tab(&saved);
    assert!(tab_close_marks(world, &tab) > world.observed_counts[0]);
    assert_eq!(
        world.expect_tab(&neighbor).minimum[0],
        world.strip_left.unwrap()
    );
    world.save_capture("saved-tab-preview-on-shared-path");
}

fn shared_tab_names(world: &mut ApplicationWorld) -> (String, String) {
    (
        world.document_name(world.previous_document.unwrap()),
        world.document_name(world.other_document.unwrap()),
    )
}

fn hover_tab(world: &mut ApplicationWorld, tab: &diagnostics::RenderedRun) {
    desktop_input::hover(world, diagnostics::pointer_pixels(tab.center()));
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_tab_preview_filters_the_picture_and_normalizes_the_caption() {
    let mut world = ApplicationWorld::new();
    world.restore_layout_without_asset_tools();
    let saved = open_saved_solid(&mut world);
    let neighbor = world.add_part();
    world.pump(Vec::new());
    let name = world.document_name(saved);
    let bounds = world.expect_tab(&name);
    let neighbor_name = world.document_name(neighbor);
    let next = world.expect_tab(&neighbor_name);
    let initial_marks = tab_close_marks(&world, &bounds);
    let saved_path = world.state().document_storage[&saved].path.clone();
    world.app().window.document_path_override = Some((saved, RAW_PATH.into()));
    hover_tab(&mut world, &bounds);
    expect_preview(&mut world, DISPLAY_PATH);
    assert!(!world.shows(RAW_PATH));
    assert!(tab_close_marks(&world, &bounds) > initial_marks);
    assert_eq!(world.expect_tab(&neighbor_name).minimum, next.minimum);
    assert_eq!(world.state().document_storage[&saved].path, saved_path);
    world.save_capture("saved-tab-filtered-network-caption");
    world.close();
}

fn expect_preview(world: &mut ApplicationWorld, path: &str) {
    let caption = world.run_in_window(PREVIEW, path);
    assert_eq!(caption.text, path);
    let image_top = caption.maximum[1] + 10.;
    let (position, size) = window_geometry(PREVIEW).unwrap();
    let (strip, _) = window_geometry(TAB_STRIP).unwrap();
    assert!(position[1] + size[1] <= strip[1] + 2.);
    assert_one_preview_texture();
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(super::home_and_previews::preview_probe::picture_has_detail(
        &capture, position, size, image_top
    ));
}

fn open_saved_solid(world: &mut ApplicationWorld) -> u64 {
    use super::super::workspace_layout::browser_feature::extruded_part_bytes;
    let folder = world.scratch_folder();
    let saved = folder.join("oblique-part.opd");
    std::fs::write(&saved, extruded_part_bytes("Extrusion1", "Profile", 2.)).unwrap();
    world.open_fixture(&saved);
    let document = world.active_id();
    look_obliquely(world);
    document
}

fn look_obliquely(world: &mut ApplicationWorld) {
    let mut camera = super::super::viewport_pan::camera(world);
    camera.eye = [20., 16., 24.];
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
}

fn tab_close_marks(world: &ApplicationWorld, row: &diagnostics::RenderedRun) -> usize {
    world
        .frame
        .marks
        .iter()
        .filter(|mark| {
            let [x, y] = mark.position;
            mark.list == row.list
                && mark.bright
                && (row.maximum[0]..row.maximum[0] + 30.).contains(&x)
                && (row.minimum[1] - 2. ..row.maximum[1] + 2.).contains(&y)
        })
        .count()
}

fn assert_one_preview_texture() {
    let list = diagnostics::window_draw_list(PREVIEW).unwrap();
    // The rendered frame owns the draw lists until the next pump.
    let raw = unsafe { sys::igGetDrawData().as_ref() }.unwrap();
    let frame = unsafe { DrawData::from_raw(raw) };
    let draw = frame.draw_lists().nth(list).unwrap();
    let triangles: usize = draw
        .commands()
        .filter_map(|command| match command {
            DrawCmd::Elements { count, cmd_params }
                if cmd_params.texture_id.id()
                    == oblikovati_gui::GuiTextureKey::TAB_PREVIEW.raw() =>
            {
                Some(count)
            }
            _ => None,
        })
        .sum();
    assert_eq!(triangles, 6, "expected one preview image quad");
}
