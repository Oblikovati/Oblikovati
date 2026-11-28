// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Derive scenarios (derive-01..derive-06).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiDerivePreview;
use oblikovati_model::feature::derived_part::{DeriveMirrorPlane, DeriveStyle};
use setup::{
    DERIVE_WINDOW, PICKER_WINDOW, choose_mirror_plane, click_derive_control, click_derive_ribbon,
    configure_separate_solid_yz_mirror, derived_browser_name, open_derive_dialog_for_source,
    open_disposable_destination, replace_scale_factor, scene_max_span,
    select_and_open_source_in_picker, select_derive_tab, stage_source_component,
};

#[given(expr = "a disposable copy of the native fixture {string} is open as the destination")]
fn disposable_destination_open(world: &mut ApplicationWorld, fixture: String) {
    open_disposable_destination(world, &fixture);
}

#[given(expr = "{string} is available as the source component")]
#[given(expr = "the unchanged native fixture {string} is available as the separate source")]
fn source_component_available(world: &mut ApplicationWorld, fixture: String) {
    stage_source_component(world, &fixture);
}

#[when("the user clicks Derive")]
fn user_clicks_derive(world: &mut ApplicationWorld) {
    click_derive_ribbon(world);
}

#[then("the MCAD Software opens a component file picker.")]
fn opens_component_file_picker(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(PICKER_WINDOW).is_some(),
        "expected component file picker {PICKER_WINDOW}, saw {}",
        world.text()
    );
}

#[when("the user selects and opens the native source component")]
fn user_selects_and_opens_source(world: &mut ApplicationWorld) {
    select_and_open_source_in_picker(world);
}

#[then("the Derived Part dialog opens with an object tree and preview.")]
fn derived_part_dialog_opens_with_tree_and_preview(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_some(),
        "expected {DERIVE_WINDOW} dialog, saw {}",
        world.text()
    );
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    assert!(
        panel.contains("SolidBodies"),
        "expected object tree in {DERIVE_WINDOW}, saw {panel}"
    );
    assert_eq!(
        world
            .state()
            .derive_preview
            .as_ref()
            .map(|preview| &preview.result),
        Some(&GuiDerivePreview::Ready)
    );
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected live preview geometry in viewport"
    );
}

#[given("the Derived Part dialog is open for the native source component")]
#[given("the native source component is open in Derived Part")]
fn derived_part_dialog_open(world: &mut ApplicationWorld) {
    open_derive_dialog_for_source(world);
}

#[when("the user opens Objects")]
fn user_opens_objects(world: &mut ApplicationWorld) {
    select_derive_tab(world, "Objects");
}

#[then("the object tree includes Solid Bodies, Sketches, Work Geometry and Parameters")]
fn object_tree_includes_groups(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    for label in ["SolidBodies", "Sketches", "WorkGeometry", "Parameters"] {
        assert!(
            panel.contains(label),
            "expected {label} in Derived Part Objects tab, saw {panel}"
        );
    }
}

#[then("derive-style choices are available")]
fn derive_style_choices_available(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    for style in [
        "Singlesolidbodymergingoutseams",
        "Solidbodykeepseams",
        "Maintaineachsolidasasolidbody",
        "Singlecompositefeature",
    ] {
        assert!(
            panel.contains(style),
            "expected derive style {style} in {panel}"
        );
    }
}

#[when("the user opens Options")]
fn user_opens_options(world: &mut ApplicationWorld) {
    select_derive_tab(world, "Options");
}

#[then("link controls, a scale factor and Mirror part are visible")]
fn options_controls_visible(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    for label in [
        "Linkfacecolorfromsourcecomponent",
        "ScaleFactor",
        "Mirrorpart",
    ] {
        assert!(
            panel.contains(label),
            "expected {label} in Derived Part Options tab, saw {panel}"
        );
    }
}

#[when("the user selects Representation")]
fn user_selects_representation(world: &mut ApplicationWorld) {
    select_derive_tab(world, "Representation");
}

#[then("Model State and Design View show [Primary].")]
fn representation_shows_primary(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    assert!(
        panel.contains("ModelState"),
        "expected Model State in {panel}"
    );
    assert!(
        panel.contains("DesignView"),
        "expected Design View in {panel}"
    );
    assert!(
        panel.matches("[Primary]").count() >= 2,
        "expected [Primary] for both Model State and Design View in {panel}"
    );
}

#[given("the native source component is open in Derived Part with the Options tab selected")]
fn derive_open_on_options(world: &mut ApplicationWorld) {
    open_derive_dialog_for_source(world);
    select_derive_tab(world, "Options");
}

#[when("the user enables Mirror part")]
fn user_enables_mirror_part(world: &mut ApplicationWorld) {
    click_derive_control(world, "Mirrorpart");
}

#[then("the mirror-plane selector becomes available with XY, YZ and XZ planes.")]
fn mirror_plane_selector_available(world: &mut ApplicationWorld) {
    let label = world.run_in_window(DERIVE_WINDOW, "Mirrorplane").clone();
    let (origin, size) = super::super::diagnostics::window_geometry(DERIVE_WINDOW).unwrap();
    let combo =
        super::super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.72, label.center()[1]]);
    world.click(combo);
    world.pump(Vec::new());
    for plane in ["XYPlane", "YZPlane", "XZPlane"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|row| world.frame.is_popup_row(row) && row.text == plane),
            "expected {plane} in Mirror plane dropdown, saw {}",
            world.text()
        );
    }
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
}

