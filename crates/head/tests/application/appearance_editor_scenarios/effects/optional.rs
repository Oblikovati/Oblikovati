// SPDX-License-Identifier: GPL-2.0-only
//! Verify the optional Generic appearance controls without activating the effects.
use super::*;

#[given(regex = r"^the (Cutouts|Self Illumination|Bump|Tint) effect is unchecked$")]
fn optional_unchecked(world: &mut ApplicationWorld, title: String) {
    let caption = title.replace(' ', "");
    if !world.window_text(EDITOR).contains(&caption) {
        scroll_properties(world, -4);
    }
    assert!(!world.checked_run(effect_checkbox(world, &caption)));
    clean_dialog(world);
    world.observed_rows = Some(caption);
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
}

#[when(regex = r"^I expand (Cutouts|Self Illumination|Bump|Tint)$")]
fn expand_optional(world: &mut ApplicationWorld, title: String) {
    let caption = title.replace(' ', "");
    click_in(world, EDITOR, &caption);
}

#[then(regex = r"^the section shows disabled (.+) controls$")]
fn optional_controls_disabled(world: &mut ApplicationWorld, labels: String) {
    let title = world.observed_rows.as_deref().unwrap().to_owned();
    let labels: Vec<_> = labels
        .replace(", ", " and ")
        .split(" and ")
        .map(|label| label.replace(' ', ""))
        .collect();
    if labels
        .iter()
        .any(|label| optional_control(world, &title, label).is_none())
    {
        scroll_properties(world, -4);
    }
    for label in labels {
        let run = optional_control(world, &title, &label).unwrap_or_else(|| {
            panic!(
                "expected {label} below {title}: {}",
                world.window_text(EDITOR)
            )
        });
        assert!(run.alpha < u8::MAX, "{title} {label} was enabled");
    }
}

fn optional_control<'a>(
    world: &'a ApplicationWorld,
    title: &str,
    label: &str,
) -> Option<&'a diagnostics::RenderedRun> {
    let header = world.run_in_window(EDITOR, title);
    let bottom = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.list == header.list
                && run.center()[1] > header.center()[1] + 6.
                && ["Cutouts", "SelfIllumination", "Bump", "Tint"].contains(&run.text.as_str())
        })
        .map(|run| run.center()[1])
        .min_by(f32::total_cmp)
        .unwrap_or_else(|| world.run_in_window(EDITOR, "Apply").center()[1]);
    world.frame.runs.iter().find(|run| {
        let y = run.center()[1];
        run.text == label && run.list == header.list && y > header.center()[1] && y < bottom
    })
}

#[then("the effect remains unchecked")]
fn disclosure_preserves_draft(world: &mut ApplicationWorld) {
    let title = world.observed_rows.as_deref().unwrap();
    assert!(!world.checked_run(effect_checkbox(world, title)));
    clean_dialog(world);
    assert!(owned_asset(world).appearance_metadata.is_none());
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    assert!(preview_pixels(&before, EDITOR).eq(preview_pixels(&after, EDITOR)));
}

#[test]
fn optional_bump_strength_survives_disabling_the_effect() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    optional_unchecked(&mut world, "Bump".into());
    expand_optional(&mut world, "Bump".into());
    optional_controls_disabled(&mut world, "Image and Amount".into());
    let at = diagnostics::pointer_pixels(effect_checkbox(&world, "Bump").center());
    world.click(at);
    world.pump(Vec::new());
    let amount = optional_control(&world, "Bump", "Amount").unwrap();
    assert_eq!(amount.alpha, u8::MAX);
    let at = diagnostics::pointer_pixels([amount.minimum[0] - 45., amount.center()[1]]);
    world.click(at);
    world.pump(Vec::new());
    can_apply(&mut world);
    apply(&mut world);
    let bump = owned_asset(&mut world)
        .appearance_metadata
        .unwrap()
        .generic_effects
        .unwrap()
        .bump;
    assert!(bump.enabled && bump.amount > 60, "{bump:?}");
    let at = diagnostics::pointer_pixels(effect_checkbox(&world, "Bump").center());
    world.click(at);
    world.pump(Vec::new());
    apply(&mut world);
    let disabled = owned_asset(&mut world)
        .appearance_metadata
        .unwrap()
        .generic_effects
        .unwrap()
        .bump;
    assert_eq!((disabled.enabled, disabled.amount), (false, bump.amount));
}
