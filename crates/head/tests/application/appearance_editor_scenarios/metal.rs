// SPDX-License-Identifier: GPL-2.0-only
//! Verify Metal creation, alloy selection, optional controls, and the saved surface.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::{
    ceramic::{
        height_maps::{choose_file, write_map},
        select_menu_item,
    },
    concrete::expect_popup_choices,
    *,
};
use oblikovati_api::contract::Appearance;
use oblikovati_model::assets::{MetalAppearance, MetalFinish, MetalKind};
use oblikovati_persistence::package::DocumentPackage;

const METAL_ID: &str = "default-metal";
const METAL_NAME: &str = "Default Metal";

#[given("an applied appearance is open in the Appearance Editor")]
fn applied_appearance(world: &mut ApplicationWorld) {
    create_generic(world);
    assert!(owned_asset(world).document);
    clean_dialog(world);
}

#[when("I choose Metal from the editor creation menu")]
fn create_metal(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Createappearance");
    select_menu_item(world, "Metal");
}

#[then(expr = "the editor switches to a new appearance named {string}")]
fn new_appearance(world: &mut ApplicationWorld, name: String) {
    separate_window(world, name.clone());
    assert!(
        world
            .state()
            .browser
            .unwrap()
            .appearances
            .iter()
            .any(|asset| {
                asset.document
                    && asset.name == name
                    && asset
                        .appearance_preview
                        .is_some_and(|finish| finish.document_owned)
            }),
        "expected a new owned appearance named {name:?}"
    );
}

#[then("the Metal section shows Type Aluminum and Finish Polished")]
fn metal_defaults(world: &mut ApplicationWorld) {
    for field in ["Metal", "Aluminum", "Type", "Polished", "Finish"] {
        world.run_in_window(EDITOR, field);
    }
    let asset = owned_metal(world);
    assert_eq!(asset.swatch, Some(MetalKind::Aluminum.color()));
    assert_eq!(asset.appearance_preview.unwrap().metalness, u8::MAX);
    assert_eq!(
        asset.appearance_metadata.unwrap().metal,
        Some(MetalAppearance::default())
    );
    clean_dialog(world);
}

#[then("Relief Pattern, Cutouts and Tint are separate unchecked sections")]
fn optional_sections(world: &mut ApplicationWorld) {
    for section in ["ReliefPattern", "Cutouts", "Tint"] {
        assert!(!world.checked_run(effects::effect_checkbox(world, section)));
    }
}

#[when("I open the Metal Type list")]
fn open_alloys(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Aluminum");
}

#[then(
    "it offers Aluminum, Anodized Aluminum, Chrome, Copper, Brass, Bronze, Stainless Steel and Zinc"
)]
fn alloy_choices(world: &mut ApplicationWorld) {
    expect_popup_choices(
        world,
        &[
            "Aluminum",
            "AnodizedAluminum",
            "Chrome",
            "Copper",
            "Brass",
            "Bronze",
            "StainlessSteel",
            "Zinc",
        ],
    );
}

#[when("I select Brass")]
fn choose_brass(world: &mut ApplicationWorld) {
    select_menu_item(world, "Brass");
    world.run_in_window(EDITOR, "Brass");
    assert_eq!(owned_metal(world).swatch, Some(MetalKind::Aluminum.color()));
}

#[then("the preview becomes gold colored")]
fn gold_preview(world: &mut ApplicationWorld) {
    let capture = capture_framebuffer(&mut world.app().window.native);
    let gold = preview_pixels(&capture, EDITOR)
        .filter(|[r, g, b, _]| r.saturating_sub(*g) > 18 && g.saturating_sub(*b) > 36)
        .count();
    assert!(gold > 1400, "expected brass tones, got {gold} gold pixels");
}

#[when("I open the Finish list")]
fn open_finishes(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Polished");
}

#[then("it offers Polished, Semi-polished, Satin and Brushed")]
fn finish_choices(world: &mut ApplicationWorld) {
    expect_popup_choices(world, &["Polished", "Semi-polished", "Satin", "Brushed"]);
}

#[test]
fn brushed_brass_and_cutout_map_survive_saving() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    applied_appearance(&mut world);
    create_metal(&mut world);
    new_appearance(&mut world, METAL_NAME.into());
    open_alloys(&mut world);
    choose_brass(&mut world);
    open_finishes(&mut world);
    select_menu_item(&mut world, "Brushed");
    apply(&mut world);
    let before = capture_framebuffer(&mut world.app().window.native);
    apply_cutout_image(&mut world);
    let after = capture_framebuffer(&mut world.app().window.native);
    let cutouts = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(before, after)| before[0].saturating_sub(after[0]) > 30)
        .count();
    assert!(
        cutouts > 1000,
        "expected the image to cut openings in the preview"
    );
    apply(&mut world);
    clean_dialog(&mut world);
    let path = world.save_part_to_file(document, "brushed-brass.opd");
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    check_saved_metal(&saved);
}

fn apply_cutout_image(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Cutouts");
    let checkbox = effects::effect_checkbox(world, "Cutouts");
    world.click(diagnostics::pointer_pixels(checkbox.center()));
    world.pump(Vec::new());
    click_in(world, EDITOR, "Image");
    let path = write_map(world, "metal-cutouts.png", false);
    choose_file(world, &path);
    std::fs::remove_file(path).unwrap();
}

fn check_saved_metal(saved: &DocumentPackage) {
    let assets = saved.restore_assets().unwrap();
    let metal = assets
        .appearance_metadata(METAL_ID)
        .unwrap()
        .metal
        .as_ref()
        .unwrap();
    assert_eq!(
        (metal.kind, metal.finish),
        (MetalKind::Brass, MetalFinish::Brushed)
    );
    assert!(metal.cutouts);
    assert_eq!(
        metal.cutout_image.as_ref().unwrap().name(),
        "metal-cutouts.png"
    );
    let appearance = assets.appearances().get(METAL_ID).unwrap();
    let base = appearance.base();
    assert_eq!(base.metalness, 1.);
    assert_eq!(
        [base.color.r, base.color.g, base.color.b],
        MetalKind::Brass
            .color()
            .map(|value| f32::from(value) / 255.)
    );
    let specular = appearance.specular();
    let [roughness, highlights] = MetalFinish::Brushed
        .surface()
        .map(|value| f32::from(value) / 255.);
    assert_eq!(
        [specular.roughness, specular.weight],
        [roughness, highlights]
    );
    assert_eq!(
        specular.roughness_anisotropy,
        f32::from(MetalFinish::Brushed.anisotropy()) / 255.
    );
    assert!(assets.appearances().get(ID).is_some());
}

fn owned_metal(world: &mut ApplicationWorld) -> oblikovati_gui::GuiAssetChoice {
    world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|asset| asset.id == METAL_ID && asset.document)
        .expect("expected a document-owned Metal appearance")
}
