// SPDX-License-Identifier: GPL-2.0-only
//! Native extrusion acceptance and history before solid-edge measurement.
use super::*;

#[given("Distance A 8 mm and zero taper")]
fn eight_millimeter_preview(world: &mut ApplicationWorld) {
    super::selected_rectangle_preview(world);
    set_eight_millimeter_distance(world);
}

#[cfg(target_os = "linux")]
#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with the rectangular Extrude preview at Distance A 8 mm and zero taper"
)]
fn published_eight_millimeter_preview(world: &mut ApplicationWorld) {
    super::reference_extent::native_fixture_preview(world);
    set_eight_millimeter_distance(world);
}

fn set_eight_millimeter_distance(world: &mut ApplicationWorld) {
    super::replace_extrusion_distance(world, "8");
    world.pump(Vec::new());
    super::positive_distance_is_accepted(world);
    assert!(
        world.state().extrude_preview_active,
        "expected an active straight-prism preview"
    );
}

#[when("OK is clicked then Undo then Redo")]
fn accept_and_replay(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1, "accepted feature count");
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 0, "feature count after Undo");
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1, "feature count after Redo");
}

#[then("Extrusion1 and Solid Bodies(1) return with the shaded solid")]
fn restored_solid_is_visible(world: &mut ApplicationWorld) {
    let browser = world
        .state()
        .browser
        .expect("expected a part model browser");
    assert_eq!(browser.features.len(), 1, "feature rows");
    assert_eq!(browser.features[0].name, "Extrusion1");
    assert_eq!(browser.bodies.len(), 1, "solid body rows");
    world.expect_text("Extrusion1");
    world.expect_text("Solid Bodies");
    assert_eight_millimeter_edges(world);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_solid_fill(&capture, &windows),
        "expected a visible shaded solid after Redo"
    );
}

#[cfg(target_os = "linux")]
#[then("the subsequent Measure result lists two extrusion edges of length 8.000 mm")]
fn measure_two_extrusion_edges(world: &mut ApplicationWorld) {
    use super::super::x11_input;
    use super::viewport_interaction::click_text_native;
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
    click_text_native(world, "Tools");
    let command = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Measure")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("expected the Measure ribbon command");
    let pixel = super::super::diagnostics::pointer_pixels(command.center());
    super::viewport_interaction::click_at(world, pixel);
    assert!(world.state().measure.is_some(), "expected Measure to open");
    let [first, second] = visible_depth_edge_midpoints(world)
        .map(|point| world.app().window.native.view.model_pixel(point).unwrap());
    super::viewport_interaction::click_at(world, first);
    if world.app().window.native.view.uses_wayland() {
        pick_additional_edge(world, second);
    } else {
        x11_input::control_click(second);
        world.pump(Vec::new());
    }
    assert_eq!(world.state().selected_edge_count, 2);
    let text = world.window_text("Measure");
    assert!(
        text.contains("Selection1:Edge Length8.000mm Selection2:Edge Length8.000mm"),
        "Measure should show both 8 mm edges, found {text:?}"
    );
}

#[cfg(target_os = "linux")]
fn pick_additional_edge(world: &mut ApplicationWorld, position: [i32; 2]) {
    use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportModifiers};
    world.hover(position);
    for pressed in [true, false] {
        world.pump(vec![ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            position,
            pressed,
            modifiers: ViewportModifiers {
                control: true,
                ..Default::default()
            },
        })]);
    }
}

#[cfg(target_os = "linux")]
fn visible_depth_edge_midpoints(world: &mut ApplicationWorld) -> [[f64; 3]; 2] {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let geometry = session.snapshot().sketches.sketches()[0].geometry();
    let corners = geometry.lines().iter().flat_map(|line| {
        line.endpoints
            .map(|id| geometry.point(id).unwrap().position)
    });
    let (minimum, maximum) = corners.fold(
        ([f64::INFINITY; 2], [f64::NEG_INFINITY; 2]),
        |(min, max), point| {
            (
                std::array::from_fn(|axis| min[axis].min(point[axis])),
                std::array::from_fn(|axis| max[axis].max(point[axis])),
            )
        },
    );
    // Home looks from positive X, Y and Z. Both corner edges lie on the visible outline.
    [[maximum[0], minimum[1], 0.4], [minimum[0], maximum[1], 0.4]]
}

fn assert_eight_millimeter_edges(world: &mut ApplicationWorld) {
    let edges = named_solid_edges(world);
    let matching = edges
        .iter()
        .filter(|key| edge_is_eight_mm(world, key))
        .count();
    assert!(
        matching >= 4,
        "expected four 8 mm extrusion edges; found {matching}"
    );
}

