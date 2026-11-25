// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::icons::{IconGlyph, IconSize};

fn ribbon_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1440, 900],
        events: Vec::new(),
        commands: Default::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
fn settled_frame(owner: &mut GuiFrameEngine, request: &GuiFrameRequest) -> GuiFrame {
    for _ in 0..32 {
        let output = owner.frame(request).unwrap();
        if !output.activity.pending_layout
            && !output.activity.queued_input
            && !output.activity.pending_navigation
        {
            return output.draws;
        }
    }
    panic!("icon ribbon exceeded 32 frames: expected a stable measured layout");
}
#[test]
fn zero_document_ribbon_submits_the_owned_new_part_icon_and_caption() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let frame = settled_frame(&mut owner, &ribbon_request());
    let key = owner
        .icons
        .presentation
        .key(IconGlyph::NewPart, IconSize::Large);
    assert!(
        frame.batches.iter().any(|batch| batch.texture == key),
        "expected an actual New Part icon draw batch"
    );
    assert!(
        frame
            .batches
            .iter()
            .any(|batch| batch.texture == crate::GuiTextureKey::FONT)
    );
    assert_eq!(
        frame.resources.source(key).unwrap().pixels().extent,
        [40; 2]
    );
}

#[test]
fn modeling_and_sketch_ribbons_submit_the_extrude_and_move_icons() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Part;
    let model = settled_frame(&mut owner, &request);
    assert_named_icon(&owner, &model, IconGlyph::Extrude, IconSize::Small, 18.);
    request.commands.authoring = crate::GuiAuthoringContext::Sketch;
    let sketch = settled_frame(&mut owner, &request);
    assert_named_icon(&owner, &sketch, IconGlyph::Move, IconSize::Small, 18.);
}

#[test]
fn tools_ribbon_submits_the_go_measure_icon() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Part;
    settled_frame(&mut owner, &request);
    owner.ribbon.select_tab(crate::ribbon::RibbonTab::Tools);
    for _ in 0..4 {
        owner.frame(&request).unwrap();
    }
    let frame = settled_frame(&mut owner, &request);
    assert_named_icon(&owner, &frame, IconGlyph::Measure, IconSize::Large, 40.);
}

#[test]
fn ribbon_appearance_controls_actual_work_feature_icon_size() {
    use crate::ribbon_command_layout::RibbonAppearance;
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Part;
    for (appearance, size, pixels) in [
        (RibbonAppearance::Normal, IconSize::Small, 18.),
        (RibbonAppearance::TextOff, IconSize::Large, 40.),
        (RibbonAppearance::Small, IconSize::Small, 18.),
        (RibbonAppearance::Compact, IconSize::Small, 18.),
        (RibbonAppearance::Large, IconSize::Large, 40.),
        (RibbonAppearance::Normal, IconSize::Small, 18.),
    ] {
        owner.ribbon.customization.select_appearance(appearance);
        let frame = settled_frame(&mut owner, &request);
        assert_named_icon(&owner, &frame, IconGlyph::WorkPlane, size, pixels);
    }
}

#[test]
fn sketch_appearance_keeps_create_and_constraint_icons_in_the_frame() {
    use crate::ribbon_command_layout::RibbonAppearance;
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Sketch;
    for (appearance, size, pixels) in [
        (RibbonAppearance::TextOff, IconSize::Large, 40.),
        (RibbonAppearance::Small, IconSize::Small, 18.),
        (RibbonAppearance::Compact, IconSize::Small, 18.),
        (RibbonAppearance::Large, IconSize::Large, 40.),
    ] {
        owner.ribbon.customization.select_appearance(appearance);
        let frame = settled_frame(&mut owner, &request);
        for glyph in [
            IconGlyph::Rectangle,
            IconGlyph::Circle,
            IconGlyph::Horizontal,
            IconGlyph::Dimension,
            IconGlyph::Construction,
            IconGlyph::Centerline,
            IconGlyph::CenterPoint,
            IconGlyph::SketchFillet,
            IconGlyph::SketchText,
        ] {
            assert_named_icon(&owner, &frame, glyph, size, pixels);
        }
    }
}

