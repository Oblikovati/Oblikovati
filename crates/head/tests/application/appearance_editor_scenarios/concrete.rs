// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Concrete property page and verify its applied surface in a saved Part.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::{
    ceramic::{
        check_choices,
        height_maps::{choose_file, write_map},
        preview_changed, select_menu_item,
    },
    *,
};
use oblikovati_api::contract::Appearance;
use oblikovati_model::assets::{ConcreteAppearance, ConcreteFinish, ConcreteSealant};
use oblikovati_persistence::package::DocumentPackage;

const CONCRETE_ID: &str = "default-concrete";
const CONCRETE_NAME: &str = "Default Concrete";

#[when("I choose Concrete from the create-appearance menu")]
#[given("a new concrete appearance is open in the Appearance Editor")]
fn create_concrete(world: &mut ApplicationWorld) {
    click_in(world, BROWSER, "Createappearance");
    select_menu_item(world, "Concrete");
    concrete_open(world, CONCRETE_NAME.into());
}

#[then(expr = "a document appearance named {string} opens in the Appearance Editor")]
fn concrete_open(world: &mut ApplicationWorld, name: String) {
    separate_window(world, name.clone());
    let asset = owned_concrete(world);
    assert!(asset.document);
    assert_eq!(asset.name, name);
    assert_eq!(asset.swatch, Some([120; 3]));
    assert_eq!(
        asset.appearance_metadata.unwrap().concrete,
        Some(ConcreteAppearance::default())
    );
}

#[then("its Concrete section shows Color and Sealant")]
fn fields(world: &mut ApplicationWorld) {
    let text = world.window_text(EDITOR);
    for field in ["Concrete", "Color120,120,120", "Sealant", "None"] {
        assert!(text.contains(field), "{field}: {text}");
    }
}

#[then("Sealant offers None, Epoxy and Acrylic")]
fn sealant_choices(world: &mut ApplicationWorld) {
    check_choices(world, "None", &["None", "Epoxy", "Acrylic"]);
}

#[when("I select Epoxy as Sealant")]
fn choose_epoxy(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    click_in(world, EDITOR, "None");
    select_menu_item(world, "Epoxy");
    assert_eq!(
        owned_concrete(world)
            .appearance_metadata
            .unwrap()
            .concrete
            .unwrap()
            .sealant,
        ConcreteSealant::None
    );
}

#[then("the preview changes")]
fn changed_preview(world: &mut ApplicationWorld) {
    preview_changed(world, 6, 400);
}

#[then("the editor stays open and shows Epoxy")]
fn applied_epoxy(world: &mut ApplicationWorld) {
    separate_window(world, CONCRETE_NAME.into());
    world.run_in_window(EDITOR, "Epoxy");
    assert_eq!(
        owned_concrete(world)
            .appearance_metadata
            .unwrap()
            .concrete
            .unwrap()
            .sealant,
        ConcreteSealant::Epoxy
    );
}

#[when("I expand Finish Bumps and open its Type list")]
fn open_finish_bumps(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "FinishBumps");
    click_in(world, EDITOR, "BroomStraight");
}

#[then("it offers Broom Straight, Broom Curved, Smooth, Polished and Stamped/Custom")]
fn finish_choices(world: &mut ApplicationWorld) {
    expect_popup_choices(
        world,
        &[
            "BroomStraight",
            "BroomCurved",
            "Smooth",
            "Polished",
            "Stamped/Custom",
        ],
    );
}

#[then("Finish Bumps has no separate enable checkbox")]
fn no_finish_toggle(world: &mut ApplicationWorld) {
    close_dropdown(world);
    let header = world.run_in_window(EDITOR, "FinishBumps");
    assert!(!world.frame.runs.iter().any(|run| {
        run.list == header.list
            && run.text == "Enabled"
            && (run.center()[1] - header.center()[1]).abs() < 6.
    }));
}

#[when("I expand Weathering and open its Type list")]
fn open_weathering(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Weathering");
    click_in(world, EDITOR, "Automatic");
}

#[then("it offers Automatic and Custom Image")]
fn weathering_choices(world: &mut ApplicationWorld) {
    expect_popup_choices(world, &["Automatic", "CustomImage"]);
}

#[then("Weathering has a separate enable checkbox")]
fn weathering_enabled(world: &mut ApplicationWorld) {
    close_dropdown(world);
    assert!(world.checked_run(effects::effect_checkbox(world, "Weathering")));
}

#[then("Tint is a separate optional section")]
fn tint_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(effects::effect_checkbox(world, "Tint")));
    clean_dialog(world);
}

#[test]
fn concrete_created_from_the_editor_retains_its_applied_settings() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    click_in(&mut world, EDITOR, "Createappearance");
    select_menu_item(&mut world, "Concrete");
    concrete_open(&mut world, CONCRETE_NAME.into());
    choose_epoxy(&mut world);
    changed_preview(&mut world);
    apply(&mut world);
    applied_epoxy(&mut world);
    open_finish_bumps(&mut world);
    select_menu_item(&mut world, "Polished");
    open_weathering(&mut world);
    select_menu_item(&mut world, "CustomImage");
    let weathered = write_map(&mut world, "concrete-weathering.png", true);
    choose_file(&mut world, &weathered);
    std::fs::remove_file(weathered).unwrap();
    apply(&mut world);
    clean_dialog(&mut world);
    let source = capture_framebuffer(&mut world.app().window.native);
    click_in(&mut world, EDITOR, "Createappearance");
    select_menu_item(&mut world, "DuplicateasGeneric");
    separate_window(&mut world, "Default Concrete(1)".into());
    generic_fields(&mut world);
    let converted = capture_framebuffer(&mut world.app().window.native);
    assert!(preview_pixels(&source, EDITOR).eq(preview_pixels(&converted, EDITOR)));
    let path = world.save_part_to_file(document, "concrete-epoxy.opd");
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let assets = saved.restore_assets().unwrap();
    let concrete = assets
        .appearance_metadata(CONCRETE_ID)
        .unwrap()
        .concrete
        .as_ref()
        .unwrap();
    assert_eq!(concrete.sealant, ConcreteSealant::Epoxy);
    assert_eq!(concrete.finish, ConcreteFinish::Polished);
    assert_eq!(
        concrete.weathering_image.as_ref().unwrap().name(),
        "concrete-weathering.png"
    );
    let converted = assets
        .appearance_metadata("default-concrete-copy-1")
        .unwrap();
    assert!(converted.concrete.is_none());
    assert_eq!(
        converted
            .generic_effects
            .as_ref()
            .unwrap()
            .concrete_surface
            .as_deref(),
        Some(concrete)
    );
    let [roughness, highlights] = concrete.surface().map(|value| f32::from(value) / 255.);
    let specular = assets.appearances().get(CONCRETE_ID).unwrap().specular();
    assert_eq!(
        [specular.roughness, specular.weight],
        [roughness, highlights]
    );
    assert!(
        assets.appearances().get(ID).is_some(),
        "source Generic appearance was lost"
    );
}

fn owned_concrete(world: &mut ApplicationWorld) -> oblikovati_gui::GuiAssetChoice {
    world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|asset| asset.id == CONCRETE_ID && asset.document)
        .expect("expected a document-owned Concrete appearance")
}

pub(super) fn expect_popup_choices(world: &ApplicationWorld, choices: &[&str]) {
    for label in choices {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.text == *label && world.frame.is_popup_row(run)),
            "expected {label} in {}",
            world.text()
        );
    }
}

pub(super) fn close_dropdown(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
