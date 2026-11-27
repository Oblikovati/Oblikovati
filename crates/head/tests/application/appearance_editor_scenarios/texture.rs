// SPDX-License-Identifier: GPL-2.0-only
//! Drive the Generic Image menu and Checker editor through native input.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::*;
use oblikovati_persistence::package::DocumentPackage;

const TEXTURE: &str = "###texture-editor";

#[given("a generic appearance has no image selected")]
fn no_image(world: &mut ApplicationWorld) {
    create_generic(world);
    world.run_in_window(EDITOR, "Noimageselected");
    assert!(owned_asset(world).appearance_metadata.is_none());
}

#[when("I open the Generic Image menu")]
fn open_image_menu(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Image");
}

#[then("it offers Image, Checker, Gradient, Marble, Noise, Speckle, Tiles, Waves and Wood")]
fn image_choices(world: &mut ApplicationWorld) {
    for name in [
        "Image", "Checker", "Gradient", "Marble", "Noise", "Speckle", "Tiles", "Waves", "Wood",
    ] {
        popup_item(world, name);
    }
}

#[then("Edit Image and Remove Image are disabled")]
fn unavailable_image_commands(world: &mut ApplicationWorld) {
    for name in ["EditImage", "RemoveImage"] {
        assert!(popup_item(world, name).alpha < u8::MAX);
    }
}

#[when("I choose Checker from the Generic Image menu")]
fn select_checker(world: &mut ApplicationWorld) {
    open_image_menu(world);
    let item = popup_item(world, "Checker");
    assert_eq!(item.alpha, u8::MAX);
    world.click(diagnostics::pointer_pixels(item.center()));
    world.pump(Vec::new());
}

#[then("a separate Texture Editor opens with a checker preview")]
fn checker_preview(world: &mut ApplicationWorld) {
    world.run_in_window(TEXTURE, "TextureEditor—Checker");
    let (position, _) = diagnostics::window_geometry(TEXTURE).unwrap();
    let [x, y] = diagnostics::pointer_pixels(position).map(|value| value as usize);
    let capture = capture_framebuffer(&mut world.app().window.native);
    let samples: Vec<_> = pixels_in(&capture, [x + 35, y + 53], [x + 155, y + 170]).collect();
    assert!(samples.iter().filter(|pixel| pixel[0] < 35).count() > 1200);
    assert!(samples.iter().filter(|pixel| pixel[0] > 225).count() > 1200);
}

#[then("the appearance shows an image in its Generic Image slot")]
fn selected_image(world: &mut ApplicationWorld) {
    world.run_in_window(EDITOR, "Checkertexture");
    can_apply(world);
    assert!(owned_asset(world).appearance_metadata.is_none());
}

#[then("the Texture Editor shows Color 1, Color 2 and Soften")]
fn color_and_soften(world: &mut ApplicationWorld) {
    for label in ["Color1", "Color2", "Soften"] {
        world.run_in_window(TEXTURE, label);
    }
}

#[then("its transforms include position, scale and horizontal and vertical repeat controls")]
fn texture_transforms(world: &mut ApplicationWorld) {
    let text = world.window_text(TEXTURE);
    for label in [
        "Transforms",
        "Position",
        "Xoffset(mm)",
        "Yoffset(mm)",
        "Rotation(°)",
        "Scale",
        "Samplewidth(mm)",
        "Sampleheight(mm)",
        "Horizontal",
        "Vertical",
    ] {
        assert!(text.contains(label), "expected {label} in {text}");
    }
}

#[then("the preview shows the sample dimensions")]
fn sample_dimensions(world: &mut ApplicationWorld) {
    world.run_in_window(TEXTURE, "Sample25×25mm");
}

#[test]
fn checker_transforms_persist_only_after_appearance_apply() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    no_image(&mut world);
    let before = capture_framebuffer(&mut world.app().window.native);
    select_checker(&mut world);
    let after = capture_framebuffer(&mut world.app().window.native);
    let changed = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(before, after)| before[0].abs_diff(after[0]) > 22)
        .count();
    assert!(
        changed > 900,
        "expected Checker to change the appearance preview"
    );
    let (position, _) = diagnostics::window_geometry(TEXTURE).unwrap();
    let width = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.ends_with("Samplewidth(mm)") && run.minimum[0] >= position[0])
        .expect("expected the Checker sample width field");
    world.click(diagnostics::pointer_pixels([
        position[0] + 50.,
        width.center()[1],
    ]));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("36");
    world.pump(Vec::new());
    world.run_in_window(TEXTURE, "Sample36×36mm");
    selected_image(&mut world);
    apply(&mut world);
    clean_dialog(&mut world);
    let path = world.save_part_to_file(document, "checker-texture.opd");
    let encoded = std::fs::read(path).unwrap();
    let assets = DocumentPackage::decode(&encoded)
        .unwrap()
        .restore_assets()
        .unwrap();
    let checker = assets.appearance_metadata(ID).unwrap().checker.unwrap();
    assert_eq!(checker.size_mm, [36, 36]);
}

fn popup_item<'a>(world: &'a ApplicationWorld, name: &str) -> &'a diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == name && world.frame.is_popup_row(run))
        .unwrap_or_else(|| {
            panic!(
                "expected {name} in the Generic Image menu: {}",
                world.text()
            )
        })
}
