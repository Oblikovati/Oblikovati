// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Finish scenarios (finish-01..05, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiFinishPreview;
use setup::{
    FINISH_WINDOW, click_cylindrical_body_face, click_finish_ribbon, click_finish_type,
    click_panel_control, click_top_planar_face, click_visible_block_face,
    double_click_finish1_in_browser, excluded_count, expand_finishes_browser_folder,
    has_face_color, included_count, select_process_choice,
};

#[given("Finish is open without included geometry")]
fn finish_open_without_geometry(world: &mut ApplicationWorld) {
    click_finish_ribbon(world);
    assert_eq!(included_count(world), 0);
}

#[then("the displayed area is zero and OK is disabled.")]
fn displayed_area_zero_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("0.000mm^2"),
        "expected 0.000 mm^2 area display, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("the user selects the 40 by 25 mm top planar face")]
fn user_selects_40_by_25_top_face(world: &mut ApplicationWorld) {
    click_top_planar_face(world);
}

#[then("Include shows 1 Entity, the area is 1000 mm^2, and OK is enabled.")]
fn include_one_entity_area_1000_ok_enabled(world: &mut ApplicationWorld) {
    assert_eq!(included_count(world), 1);
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish preview");
    let GuiFinishPreview::Ready { area_mm2 } = preview.result else {
        panic!("expected Ready finish preview, got {:?}", preview.result);
    };
    assert!(
        (area_mm2 - 1000.0).abs() < 1e-2,
        "expected 1000 mm² analytic area, got {area_mm2}"
    );
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("1Entity") && text.contains("1000mm^2"),
        "expected 1 Entity and 1000 mm^2, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given("the user opens Finish, selects the top planar face and activates the second type")]
fn open_finish_select_top_and_type_2(world: &mut ApplicationWorld) {
    click_finish_ribbon(world);
    click_top_planar_face(world);
    click_finish_type(world, 2);
}

#[when("the user chooses Zn12/Cn/T2 from Process")]
fn user_chooses_zn12_process(world: &mut ApplicationWorld) {
    select_process_choice(world, "Zn12/Cn/T2");
}

#[then("Appearance shows Turquoise and Thickness shows 12 micron.")]
fn appearance_turquoise_thickness_12_micron(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish preview");
    assert_eq!(preview.request.process, "Zn12/Cn/T2");
    assert_eq!(preview.request.appearance, "Turquoise");
    assert_eq!(preview.request.thickness, "12 micron");
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("Turquoise") && text.contains("12micron"),
        "expected Turquoise and 12 micron in panel, saw {text}"
    );
}

#[then("the face preview updates.")]
fn face_preview_updates(world: &mut ApplicationWorld) {
    assert!(
        has_face_color(world, "Turquoise"),
        "expected live Turquoise face preview color in viewport scene"
    );
}

#[given("the user opens Finish and selects the top planar face")]
fn open_finish_and_select_top_face(world: &mut ApplicationWorld) {
    click_finish_ribbon(world);
    click_top_planar_face(world);
}

#[when("the user chooses the third type")]
fn user_chooses_third_type(world: &mut ApplicationWorld) {
    click_finish_type(world, 3);
}

#[then("the properties include Hardness and Depth.")]
fn properties_include_hardness_and_depth(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("Hardness") && text.contains("Depth"),
        "expected Hardness and Depth for third type, saw {text}"
    );
}

#[when("the user chooses the fifth type")]
fn user_chooses_fifth_type(world: &mut ApplicationWorld) {
    click_finish_type(world, 5);
}

#[then("the defaults show Prime/Paint RAL 3018, Red, and 50 micron.")]
fn defaults_show_ral_3018_red_50_micron(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish preview");
    assert_eq!(preview.request.process, "Prime/Paint RAL 3018");
    assert_eq!(preview.request.appearance, "Red");
    assert_eq!(preview.request.thickness, "50 micron");
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("Prime/PaintRAL3018") && text.contains("Red") && text.contains("50micron"),
        "expected Prime/Paint RAL 3018, Red, and 50 micron, saw {text}"
    );
}

#[given("the fifth type is selected and Export all Finish Parameters is enabled")]
fn fifth_type_and_export_enabled(world: &mut ApplicationWorld) {
    click_finish_type(world, 5);
    let already_checked = world
        .state()
        .finish_preview
        .as_ref()
        .is_some_and(|p| p.request.export_all_parameters);
    if !already_checked {
        click_panel_control(world, "Export all Finish Parameters");
    }
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish preview");
    assert!(preview.request.export_all_parameters);
}

#[then("the properties close and the face appears red.")]
fn properties_close_and_face_appears_red(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(FINISH_WINDOW).is_none(),
        "expected Finish panel to close after OK"
    );
    assert!(
        has_face_color(world, "Red"),
        "expected committed face to appear red in viewport scene"
    );
}

#[then("expanding Finishes in the browser shows Finish1.")]
fn expanding_finishes_shows_finish1(world: &mut ApplicationWorld) {
    expand_finishes_browser_folder(world);
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("Finishes") && model.contains("Finish1"),
        "expected Finishes > Finish1 in Model browser, saw {model}"
    );
}

