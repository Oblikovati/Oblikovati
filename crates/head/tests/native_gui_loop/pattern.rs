// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement,
    history::DocumentSketchCreation,
    sketch::{SketchPlacementRecipe, SketchPlane},
};

#[test]
fn native_rectangular_pattern_dialog_previews_cancels_and_creates() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let sketch = owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "SeedSketch".into(),
                placement: SketchPlacement::Fixed(SketchPlane::xy()),
                orientation: None,
                project_origin: false,
            },
        )
        .unwrap();
    let recipe = SketchPlacementRecipe::line_chain(&[[0., 2.], [5., 2.]], false).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &recipe, &[])
        .unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let pixel = window
        .native
        .view
        .sketch_pixel(SketchPlane::xy(), [2.5, 2.])
        .unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .len(),
        1
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Sketch"),
    );
    open_pattern_group(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Rectangular"),
    );
    assert!(application_bdd::window_geometry("Rectangular Pattern").is_some());
    assert!(owner.sketch_preview().is_some());
    if let Ok(path) = std::env::var("OBLIKOVATI_PATTERN_TEST_CAPTURE") {
        use crate::WindowCaptureFiles;
        let capture = capture_framebuffer(&mut window.native);
        crate::NativeWindowCaptureFiles
            .write_capture(std::path::Path::new(&path), &capture)
            .unwrap();
    }
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .rectangular_patterns()
            .is_empty()
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Cancel"),
    );
    assert!(owner.sketch_preview().is_none());
    open_pattern_group(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Rectangular"),
    );
    assert!(owner.sketch_preview().is_some());
    let (position, size) = application_bdd::window_geometry("Rectangular Pattern").unwrap();
    let close = application_bdd::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    click_shell_position(&mut window, &mut owner, close);
    assert!(
        owner.sketch_preview().is_none(),
        "closing must clear the pattern preview"
    );
    open_pattern_group(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Rectangular"),
    );
    assert!(owner.sketch_preview().is_some());
    set_first_count(&mut window, &mut owner, "0");
    assert!(owner.sketch_preview().is_none());
    set_first_count(&mut window, &mut owner, "3");
    assert!(owner.sketch_preview().is_some());
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Reverse Direction 1"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("OK"),
    );
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry();
    assert_eq!(geometry.rectangular_patterns().len(), 1);
    assert_eq!(geometry.rectangular_patterns()[0].definition.counts, [3, 1]);
    assert_eq!(
        geometry.rectangular_patterns()[0].definition.directions[0],
        [-1., 0.]
    );
    let direction_bits =
        geometry.rectangular_patterns()[0].definition.directions[0].map(f64::to_bits);
    assert_eq!(geometry.lines().len(), 3);
    assert!(owner.sketch_preview().is_none());
    open_saved_pattern_menu(&mut window, &mut owner);
    set_first_count(&mut window, &mut owner, "4");
    assert!(owner.sketch_preview().is_some());
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Cancel"),
    );
    assert_eq!(pattern_counts(&owner, document, sketch), [3, 1]);
    open_saved_pattern(&mut window, &mut owner);
    set_first_count(&mut window, &mut owner, "4");
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("OK"),
    );
    assert_eq!(pattern_counts(&owner, document, sketch), [4, 1]);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .rectangular_patterns()[0]
            .definition
            .directions[0],
        [-1., 0.]
    );
    let edited_direction = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .rectangular_patterns()[0]
        .definition
        .directions[0];
    assert_eq!(edited_direction.map(f64::to_bits), direction_bits);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Undo"),
    );
    assert_eq!(pattern_counts(&owner, document, sketch), [3, 1]);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Redo"),
    );
    assert_eq!(pattern_counts(&owner, document, sketch), [4, 1]);
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn open_saved_pattern(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let position = application_bdd::rendered_label_center("Rectangular Pattern1");
    drive_pointer_clicks(window, owner, position, 2);
    assert!(application_bdd::window_geometry("Rectangular Pattern").is_some());
}
fn open_saved_pattern_menu(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let position = application_bdd::rendered_label_center("Rectangular Pattern1");
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Right,
                pressed,
                position,
                modifiers: Default::default(),
            })
        })
        .into();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    click_shell_position(
        window,
        owner,
        application_bdd::rendered_label_center("Edit Pattern"),
    );
    assert!(application_bdd::window_geometry("Rectangular Pattern").is_some());
}
fn set_first_count(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner, value: &str) {
    let label = application_bdd::rendered_label_center("Direction 1 count");
    click_shell_position(window, owner, [label[0], label[1] + 22]);
    press_shell_key(
        window,
        owner,
        oblikovati_viewport::ViewportPhysicalKey::Backspace,
    );
    type_native_text(window, owner, value);
}
fn pattern_counts(
    owner: &NativeDesktopOwner,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
) -> [u32; 2] {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .rectangular_patterns()[0]
        .definition
        .counts
}

fn open_pattern_group(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let label = application_bdd::rendered_label_center("Pattern");
    click_shell_position(window, owner, [label[0], label[1] - 35]);
}
