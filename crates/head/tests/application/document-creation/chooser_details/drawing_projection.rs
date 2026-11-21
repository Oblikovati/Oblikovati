// SPDX-License-Identifier: GPL-2.0-only
//! Rendering the referenced part in a drawing sheet.
use super::*;

#[test]
fn drawing_sheet_displays_the_referenced_part_projection() {
    use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
    use oblikovati_model::drawing::DrawingDocument;
    use oblikovati_persistence::package::DocumentPackage;
    let mut world = ApplicationWorld::new();
    let folder = world.scratch_folder();
    let source = folder.join("model.opd");
    let snapshot = crate::scene_fixture::selected_prism_document();
    let part = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
        .encode()
        .unwrap();
    std::fs::write(&source, part).unwrap();
    let open = |world: &mut ApplicationWorld, path: &std::path::Path| {
        Client::new(world.app().owner.document_host())
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: path.to_string_lossy().into_owned(),
                ..Default::default()
            })
            .unwrap();
        world.pump(Vec::new());
    };
    open(&mut world, &source);
    let blank_path = folder.join("blank.odd");
    let blank = DrawingDocument::new();
    let bytes = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&blank))
        .unwrap()
        .encode()
        .unwrap();
    std::fs::write(&blank_path, bytes).unwrap();
    open(&mut world, &blank_path);
    let before = super::super::super::super::capture_framebuffer(&mut world.app().window.native);
    world.click_text("Base View");
    world.expect_text("Orientation");
    world.click_text("Preview");
    let preview_capture =
        super::super::super::super::capture_framebuffer(&mut world.app().window.native);
    let state = world.app().owner.gui_command_state().unwrap();
    let drawing = state.document.unwrap();
    assert!(
        state.drawing_projections[&drawing]
            .lines
            .iter()
            .any(|line| line.preview)
    );
    assert!(
        state.drawing_documents[&drawing].sheets()[0]
            .views
            .is_empty()
    );
    let width = preview_capture.extent[0] as usize;
    let height = preview_capture.extent[1] as usize;
    let mut blue = 0;
    for y in height * 35 / 100..height * 65 / 100 {
        for x in width * 40 / 100..width * 75 / 100 {
            let index = (y * width + x) * 4;
            let pixel = &preview_capture.rgba[index..index + 3];
            if before.rgba[index] > 200 && pixel[0] < 90 && pixel[1] < 150 && pixel[2] > 170 {
                blue += 1;
            }
        }
    }
    assert!(blue > 20, "expected blue preview pixels, found {blue}");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(!world.shows("Orientation"));
    let state = world.app().owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&drawing].lines.is_empty());
    world.click_text("Base View");
    world.expect_text("Orientation");
    world.click_text("Preview");
    let (position, size) =
        super::super::super::super::diagnostics::window_geometry("Base View").unwrap();
    let close = super::super::super::super::diagnostics::pointer_pixels([
        position[0] + size[0] - 13.,
        position[1] + 10.,
    ]);
    world.click(close);
    let state = world.app().owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&drawing].lines.is_empty());
    world.click_text("Base View");
    world.expect_text("Orientation");
    world.click_text("Apply");
    world.pump(Vec::new());
    let after = super::super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = after.extent[0] as usize;
    let height = after.extent[1] as usize;
    let mut darker = 0;
    for y in height * 35 / 100..height * 65 / 100 {
        for x in width * 40 / 100..width * 75 / 100 {
            let index = (y * width + x) * 4;
            if before.rgba[index] > 200 && after.rgba[index] < 100 {
                darker += 1;
            }
        }
    }
    assert!(
        darker > 20,
        "expected projected drawing lines, found {darker} darker pixels"
    );
    world.close();
}
