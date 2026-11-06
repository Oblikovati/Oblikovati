// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::ui_scale::UiScaleOptions;
fn scale_request(font: f32, backing: f32) -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [(1440. * backing) as u32, (900. * backing) as u32],
        events: Vec::new(),
        commands: GuiCommandState {
            ui_scale: Some(UiScaleOptions::new(font, 1.5).unwrap()),
            ..Default::default()
        },
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [backing; 2],
    }
}
fn settle(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    request.events.clear();
    for _ in 0..32 {
        let output = owner.frame(request).unwrap();
        if !output.activity.queued_input
            && !output.activity.pending_layout
            && !output.activity.pending_navigation
        {
            return;
        }
    }
    panic!("scale frame: expected layout to settle within 32 frames");
}
fn click(
    owner: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerMoved(point)];
    let mut actions = owner.frame(request).unwrap().actions;
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton {
            button: crate::GuiPointerButton::Left,
            pressed,
        }];
        actions.extend(owner.frame(request).unwrap().actions);
    }
    settle(owner, request);
    actions
}
#[test]
fn content_scale_composes_with_real_backing_density_for_draws_and_hit_regions() {
    for backing in [1., 2.] {
        let mut owner = GuiFrameEngine::new().unwrap();
        let mut request = scale_request(2., backing);
        settle(&mut owner, &mut request);
        assert_eq!(request.framebuffer_scale, [backing; 2]);
        assert_eq!(request.layout_scale(), [2. * backing; 2]);
        assert_eq!(
            click(&mut owner, &mut request, [100. * backing, 180. * backing]),
            [GuiShellAction::NewPart]
        );
        #[cfg(target_os = "linux")]
        {
            let draws = owner.frame(&request).unwrap().draws;
            oblikovati_viewport::GuiFrameCompiler::new(owner.atlas())
                .unwrap()
                .compile(&draws)
                .unwrap();
        }
    }
}
#[test]
fn font_changes_replace_the_owned_atlas_once_and_restore_default_geometry() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let initial = owner.atlas().clone();
    let mut request = scale_request(2., 1.);
    settle(&mut owner, &mut request);
    let enlarged = owner.atlas().clone();
    assert!(!Arc::ptr_eq(&initial.rgba, &enlarged.rgba));
    assert!(enlarged.rgba.len() > initial.rgba.len());
    owner.frame(&request).unwrap();
    assert!(Arc::ptr_eq(&enlarged.rgba, &owner.atlas().rgba));
    request.commands.ui_scale = Some(UiScaleOptions::default());
    settle(&mut owner, &mut request);
    assert!(
        owner.atlas() == &initial,
        "restored scale: expected byte-identical default atlas"
    );
    assert_eq!(
        click(&mut owner, &mut request, [50., 90.]),
        [GuiShellAction::NewPart]
    );
}
#[test]
fn excessive_raster_sizes_are_refused_before_altering_the_live_atlas() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let initial = owner.atlas().clone();
    let request = scale_request(2., 100.);
    let cause = owner.frame(&request).unwrap_err();
    assert!(cause.contains("GUI font size 2600") && cause.contains("expected finite size"));
    assert!(Arc::ptr_eq(&initial.rgba, &owner.atlas().rgba));
}
#[test]
fn preferences_choose_text_scale_without_changing_the_stored_icon_factor() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = scale_request(1., 1.);
    settle(&mut owner, &mut request);
    assert!(click(&mut owner, &mut request, [55., 10.]).is_empty());
    assert!(click(&mut owner, &mut request, [100., 32.]).is_empty());
    let actions = click(&mut owner, &mut request, [530., 514.]);
    assert_eq!(
        actions,
        [GuiShellAction::SetUiScale(
            UiScaleOptions::new(2., 1.5).unwrap()
        )]
    );
}

#[test]
fn narrow_two_hundred_percent_content_keeps_pointer_access_to_the_view_centre() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = scale_request(2., 1.);
    request.extent = [640, 900];
    settle(&mut owner, &mut request);
    request.events = vec![GuiInputEvent::PointerMoved([320., 700.])];
    assert!(!owner.frame(&request).unwrap().capture.pointer);
}