#[when("the user chooses YZ Plane")]
fn user_chooses_yz_plane(world: &mut ApplicationWorld) {
    choose_mirror_plane(world, "YZ Plane");
}

#[then("YZ Plane is selected.")]
fn yz_plane_is_selected(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .derive_preview
        .expect("expected active derive preview");
    assert!(preview.request.mirror);
    assert_eq!(preview.request.mirror_plane, DeriveMirrorPlane::YZ);
    let panel = super::super::world::squeezed(&world.window_text(DERIVE_WINDOW));
    assert!(
        panel.contains("YZPlane"),
        "expected YZ Plane selected in {panel}"
    );
}

#[given(
    "the user selects the source in Derive, chooses separate-solid output and enables mirroring across the YZ plane"
)]
fn select_source_separate_solid_yz_mirror(world: &mut ApplicationWorld) {
    configure_separate_solid_yz_mirror(world);
    let preview = world
        .state()
        .derive_preview
        .expect("expected derive preview");
    assert_eq!(preview.request.style, DeriveStyle::MaintainSolids);
    assert!(preview.request.mirror);
    assert_eq!(preview.request.mirror_plane, DeriveMirrorPlane::YZ);
    assert_eq!(preview.result, GuiDerivePreview::Ready);
}

#[then(
    "the dialog closes, the native source component appears in the browser, and the solid-body count is two."
)]
fn dialog_closes_browser_and_two_bodies(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_none(),
        "expected {DERIVE_WINDOW} dialog to close"
    );
    let source_name = derived_browser_name(world);
    world.expect_text(&source_name);
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == source_name && feature.kind == "Derived Part"),
        "expected derived entry {source_name:?} in browser features {:?}",
        browser.features
    );
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected two solid bodies after separate-solid derive, got {:?}",
        browser.bodies
    );
}

#[given(
    expr = "the user derives the unchanged native fixture {string} into it as a separate solid mirrored across the YZ plane"
)]
fn derive_fixture_as_separate_mirrored_solid(world: &mut ApplicationWorld, fixture: String) {
    stage_source_component(world, &fixture);
    configure_separate_solid_yz_mirror(world);
    click_derive_control(world, "OK");
    assert!(super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_none());
}

#[given("the destination contains the derived source component")]
fn destination_contains_derived_source(world: &mut ApplicationWorld) {
    let source_name = derived_browser_name(world);
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == source_name && feature.kind == "Derived Part"),
        "expected derived component {source_name:?} in destination browser"
    );
    assert_eq!(browser.bodies.len(), 2);
}

#[when("the user double-clicks its browser entry")]
fn user_double_clicks_derived_browser_entry(world: &mut ApplicationWorld) {
    let source_name = derived_browser_name(world);
    let row = world
        .run_in_window("Model", &source_name.replace(' ', ""))
        .center();
    world.double_click(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the MCAD Software opens the native source document.")]
fn opens_native_source_document(world: &mut ApplicationWorld) {
    let expected = world
        .saved_path
        .clone()
        .expect("expected staged source path");
    let active = world.active_id();
    let destination = world.other_document.expect("expected destination id");
    assert_ne!(
        active, destination,
        "expected active document to switch to native source document"
    );
    let active_path = world.state().document_storage[&active]
        .path
        .as_deref()
        .map(std::path::PathBuf::from)
        .expect("expected path on opened source document");
    assert_eq!(
        active_path.file_name(),
        expected.file_name(),
        "expected opened source document {:?}, got {:?}",
        expected,
        active_path
    );
}

#[when(
    "the user returns to the destination and chooses Edit Derived Part from the entry context menu"
)]
fn return_and_edit_derived_part(world: &mut ApplicationWorld) {
    let destination = world.other_document.expect("expected destination id");
    let dest_name = world.document_name(destination);
    let tab = world.expect_tab(&dest_name).center();
    world.click(super::super::diagnostics::pointer_pixels(tab));
    world.pump(Vec::new());
    assert_eq!(
        world.state().document,
        Some(destination),
        "expected destination document tab click to activate destination"
    );
    let source_name = derived_browser_name(world);
    let row = world
        .run_in_window("Model", &source_name.replace(' ', ""))
        .center();
    world.right_click(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.click_text("Edit Derived Part");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the Derived Part dialog opens.")]
fn derived_part_dialog_opens(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_some(),
        "expected {DERIVE_WINDOW} dialog to open"
    );
    world.observed_row_gap = Some(scene_max_span(world));
}

#[when("the user opens Options and changes the scale factor from 1 to 2")]
fn open_options_and_scale_1_to_2(world: &mut ApplicationWorld) {
    select_derive_tab(world, "Options");
    replace_scale_factor(world, "2");
    assert_eq!(
        world
            .state()
            .derive_preview
            .as_ref()
            .map(|preview| &preview.result),
        Some(&GuiDerivePreview::Ready)
    );
}

#[then("the preview becomes larger.")]
fn preview_becomes_larger(world: &mut ApplicationWorld) {
    let before = world
        .observed_row_gap
        .expect("expected baseline preview span before scaling");
    let after = scene_max_span(world);
    assert!(
        after > before * 1.4,
        "expected scale=2 preview span ({after}) to be substantially larger than scale=1 ({before})"
    );
}

#[then("the dialog closes.")]
fn derived_dialog_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_none(),
        "expected {DERIVE_WINDOW} dialog to close"
    );
    assert!(
        super::super::diagnostics::window_geometry(PICKER_WINDOW).is_none(),
        "expected {PICKER_WINDOW} picker to close"
    );
}