#[given(
    "the user creates Finish1 on an exposed cylindrical-body face using Prime/Paint RAL 3018, Red, 50 micron, with Export all Finish Parameters enabled"
)]
fn create_finish1_on_cylindrical_face(world: &mut ApplicationWorld) {
    click_finish_ribbon(world);
    click_cylindrical_body_face(world);
    fifth_type_and_export_enabled(world);
    click_panel_control(world, "OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(FINISH_WINDOW).is_none(),
        "expected Finish1 to commit cleanly"
    );
}

#[given("the user suppresses the cylindrical body's extrusion while leaving the block visible")]
fn suppress_cylindrical_extrusion(world: &mut ApplicationWorld) {
    let doc = world.active_id();
    let cyl_feature_id = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        let snapshot = session.snapshot();
        // Find the extrusion that owns the cylindrical body (possibly upstream of Finish1).
        let finish_source = snapshot.work_geometry.features().iter().find_map(|f| {
            let oblikovati_model::feature::part::PartFeatureDefinition::Finish(def) =
                f.definition()
            else {
                return None;
            };
            Some(def.body_source())
        });
        finish_source.unwrap_or_else(|| {
            eval.ordered_body_outputs()
                .find_map(|(fid, body)| {
                    let faces = body.face_reference_keys(session.history(), guid).ok()?;
                    faces
                        .iter()
                        .any(|face| {
                            body.face_topology(session.history(), guid, face)
                                .is_ok_and(|t| {
                                    t.surface == oblikovati_martensite::PrismSurfaceKind::Cylinder
                                })
                        })
                        .then_some(fid)
                })
                .expect("expected cylindrical body feature")
        })
    };
    let name = world
        .state()
        .browser
        .expect("expected browser")
        .features
        .iter()
        .find(|f| f.id == cyl_feature_id)
        .expect("expected cylindrical extrusion feature")
        .name
        .clone();
    let _ = (doc, name);
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .features()
        .set_suppressed(&oblikovati_api::wire::SetFeatureSuppressedArgs {
            id: cyl_feature_id.0,
            suppressed: true,
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[given("Finish1 has a warning because its original included geometry is unavailable")]
fn finish1_has_warning(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1, "expected only block body visible");
    let finish_id = browser
        .features
        .iter()
        .find(|f| f.name == "Finish1")
        .expect("expected Finish1 feature")
        .id;
    let seed = browser
        .finish_edits
        .get(&finish_id)
        .expect("expected Finish1 edit seed");
    assert!(
        seed.has_warning,
        "expected Finish1 warning when upstream geometry is suppressed"
    );
    assert!(
        seed.included_faces.is_empty(),
        "expected Finish1 live included faces to be empty"
    );
    expand_finishes_browser_folder(world);
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("Finish1[!]"),
        "expected Finish1 warning badge in browser, saw {model}"
    );
}

#[when("I open Finish1 for editing")]
fn open_finish1_for_editing(world: &mut ApplicationWorld) {
    double_click_finish1_in_browser(world);
}

#[then(
    "the panel shows Prime/Paint RAL 3018, Red, 50 micron, and Export all Finish Parameters checked"
)]
fn panel_shows_saved_paint_settings(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish edit preview");
    assert_eq!(preview.request.process, "Prime/Paint RAL 3018");
    assert_eq!(preview.request.appearance, "Red");
    assert_eq!(preview.request.thickness, "50 micron");
    assert!(preview.request.export_all_parameters);
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("Prime/PaintRAL3018") && text.contains("Red") && text.contains("50micron"),
        "expected preserved Paint settings, saw {text}"
    );
}

#[then("Include is empty and OK is disabled in this later model state")]
fn include_empty_and_ok_disabled(world: &mut ApplicationWorld) {
    assert_eq!(included_count(world), 0);
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected Include showing Select Faces, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("I select a visible entity")]
fn select_visible_entity(world: &mut ApplicationWorld) {
    let _ = click_visible_block_face(world);
}

#[then("Include shows 1 Entity, an area is displayed, and OK is enabled")]
fn include_one_entity_area_and_ok_enabled(world: &mut ApplicationWorld) {
    assert_eq!(included_count(world), 1);
    let preview = world
        .state()
        .finish_preview
        .expect("expected finish preview");
    let GuiFinishPreview::Ready { area_mm2 } = preview.result else {
        panic!("expected Ready preview, got {:?}", preview.result);
    };
    assert!(area_mm2 > 0.0, "expected positive area, got {area_mm2}");
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("1Entity") && text.contains("mm^2"),
        "expected 1 Entity and displayed area, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("I activate Exclude and click the visible face")]
fn activate_exclude_and_click_visible_face(world: &mut ApplicationWorld) {
    click_panel_control(world, "Select Faces");
    let _ = click_visible_block_face(world);
}

#[then("Exclude remains at Select Faces")]
fn exclude_remains_at_select_faces(world: &mut ApplicationWorld) {
    assert_eq!(excluded_count(world), 0);
    let text = super::super::world::squeezed(&world.window_text(FINISH_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected Exclude to remain at Select Faces, saw {text}"
    );
}
