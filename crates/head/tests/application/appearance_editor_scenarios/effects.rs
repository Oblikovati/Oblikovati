// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for the Generic appearance effects.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::*;

#[given("Reflectivity is unchecked")]
#[then("Reflectivity remains unchecked")]
fn reflectivity_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(world.run_in_window(EDITOR, "Enabled")));
    clean_dialog(world);
}

#[when("I expand Reflectivity")]
fn expand_reflectivity(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Reflectivity");
}

#[then("Direct and Oblique controls are visible but disabled")]
fn reflectivity_controls_disabled(world: &mut ApplicationWorld) {
    for label in ["Direct", "Oblique"] {
        assert!(world.run_in_window(EDITOR, label).alpha < u8::MAX);
    }
}

#[when("I enable Reflectivity")]
fn enable_reflectivity(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    click_in(world, EDITOR, "Enabled");
}

#[then("Direct and Oblique controls become enabled")]
fn reflectivity_controls_enabled(world: &mut ApplicationWorld) {
    assert!(world.checked_run(world.run_in_window(EDITOR, "Enabled")));
    for label in ["Direct", "Oblique"] {
        assert_eq!(world.run_in_window(EDITOR, label).alpha, u8::MAX);
    }
}

#[then("the appearance preview changes to a reflective surface")]
fn reflectivity_changes_preview(world: &mut ApplicationWorld) {
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    let changed = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(previous, current)| previous[2].abs_diff(current[2]) > 10)
        .count();
    assert!(
        changed > 500,
        "expected the reflection to change the sphere, saw {changed} pixels"
    );
    assert!(owned_asset(world).appearance_metadata.is_none());
}

#[test]
fn appearance_reflectivity_controls_apply_and_restore_independent_values() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    expand_reflectivity(&mut world);
    enable_reflectivity(&mut world);
    for (label, offset) in [("Direct", 45.), ("Oblique", 190.)] {
        let caption = world.run_in_window(EDITOR, label);
        world.click(diagnostics::pointer_pixels([
            caption.minimum[0] - offset,
            caption.center()[1],
        ]));
        world.pump(Vec::new());
    }
    can_apply(&mut world);
    apply(&mut world);
    clean_dialog(&mut world);
    let effect = owned_asset(&mut world)
        .appearance_metadata
        .unwrap()
        .reflectivity
        .unwrap();
    assert!(effect.enabled);
    assert!(effect.direct > 60, "{effect:?}");
    assert!(effect.oblique < 40, "{effect:?}");

    click_in(&mut world, EDITOR, "Enabled");
    reflectivity_controls_disabled(&mut world);
    apply(&mut world);
    let disabled = owned_asset(&mut world)
        .appearance_metadata
        .unwrap()
        .reflectivity
        .unwrap();
    assert!(!disabled.enabled);
    assert_eq!(
        [disabled.direct, disabled.oblique],
        [effect.direct, effect.oblique]
    );
}

#[given("Transparency is unchecked")]
fn transparency_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(transparency_checkbox(world)));
    clean_dialog(world);
}

fn transparency_checkbox(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    let header = world.run_in_window(EDITOR, "Transparency");
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "Enabled"
                && run.list == header.list
                && run.minimum[0] > header.maximum[0]
                && (run.center()[1] - header.center()[1]).abs() < 8.
        })
        .expect("expected the Transparency checkbox beside its disclosure")
}

#[when("I expand and enable Transparency")]
fn enable_transparency(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    click_in(world, EDITOR, "Transparency");
    let point = diagnostics::pointer_pixels(transparency_checkbox(world).center());
    world.click(point);
    world.pump(Vec::new());
    assert!(world.checked_run(transparency_checkbox(world)));
    scroll_to_refraction(world);
}

fn scroll_to_refraction(world: &mut ApplicationWorld) {
    if world.window_text(EDITOR).contains("Indexofrefraction") {
        return;
    }
    let (position, size) = diagnostics::window_geometry(EDITOR).unwrap();
    let at = diagnostics::pointer_pixels([position[0] + 90., position[1] + size[1] - 58.]);
    #[cfg(target_os = "linux")]
    super::super::desktop_input::scroll(world, at, -4);
    #[cfg(not(target_os = "linux"))]
    {
        use oblikovati_viewport::{ViewportEvent, ViewportInput};
        world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
            position: at,
            notches: -4,
        })]);
    }
}

#[then("Amount, Image, Image Fade, Translucency and Refraction are shown")]
fn transparency_fields(world: &mut ApplicationWorld) {
    let header = world.run_in_window(EDITOR, "Transparency").center()[1];
    let shown = world.window_text(EDITOR);
    for label in ["Amount", "Image", "ImageFade", "Translucency", "Refraction"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.text == label && run.center()[1] > header),
            "{label}: {shown}"
        );
    }
}

#[then("the preview changes to show transparency")]
fn transparent_preview(world: &mut ApplicationWorld) {
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    let changed = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(old, new)| old[0].abs_diff(new[0]) > 12)
        .count();
    assert!(
        changed > 1500,
        "expected a visible transmission preview, saw {changed} changed pixels"
    );
    assert!(owned_asset(world).appearance_metadata.is_none());
}

#[when("I open the Refraction preset list")]
fn open_refraction(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Glass");
}

#[then("the list offers Air, Water, Alcohol, Quartz, Glass, Diamond and Custom")]
fn refraction_presets(world: &mut ApplicationWorld) {
    for preset in [
        "Air", "Water", "Alcohol", "Quartz", "Glass", "Diamond", "Custom",
    ] {
        let popup_item = world
            .frame
            .runs
            .iter()
            .find(|run| run.text == preset && world.frame.is_popup_row(run));
        assert!(
            popup_item.is_some(),
            "expected {preset} in {}",
            world.text()
        );
    }
}

#[test]
fn transparency_preserves_custom_refraction_through_apply() {
    use oblikovati_model::assets::RefractionPreset;

    let mut world = ApplicationWorld::new();
    world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    enable_transparency(&mut world);
    open_refraction(&mut world);
    refraction_presets(&mut world);
    world.click_text("Diamond");
    world.pump(Vec::new());
    assert!(world.window_text(EDITOR).contains("2.300"));
    click_in(&mut world, EDITOR, "Diamond");
    world.click_text("Custom");
    world.pump(Vec::new());
    let index = world.run_in_window(EDITOR, "Indexofrefraction");
    world.click(diagnostics::pointer_pixels([
        index.minimum[0] - 100.,
        index.center()[1],
    ]));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("1.7");
    world.pump(Vec::new());
    can_apply(&mut world);
    apply(&mut world);
    clean_dialog(&mut world);
    let applied = owned_asset(&mut world)
        .appearance_metadata
        .unwrap()
        .transparency
        .unwrap();
    assert!(applied.enabled);
    assert_eq!(applied.refraction, RefractionPreset::Custom);
    assert_eq!(applied.custom_index_milli, 1700);
}
