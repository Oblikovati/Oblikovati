// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer coverage for configured Part marking-menu commands (#4049).
use super::{ApplicationWorld, diagnostics, x11_input};

fn start_extrude_from_part_menu(world: &mut ApplicationWorld) {
    x11_input::focus_window();
    x11_input::move_pointer([900, 450]);
    world.pump(Vec::new());
    x11_input::right_click([900, 450]);
    world.pump(Vec::new());
    let button = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Extrude" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("Part menu: expected Extrude; rendered {:?}", world.text()));
    let point = diagnostics::pointer_pixels(button.center());
    x11_input::click(point);
    world.pump(Vec::new());
}

fn click_native_label(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    x11_input::click(point);
    world.pump(Vec::new());
}

#[test]
fn native_part_menu_extrude_starts_sketch_when_part_has_none() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    start_extrude_from_part_menu(&mut world);
    assert!(
        diagnostics::window_geometry("##plane-selection").is_some(),
        "expected plane-selection overlay"
    );
    let plane = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.replace(' ', "") == "XYPlane" && run.center()[0] > 310.)
        .unwrap_or_else(|| panic!("expected XY plane label in viewport: {:?}", world.text()))
        .center();
    x11_input::click(diagnostics::pointer_pixels(plane));
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    click_native_label(&mut world, "Rectangle");
    for point in super::documents::RECTANGLE {
        x11_input::click(point);
        world.pump(Vec::new());
    }
    click_native_label(&mut world, "Finish Sketch");
    world.pump(Vec::new());
    world.expect_text("Distance A");
    assert!(world.state().extrude_preview_active);
    click_native_label(&mut world, "Cancel");
    assert!(!world.state().extrude_preview_active);
    world.close();
}

#[test]
fn native_part_menu_extrude_opens_dialog_for_finished_sketch() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    start_extrude_from_part_menu(&mut world);
    world.expect_text("Distance A");
    world.expect_text("Profiles");
    click_native_label(&mut world, "Cancel");
    world.expect_no_text("Distance A");
    world.close();
}

#[test]
fn native_part_menu_starts_sketch_on_selected_face_and_reopens_it() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let (document, point, selected) = select_front_face(&mut world);
    x11_input::right_click(point);
    world.pump(Vec::new());
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("selected-face-new-sketch-menu");
    }
    click_native_label(&mut world, "New Sketch");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    let sketch = world
        .state()
        .active_sketch
        .expect("expected the new sketch to be active");
    let face = sketch_face_key(&mut world, document, sketch);
    assert_eq!(
        face.iter()
            .map(|byte| format!("{byte:02x}"))
            .collect::<String>(),
        selected
    );
    click_native_label(&mut world, "Finish Sketch");
    let reopened = save_and_reopen_face_sketch(&mut world, document);
    assert_eq!(sketch_face_key(&mut world, reopened, sketch), face);
    world.close();
}

fn select_front_face(world: &mut ApplicationWorld) -> (u64, [i32; 2], String) {
    let document = world.active_id();
    let (low, high) = super::viewport_orientation::solid_bounds(world);
    let point = [
        ((low[0] + high[0]) / 2) as i32,
        ((low[1] + high[1]) / 2) as i32,
    ];
    x11_input::focus_window();
    x11_input::move_pointer(point);
    world.pump(Vec::new());
    x11_input::click(point);
    world.pump(Vec::new());
    let state = world.state();
    assert!(
        state.model_face_chosen,
        "expected the clicked face to be selected"
    );
    let [(key, _)] = state.selected_face_appearances.as_slice() else {
        panic!("expected exactly one named face")
    };
    (document, point, key.clone())
}

fn sketch_face_key(
    world: &mut ApplicationWorld,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
) -> Vec<u8> {
    let placement = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .placement();
    let oblikovati_model::document_sketch::SketchPlacement::Face(face) = placement else {
        panic!("document {document}, sketch {sketch:?}: expected a face-based sketch");
    };
    face.key().to_vec()
}

fn save_and_reopen_face_sketch(world: &mut ApplicationWorld, document: u64) -> u64 {
    use oblikovati_api::{
        client::Client,
        wire::{CloseDocumentArgs, OpenDocumentArgs, SaveDocumentAsArgs},
    };
    let path = world
        .scratch_folder()
        .join("selected-face-sketch.opd")
        .to_string_lossy()
        .into_owned();
    Client::new(world.app().owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.clone(),
        })
        .unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .unwrap()
        .id
}

#[test]
fn native_part_menu_chamfer_opens_and_cancels_without_model_changes() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let before = world.state().browser.unwrap().features;
    x11_input::focus_window();
    x11_input::move_pointer([900, 450]);
    world.pump(Vec::new());
    x11_input::right_click([900, 450]);
    world.pump(Vec::new());
    let button = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Chamfer" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("Part menu: expected Chamfer; rendered {:?}", world.text()));
    x11_input::click(diagnostics::pointer_pixels(button.center()));
    world.pump(Vec::new());
    world.expect_text("Select Edges");
    assert!(!world.state().extrude_preview_active);
    click_native_label(&mut world, "Cancel");
    assert_eq!(world.state().browser.unwrap().features, before);
    assert!(!world.state().extrude_preview_active);
    world.close();
}