#[test]
fn model_ribbon_submits_icons_for_work_features_and_patterns() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Part;
    let frame = settled_frame(&mut owner, &request);
    for glyph in [
        IconGlyph::WorkPlane,
        IconGlyph::WorkAxis,
        IconGlyph::WorkPoint,
        IconGlyph::RectangularPattern,
        IconGlyph::CircularPattern,
        IconGlyph::Simplify,
    ] {
        assert_named_icon(&owner, &frame, glyph, IconSize::Small, 18.);
    }
}

#[test]
fn actual_icon_quads_follow_independent_scale_and_backing_density() {
    for (text, icon, backing, expected) in [
        (1., 1., 1., [40., 18.]),
        (1., 1., 2., [80., 36.]),
        (1., 2., 1., [80., 36.]),
        (2., 1., 1., [40., 18.]),
        (1., 1.1, 1., [44., 20.]),
    ] {
        check_icon_quad_scale(text, icon, backing, expected);
    }
}
fn check_icon_quad_scale(text: f32, icon: f32, backing: f32, expected: [f32; 2]) {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = scaled_request(text, icon, backing);
    let frame = settled_frame(&mut owner, &request);
    assert_named_icon(
        &owner,
        &frame,
        IconGlyph::NewPart,
        IconSize::Large,
        expected[0],
    );
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Sketch;
    let sketch = settled_frame(&mut owner, &request);
    assert_named_icon(
        &owner,
        &sketch,
        IconGlyph::Circle,
        IconSize::Small,
        expected[1],
    );
}
fn scaled_request(text: f32, icon: f32, backing: f32) -> GuiFrameRequest {
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.framebuffer_scale = [backing; 2];
    request.commands.ui_scale =
        Some(oblikovati_model::ui_scale::UiScaleOptions::new(text, icon).unwrap());
    request
}
fn assert_named_icon(
    owner: &GuiFrameEngine,
    frame: &GuiFrame,
    glyph: IconGlyph,
    size: IconSize,
    expected: f32,
) {
    let key = owner.icons.presentation.key(glyph, size);
    let batch = frame
        .batches
        .iter()
        .find(|batch| batch.texture == key)
        .expect("expected submitted icon quad");
    assert_eq!(icon_quad_size(batch), [expected; 2]);
    assert_eq!(
        frame.resources.source(key).unwrap().pixels().extent,
        [expected as u32; 2]
    );
    assert!(
        batch
            .vertices
            .iter()
            .all(|vertex| vertex.position[1] >= batch.clip_rect[1]
                && vertex.position[1] <= batch.clip_rect[3])
    );
}
fn icon_quad_size(batch: &crate::GuiBatch) -> [f32; 2] {
    std::array::from_fn(|axis| {
        let minimum = batch
            .vertices
            .iter()
            .map(|vertex| vertex.position[axis])
            .fold(f32::INFINITY, f32::min);
        let maximum = batch
            .vertices
            .iter()
            .map(|vertex| vertex.position[axis])
            .fold(f32::NEG_INFINITY, f32::max);
        maximum - minimum
    })
}
#[test]
fn themed_frames_retain_their_own_icon_colors_after_the_owner_is_dropped() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    let first = settled_frame(&mut owner, &request);
    let key = owner
        .icons
        .presentation
        .key(IconGlyph::NewPart, IconSize::Large);
    let light = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Light").unwrap();
    request.commands.theme = Some(crate::GuiThemeState::from_library(&light));
    let second = settled_frame(&mut owner, &request);
    drop(owner);
    assert_ne!(
        first.resources.source(key).unwrap().pixels(),
        second.resources.source(key).unwrap().pixels()
    );
    #[cfg(target_os = "linux")]
    for frame in [&first, &second] {
        oblikovati_viewport::GuiFrameCompiler::with_resources(&frame.resources)
            .unwrap()
            .compile(frame)
            .unwrap();
    }
}

