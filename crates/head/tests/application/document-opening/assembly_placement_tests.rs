// SPDX-License-Identifier: GPL-2.0-only
//! Native assembly opening and placement checks.
use super::*;

#[test]
fn assembly_ribbon_places_a_component_from_a_file() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let fixture = crate::scene_fixture::selected_prism_document();
    let part = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap()
    .with_parameters(&fixture.parameters, Default::default())
    .unwrap()
    .with_sketches(&fixture.sketches, 1, 104)
    .unwrap()
    .with_work_geometry(&fixture.work_geometry, 1)
    .unwrap();
    std::fs::write(folder.join("file-part.opd"), part.encode().unwrap()).unwrap();
    let assembly = saved_document(
        &mut world,
        GuiTemplateKind::Assembly,
        &folder,
        "file-root.oad",
    );
    let initial_count = Client::new(world.app().owner.document_host())
        .documents()
        .list()
        .unwrap()
        .documents
        .unwrap()
        .len();
    world.click_text("Place");
    browse_assembly_component(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "file-part.opd");
    choose_open(&mut world);
    settle(&mut world);
    world.save_capture("assembly-file-placement-preview");
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let orange = capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| {
            pixel[0] > pixel[1].saturating_add(15) && pixel[1] > pixel[2].saturating_add(30)
        })
        .count();
    assert!(
        orange > 100,
        "expected visible preview pixels, got {orange}"
    );
    let preview_count = Client::new(world.app().owner.document_host())
        .documents()
        .list()
        .unwrap()
        .documents
        .unwrap()
        .len();
    assert_eq!(preview_count, initial_count + 1);
    world.click_text("Cancel");
    settle(&mut world);
    let after_cancel = Client::new(world.app().owner.document_host())
        .documents()
        .list()
        .unwrap()
        .documents
        .unwrap()
        .len();
    assert_eq!(after_cancel, initial_count);
    world.click_text("Place");
    browse_assembly_component(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "file-part.opd");
    choose_open(&mut world);
    let button = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Place")
        .max_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .unwrap()
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(button));
    settle(&mut world);
    assert_eq!(world.active_id(), assembly);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 1);
    assert!(status.scene_errors.is_empty(), "{:?}", status.scene_errors);
    world.expect_text("file-part.opd:1");
    world.close();
}

#[test]
fn assembly_ribbon_places_an_open_component_and_replays_it() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let fixture = crate::scene_fixture::selected_prism_document();
    let part = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap()
    .with_parameters(&fixture.parameters, Default::default())
    .unwrap()
    .with_sketches(&fixture.sketches, 1, 104)
    .unwrap()
    .with_work_geometry(&fixture.work_geometry, 1)
    .unwrap();
    std::fs::write(folder.join("place-part.opd"), part.encode().unwrap()).unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "place-part.opd");
    choose_open(&mut world);
    let component = world.active_id();
    let name = world
        .state()
        .documents
        .iter()
        .find(|row| row.id == component)
        .unwrap()
        .name
        .clone();
    let assembly = saved_document(
        &mut world,
        GuiTemplateKind::Assembly,
        &folder,
        "place-root.oad",
    );
    world.click_text("Place");
    world.expect_text("Choose an open saved component");
    world.click_text("Choose an open saved component");
    world.click_text(&name);
    settle(&mut world);
    world.click_text("Cancel");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        0
    );
    world.click_text("Place");
    world.click_text("Choose an open saved component");
    world.click_text(&name);
    let button = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Place")
        .max_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .unwrap()
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(button));
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    assert_eq!(world.active_id(), assembly);
    let occurrences = Client::new(world.app().owner.document_host())
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap();
    assert!(occurrences[0].grounded);
    world.key(ViewportPhysicalKey::Z, true);
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        0
    );
    world.key(ViewportPhysicalKey::Y, true);
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    world.key(ViewportPhysicalKey::S, true);
    settle(&mut world);
    world.close_active_tab();
    settle(&mut world);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "place-root.oad");
    choose_open(&mut world);
    settle(&mut world);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 1);
    assert!(status.scene_errors.is_empty(), "{:?}", status.scene_errors);
    world.close();
}
