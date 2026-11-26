// SPDX-License-Identifier: GPL-2.0-only
//! Verify the Ceramic creation, finish, and relief controls and saved recipe.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::*;
use oblikovati_api::contract::Appearance;
use oblikovati_model::assets::{CeramicAppearance, CeramicFinish, CeramicKind};
use oblikovati_persistence::package::DocumentPackage;

const CERAMIC_ID: &str = "default-ceramic";
const CERAMIC_NAME: &str = "Default Ceramic";

#[when("I choose Ceramic from the create-appearance menu")]
#[given("a newly created ceramic appearance is open in the Appearance Editor")]
#[given("a ceramic appearance is open in the Appearance Editor")]
fn create_ceramic(world: &mut ApplicationWorld) {
    click_in(world, BROWSER, "Createappearance");
    select_menu_item(world, "Ceramic");
    separate_window(world, CERAMIC_NAME.into());
}

#[then(expr = "a selected document appearance named {string} is created")]
fn selected_ceramic(world: &mut ApplicationWorld, name: String) {
    let asset = ceramic_asset(world);
    assert!(asset.document);
    assert_eq!(asset.name, name);
    assert_eq!(asset.swatch, Some([255; 3]));
    assert_eq!(
        asset.appearance_metadata.unwrap().ceramic,
        Some(CeramicAppearance::default())
    );
    let text = world.window_text(BROWSER);
    let name = super::super::world::squeezed(&name);
    assert!(
        text.matches(&name).count() >= 2,
        "expected selected browser appearance: {text}"
    );
}

#[then("the Appearance Editor opens with a preview and an editable name")]
fn editor_preview(world: &mut ApplicationWorld) {
    world.run_in_window(EDITOR, "Name");
    world.run_in_window(EDITOR, "DefaultCeramic");
    let capture = capture_framebuffer(&mut world.app().window.native);
    let tones: std::collections::BTreeSet<_> = preview_pixels(&capture, EDITOR)
        .map(|pixel| pixel[0])
        .collect();
    assert!(tones.len() > 10, "expected a shaded Ceramic preview");
    clean_dialog(world);
}

#[then("the Ceramic section shows Type, Color and Finish")]
fn ceramic_fields(world: &mut ApplicationWorld) {
    let text = world.window_text(EDITOR);
    for field in [
        "Ceramic",
        "Type",
        "Porcelain",
        "Color255,255,255",
        "Finish",
        "HighGloss/Glazed",
    ] {
        assert!(text.contains(field), "missing {field} in {text:?}");
    }
}

#[then("Type offers Ceramic and Porcelain")]
fn types(world: &mut ApplicationWorld) {
    check_choices(world, "Porcelain", &["Ceramic", "Porcelain"]);
}

#[then("Finish offers High Gloss / Glazed, Satin and Matte")]
fn finishes(world: &mut ApplicationWorld) {
    check_choices(
        world,
        "HighGloss/Glazed",
        &["HighGloss/Glazed", "Satin", "Matte"],
    );
}

#[then("Finish Bumps, Relief Pattern and Tint are separate optional sections")]
fn effects(world: &mut ApplicationWorld) {
    for section in ["FinishBumps", "ReliefPattern", "Tint"] {
        assert!(!world.checked_run(super::effects::effect_checkbox(world, section)));
    }
}

#[when("I select Ceramic as its Type")]
fn choose_ceramic(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Porcelain");
    select_menu_item(world, "Ceramic");
}

#[when("I select Matte as its Finish")]
fn choose_matte(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    click_in(world, EDITOR, "HighGloss/Glazed");
    select_menu_item(world, "Matte");
}

#[then("the selected Type and Finish are shown in their controls")]
fn selected_fields(world: &mut ApplicationWorld) {
    let text = world.window_text(EDITOR);
    assert!(
        !text.contains("Porcelain") && text.matches("Ceramic").count() >= 3,
        "{text}"
    );
    world.run_in_window(EDITOR, "Matte");
    assert_eq!(
        ceramic_asset(world).appearance_metadata.unwrap().ceramic,
        Some(CeramicAppearance::default())
    );
}

#[then("the preview displays the changed finish")]
fn changed_finish(world: &mut ApplicationWorld) {
    preview_changed(world, 8, 100);
}

#[given("Relief Pattern is unchecked")]
fn relief_disabled(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(super::effects::effect_checkbox(world, "ReliefPattern")));
}

#[when("I expand Relief Pattern")]
fn expand_relief(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "ReliefPattern");
}

#[then("its Image slot and Amount control are disabled")]
fn disabled_relief_controls(world: &mut ApplicationWorld) {
    for label in ["Image", "Amount"] {
        assert!(world.run_in_window(EDITOR, label).alpha < u8::MAX);
    }
    clean_dialog(world);
}

#[then("the Image slot reports that no image is selected")]
fn relief_has_no_image(world: &mut ApplicationWorld) {
    world.run_in_window(EDITOR, "Noimageselected");
}

#[test]
fn ceramic_finish_and_wavy_bumps_are_saved_with_the_part() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_ceramic(&mut world);
    choose_ceramic(&mut world);
    choose_matte(&mut world);
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    let check = super::effects::effect_checkbox(&world, "FinishBumps");
    world.click(diagnostics::pointer_pixels(check.center()));
    world.pump(Vec::new());
    preview_changed(&mut world, 8, 100);
    apply(&mut world);
    clean_dialog(&mut world);
    let path = world.save_part_to_file(document, "ceramic-appearance.opd");
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let assets = saved.restore_assets().unwrap();
    let ceramic = assets
        .appearance_metadata(CERAMIC_ID)
        .unwrap()
        .ceramic
        .as_ref()
        .unwrap();
    assert_eq!(ceramic.kind, CeramicKind::Ceramic);
    assert_eq!(ceramic.finish, CeramicFinish::Matte);
    assert!(ceramic.finish_bumps.enabled);
    let specular = assets.appearances().get(CERAMIC_ID).unwrap().specular();
    let [roughness, highlights] = CeramicFinish::Matte
        .surface()
        .map(|value| f32::from(value) / 255.);
    assert_eq!(
        [specular.roughness, specular.weight],
        [roughness, highlights]
    );
}

pub(super) fn preview_changed(world: &mut ApplicationWorld, threshold: u8, minimum: usize) {
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    let changed = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(before, after)| before[0].abs_diff(after[0]) > threshold)
        .count();
    assert!(
        changed > minimum,
        "expected the ceramic preview to change, got {changed} changed pixels"
    );
}

fn ceramic_asset(world: &mut ApplicationWorld) -> oblikovati_gui::GuiAssetChoice {
    world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|asset| asset.id == CERAMIC_ID && asset.appearance_preview.unwrap().document_owned)
        .expect("expected a document-owned Ceramic appearance")
}

pub(super) fn check_choices(world: &mut ApplicationWorld, value: &str, choices: &[&str]) {
    click_in(world, EDITOR, value);
    for choice in choices {
        menu_item(world, choice);
    }
    world.escape();
    world.pump(Vec::new());
}

pub(super) fn select_menu_item(world: &mut ApplicationWorld, name: &str) {
    let at = diagnostics::pointer_pixels(menu_item(world, name).center());
    world.click(at);
    world.pump(Vec::new());
}

fn menu_item<'a>(world: &'a ApplicationWorld, name: &str) -> &'a diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == name)
        .unwrap_or_else(|| panic!("expected {name} in the appearance menu: {}", world.text()))
}

pub(super) mod height_maps;

mod duplicate;