pub(super) fn named_solid_edges(world: &mut ApplicationWorld) -> Vec<String> {
    let client = oblikovati_api::client::Client::new(world.app().owner.document_host());
    let body = client
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()
        .remove(0);
    body.edges
        .expect("expected named solid edges")
        .into_iter()
        .map(|edge| edge.key)
        .collect()
}

pub(super) fn edge_is_eight_mm(world: &mut ApplicationWorld, key: &str) -> bool {
    use oblikovati_api::{client::Client, wire::MeasureArgs};
    let client = Client::new(world.app().owner.document_host());
    let value = client
        .analysis()
        .measure(&MeasureArgs {
            body_index: 0,
            type_: "length".into(),
            key_a: key.into(),
            ..Default::default()
        })
        .unwrap();
    value.unit == "mm" && (value.value - 8.0).abs() < 1e-6
}

pub(super) fn feature_count(world: &mut ApplicationWorld) -> usize {
    let document = world.active_id();
    world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len()
}

#[test]
fn extrusion_acceptance_and_history_restore_a_visible_solid() {
    let mut world = ApplicationWorld::new();
    eight_millimeter_preview(&mut world);
    accept_and_replay(&mut world);
    restored_solid_is_visible(&mut world);
    world.close();
}

#[test]
fn free_rectangle_extrusion_survives_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    super::rectangle_for_waiting_extrusion(&mut world);
    super::finish_requested_profile(&mut world);
    world.pump(Vec::new());
    super::waiting_extrusion_shows_profile(&mut world);
    super::replace_extrusion_distance(&mut world, "8");
    world.pump(Vec::new());
    accept_and_replay(&mut world);
    restored_solid_is_visible(&mut world);
    let directory = tempfile::tempdir().unwrap();
    save_and_reopen(&mut world, &directory.path().join("free-rectangle.opd"));
    world.pump(Vec::new());
    restored_solid_is_visible(&mut world);
    world.close();
}

fn save_and_reopen(world: &mut ApplicationWorld, path: &std::path::Path) {
    use oblikovati_api::{
        client::Client,
        wire::{CloseDocumentArgs, OpenDocumentArgs, SaveDocumentAsArgs},
    };
    let document = world.active_id();
    let path = path.to_str().unwrap().to_owned();
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.clone(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .unwrap();
}

pub(super) fn open_extrusion_editor(world: &mut ApplicationWorld) {
    let position = super::super::diagnostics::pointer_pixels(world.run("Extrusion1").center());
    world.right_click(position);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text("Show Dimensions");
    let edit = edit_menu_position(world);
    world.click(edit);
    world.pump(Vec::new());
    world.expect_text("Distance A");
}

pub(super) fn edit_menu_position(world: &ApplicationWorld) -> [i32; 2] {
    let anchor = world.run("Show Dimensions").center();
    let center = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text
                .chars()
                .filter(|c| !c.is_whitespace())
                .eq("EditFeature".chars())
        })
        .min_by(|a, b| {
            (a.center()[1] - anchor[1])
                .abs()
                .total_cmp(&(b.center()[1] - anchor[1]).abs())
        })
        .expect("expected Edit Feature in the context menu")
        .center();
    super::super::diagnostics::pointer_pixels(center)
}
pub(super) fn feature_distance(world: &mut ApplicationWorld) -> f64 {
    use oblikovati_api::{client::Client, wire::FeatureRefArgs};
    Client::new(world.app().owner.document_host())
        .features()
        .get(&FeatureRefArgs { id: 1 })
        .unwrap()
        .feature
        .scalars
        .unwrap()[0]
        .value
}
#[test]
fn browser_distance_edit_previews_cancels_commits_and_reopens() {
    let mut world = ApplicationWorld::new();
    super::selected_rectangle_preview(&mut world);
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(&mut world), 1);
    let original_keys = named_solid_edges(&mut world);
    open_extrusion_editor(&mut world);
    super::replace_extrusion_distance(&mut world, "20");
    world.pump(Vec::new());
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_distance(&mut world), 10.);
    if std::env::var_os("OBK_EDIT_RECEIPT").is_some() {
        world.save_capture("extrusion-distance-edit-preview");
    }
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert_eq!(feature_distance(&mut world), 10.);
    open_extrusion_editor(&mut world);
    super::replace_extrusion_distance(&mut world, "20");
    world.pump(Vec::new());
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_distance(&mut world), 20.);
    assert_eq!(named_solid_edges(&mut world), original_keys);
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_eq!(feature_distance(&mut world), 10.);
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert_eq!(feature_distance(&mut world), 20.);
    assert_eq!(named_solid_edges(&mut world), original_keys);
    let directory = tempfile::tempdir().unwrap();
    save_and_reopen(&mut world, &directory.path().join("edited-extrusion.opd"));
    world.pump(Vec::new());
    assert_eq!(feature_distance(&mut world), 20.);
    assert_eq!(named_solid_edges(&mut world), original_keys);
    world.close();
}