#[test]
fn startup_theme_colors_apply_when_the_frame_has_no_theme_snapshot() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let request = ribbon_request();
    let first = settled_frame(&mut owner, &request);
    let key = owner
        .icons
        .presentation
        .key(IconGlyph::NewPart, IconSize::Large);
    let light = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Light").unwrap();
    owner.apply_theme(light.active());
    let second = settled_frame(&mut owner, &request);
    assert_ne!(
        first.resources.source(key).unwrap().pixels(),
        second.resources.source(key).unwrap().pixels()
    );
}

#[test]
fn a_long_small_caption_is_visible_beside_its_full_size_icon() {
    let mut owner = GuiFrameEngine::new().unwrap();
    settled_frame(&mut owner, &ribbon_request());
    small_caption_frame(&mut owner);
    let (frame, bounds) = small_caption_frame(&mut owner);
    let ink: Vec<_> = frame
        .batches
        .iter()
        .flat_map(|batch| &batch.vertices)
        .filter(|vertex| vertex.srgb == [51, 102, 153, 255])
        .collect();
    assert_eq!(ink.len(), "Perpendicular".len() * 4);
    assert!(
        ink.iter()
            .all(|vertex| vertex.position[0] >= bounds[0] + 18. && vertex.position[0] <= bounds[2])
    );
}
fn small_caption_frame(owner: &mut GuiFrameEngine) -> (GuiFrame, [f32; 4]) {
    let ui = owner.context.frame();
    crate::widget_abi::window_geometry(ui, [0.; 2], [400., 300.], imgui::Condition::Always);
    let mut bounds = [0.; 4];
    ui.window("Small caption").build(|| {
        crate::widget_abi::cursor_position(ui, [31., 47.]);
        let _color = ui.push_style_color(imgui::StyleColor::Text, [0.2, 0.4, 0.6, 1.]);
        owner
            .icons
            .presentation
            .button(ui, "Perpendicular", IconGlyph::Perpendicular, [108., 18.]);
        let lo = ui.item_rect_min();
        let hi = ui.item_rect_max();
        bounds = [lo[0], lo[1], hi[0], hi[1]];
    });
    let frame = capture_frame(owner.context.render(), [1440, 900], &owner.resources).unwrap();
    (frame, bounds)
}

#[test]
fn cancelling_a_theme_preview_restores_the_startup_icon_colors() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    let first = settled_frame(&mut owner, &request);
    let key = owner
        .icons
        .presentation
        .key(IconGlyph::NewPart, IconSize::Large);
    let light = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Light").unwrap();
    request.commands.theme = Some(crate::GuiThemeState::from_library(&light));
    let preview = settled_frame(&mut owner, &request);
    request.commands.theme = None;
    let cancelled = settled_frame(&mut owner, &request);
    assert_ne!(
        first.resources.source(key).unwrap().pixels(),
        preview.resources.source(key).unwrap().pixels()
    );
    assert_eq!(
        first.resources.source(key).unwrap().pixels(),
        cancelled.resources.source(key).unwrap().pixels()
    );
}

#[test]
fn recoloring_icons_does_not_restart_the_measured_ribbon_layout() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    let initial = settled_frame(&mut owner, &request);
    let key = owner
        .icons
        .presentation
        .key(IconGlyph::NewPart, IconSize::Large);
    let light = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Light").unwrap();
    request.commands.theme = Some(crate::GuiThemeState::from_library(&light));
    let preview = owner.frame(&request).unwrap();
    let initial_icon = initial
        .batches
        .iter()
        .find(|batch| batch.texture == key)
        .unwrap();
    let preview_icon = preview
        .draws
        .batches
        .iter()
        .find(|batch| batch.texture == key)
        .unwrap();
    assert_eq!(initial_icon.vertices, preview_icon.vertices);
    assert!(
        preview.activity.pending_layout,
        "dock tabs need a theme refresh frame"
    );
    assert!(!owner.frame(&request).unwrap().activity.pending_layout);
}
