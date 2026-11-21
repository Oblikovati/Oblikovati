// SPDX-License-Identifier: GPL-2.0-only
//! Assign part assets through the Tools ribbon.
use super::*;
use oblikovati_api::{client::Client, wire::OpenDocumentArgs};

#[test]
fn part_material_and_appearance_choices_use_host_history() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    world.click_text("Tools");
    world.expect_text("Material and Appearance");
    let material = choose_first_material(&mut world);
    world.click_text("Undo");
    assert_eq!(world.state().browser.unwrap().part_material, None);
    world.click_text("Redo");
    assert_material(&mut world, &material);
    let appearance = choose_first_appearance(&mut world);
    world.click_text("Undo");
    assert_eq!(world.state().browser.unwrap().part_appearance, None);
    world.click_text("Redo");
    assert_appearance(&mut world, &appearance);
    world.save_capture("part-material-appearance");
    reset_ribbon(&mut world);
    assert_material(&mut world, &material);
    assert_appearance(&mut world, &appearance);
    let path = world.save_part_to_file(document, "part-assets.opd");
    reopen_part(&mut world, &path);
    assert_material(&mut world, &material);
    assert_appearance(&mut world, &appearance);
    world.close();
}
fn choose_first_material(world: &mut ApplicationWorld) -> String {
    let browser = world.state().browser.unwrap();
    let asset = browser
        .materials
        .first()
        .expect("expected a shipped material");
    let (id, name) = (asset.id.clone(), asset.name.clone());
    choose_asset(world, "Material", &name);
    assert_material(world, &id);
    id
}
fn choose_first_appearance(world: &mut ApplicationWorld) -> String {
    let browser = world.state().browser.unwrap();
    let asset = browser
        .appearances
        .iter()
        .find(|asset| asset.id != "default")
        .expect("expected a shipped appearance");
    let (id, name) = (asset.id.clone(), asset.name.clone());
    choose_asset(world, "Appearance", &name);
    assert_appearance(world, &id);
    id
}
fn choose_asset(world: &mut ApplicationWorld, label: &str, name: &str) {
    let field = world.run(label).clone();
    world.click([(field.minimum[0] - 40.) as i32, field.center()[1] as i32]);
    world.click_text(name);
}
fn assert_material(world: &mut ApplicationWorld, id: &str) {
    assert_eq!(
        world.state().browser.unwrap().part_material.as_deref(),
        Some(id)
    );
}
fn assert_appearance(world: &mut ApplicationWorld, id: &str) {
    assert_eq!(
        world.state().browser.unwrap().part_appearance.as_deref(),
        Some(id)
    );
}
fn reset_ribbon(world: &mut ApplicationWorld) {
    let position = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(position);
    world.click_text("Ribbon Appearance");
    world.click_text("Reset Ribbon");
    world.click_text("Yes");
}
fn reopen_part(world: &mut ApplicationWorld, path: &std::path::Path) {
    world.reopen_application();
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn assert_submitted_color(world: &mut ApplicationWorld, rgb: [f32; 3]) {
    let colors = &world.app().window.scene_colors;
    assert!(!colors.is_empty(), "expected a submitted solid");
    assert!(
        colors
            .iter()
            .all(|color| *color == [rgb[0], rgb[1], rgb[2], 1.]),
        "expected scene color {rgb:?}, got {colors:?}"
    );
}
fn default_catalog_color() -> [f32; 3] {
    use oblikovati_api::contract::Appearance;
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let color = catalog
        .appearances()
        .get(oblikovati_model::assets::DEFAULT_APPEARANCE_ID)
        .unwrap()
        .base()
        .color;
    [color.r, color.g, color.b]
}

#[test]
fn assigning_material_updates_the_visible_solid_to_its_catalog_appearance() {
    use oblikovati_api::contract::{Appearance, Material};
    let mut world = ApplicationWorld::new();
    super::super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point =
        super::super::super::solid_visibility::visible_fixture_point(&capture, &windows).unwrap();
    let original = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert_submitted_color(&mut world, default_catalog_color());
    world.click_text("Tools");
    let material = choose_first_material(&mut world);
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let asset = catalog.materials().get(&material).unwrap();
    let color = catalog
        .appearances()
        .get(&asset.appearance_id())
        .unwrap()
        .base()
        .color;
    world.pump(Vec::new());
    assert_submitted_color(&mut world, [color.r, color.g, color.b]);
    let actual = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert_ne!(
        actual[..3],
        original[..3],
        "material {material:?}: expected a visible repaint"
    );
    world.save_capture("material-appearance-solid");
    world.click_text("Undo");
    assert_submitted_color(&mut world, default_catalog_color());
    let actual = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert!(
        (0..3).all(|i| actual[i].abs_diff(original[i]) <= 3),
        "expected {original:?} after undo, got {actual:?}"
    );
    world.close();
}

#[test]
fn assigning_part_appearance_refreshes_the_visible_solid() {
    let mut world = ApplicationWorld::new();
    super::super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point = super::super::super::solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected a visible solid before changing appearance");
    let before = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert_submitted_color(&mut world, default_catalog_color());
    world.click_text("Tools");
    let appearance = choose_first_appearance(&mut world);
    use oblikovati_api::contract::Appearance;
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let color = catalog.appearances().get(&appearance).unwrap().base().color;
    world.pump(Vec::new());
    assert_submitted_color(&mut world, [color.r, color.g, color.b]);
    let after = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert_ne!(
        before[..3],
        after[..3],
        "appearance {appearance:?}: expected the solid to refresh"
    );
    world.save_capture("part-appearance-solid");
    world.click_text("Undo");
    assert_submitted_color(&mut world, default_catalog_color());
    let restored = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert!(
        (0..3).all(|i| restored[i].abs_diff(before[i]) <= 3),
        "expected {before:?} after undo, got {restored:?}"
    );
    world.close();
}

#[test]
fn asset_assignment_refuses_stale_documents_and_unknown_assets() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    let active = world.add_part();
    let owner = &mut world.app().owner;
    let stale = owner.assign_part_asset_from_gui(
        first,
        oblikovati_gui::GuiPartAssetKind::Material,
        "beryllium",
    );
    assert!(stale.is_err(), "expected stale document refusal");
    let unknown = owner.assign_part_asset_from_gui(
        active,
        oblikovati_gui::GuiPartAssetKind::Appearance,
        "not-a-real-asset",
    );
    assert!(unknown.is_err(), "expected unknown asset refusal");
    assert_eq!(world.state().browser.unwrap().part_appearance, None);
    world.close();
}
