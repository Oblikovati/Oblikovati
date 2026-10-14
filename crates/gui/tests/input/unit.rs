// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use imgui::{Condition, Context};
#[test]
fn named_keys_and_buttons_have_unique_explicit_bindings() {
    let codes: std::collections::BTreeSet<u32> = GuiKey::ALL
        .iter()
        .map(|key| native_gui_key(*key) as u32)
        .collect();
    assert_eq!(codes.len(), GuiKey::ALL.len());
    assert_eq!(native_gui_key(GuiKey::S), Key::S);
    assert_eq!(native_gui_key(GuiKey::KeypadEnter), Key::KeypadEnter);
    assert_eq!(GuiPointerButton::Left.native(), MouseButton::Left);
    assert_eq!(GuiPointerButton::Middle.native(), MouseButton::Middle);
    assert_eq!(GuiPointerButton::Right.native(), MouseButton::Right);
}
#[test]
fn nonfinite_input_and_embedded_nul_are_refused() {
    for event in [
        GuiInputEvent::PointerMoved([f32::NAN, 0.]),
        GuiInputEvent::PointerMoved([f32::MAX, 0.]),
        GuiInputEvent::Wheel([0., f32::INFINITY]),
        GuiInputEvent::Text("bad\0text".into()),
    ] {
        assert!(validate_input(&event, [0.5; 2]).is_err());
    }
    validate_input(&GuiInputEvent::Text("λ😀".into()), [1.; 2]).unwrap();
    validate_input(&GuiInputEvent::Wheel([1., -2.]), [1.; 2]).unwrap();
}
#[test]
fn real_text_widget_consumes_unicode_keys_and_reports_channel_capture() {
    let mut context = input_probe_context();
    let mut text = String::new();
    focus_text_probe(&mut context, &mut text);
    verify_unicode_edit(&mut context, &mut text);
    verify_pointer_channels(&mut context, &mut text);
}
fn input_probe_context() -> Context {
    let mut context = Context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [400., 240.];
    context.io_mut().delta_time = 1. / 60.;
    context.fonts().build_rgba32_texture();
    context
}
fn focus_text_probe(context: &mut Context, text: &mut String) {
    widget_frame(context, &[], text, false);
    widget_frame(context, &[], text, true);
    // Focus becomes active on the next frame; typing follows that established focus.
    widget_frame(context, &[], text, false);
}
fn verify_unicode_edit(context: &mut Context, text: &mut String) {
    let captured = widget_frame(context, &[GuiInputEvent::Text("λ😀".into())], text, false);
    assert_eq!(text, "λ😀");
    assert!(captured.keyboard && captured.text);
    for pressed in [true, false] {
        let event = GuiInputEvent::Key {
            key: GuiKey::Backspace,
            pressed,
        };
        widget_frame(context, &[event], text, false);
        assert_eq!(text, "λ");
    }
}

fn widget_frame(
    context: &mut Context,
    events: &[GuiInputEvent],
    text: &mut String,
    focus: bool,
) -> GuiInputCapture {
    for event in events {
        apply_input(context.io_mut(), event, [2.; 2]);
    }
    let ui = context.frame();
    draw_text_probe(ui, text, focus);
    context.render();
    input_capture(context.io())
}
fn verify_pointer_channels(context: &mut Context, text: &mut String) {
    let events = pointer_probe_events();
    assert!(widget_frame(context, &events, text, false).pointer);
    assert_eq!(context.io().mouse_pos, [20., 50.]);
    assert!(context.io().mouse_down[1] && context.io().mouse_down[2]);
    // imgui-rs e246a334 keeps wheel/key events after button changes for the next frame.
    let ui = context.frame();
    assert_eq!([ui.io().mouse_wheel_h, ui.io().mouse_wheel], [1., -2.]);
    assert!(ui.io().key_ctrl && ui.io().key_shift && ui.io().key_alt && ui.io().key_super);
    draw_text_probe(ui, text, false);
    context.render();
}

fn draw_text_probe(ui: &imgui::Ui, text: &mut String, focus: bool) {
    ui.window("Input probe")
        .position([8., 8.], Condition::Always)
        .size([240., 160.], Condition::Always)
        .build(|| {
            if focus {
                ui.set_keyboard_focus_here();
            }
            ui.input_text("Value", text).build();
        });
}

fn pointer_probe_events() -> [GuiInputEvent; 5] {
    [
        GuiInputEvent::PointerMoved([40., 100.]),
        pointer_button_change(GuiPointerButton::Middle),
        pointer_button_change(GuiPointerButton::Right),
        GuiInputEvent::Wheel([1., -2.]),
        GuiInputEvent::Modifiers(GuiModifiers {
            control: true,
            shift: true,
            alt: true,
            super_key: true,
        }),
    ]
}
fn pointer_button_change(button: GuiPointerButton) -> GuiInputEvent {
    GuiInputEvent::PointerButton {
        button,
        pressed: true,
    }
}

#[test]
fn focus_loss_clears_held_keys_buttons_and_modifiers_in_frame_order() {
    let mut context = input_probe_context();
    let mut text = String::new();
    focus_text_probe(&mut context, &mut text);
    let held = [
        GuiInputEvent::Modifiers(GuiModifiers {
            control: true,
            ..Default::default()
        }),
        GuiInputEvent::Key {
            key: GuiKey::LeftCtrl,
            pressed: true,
        },
        GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        },
    ];
    widget_frame(&mut context, &held, &mut text, false);
    widget_frame(&mut context, &[], &mut text, false);
    assert!(context.io().key_ctrl && context.io().mouse_down[0]);
    apply_input(context.io_mut(), &GuiInputEvent::Focus(false), [1.; 2]);
    let ui = context.frame();
    assert!(ui.io().app_focus_lost);
    assert!(!ui.io().key_ctrl);
    assert!(!ui.io().mouse_down.iter().any(|down| *down));
    draw_text_probe(ui, &mut text, false);
    context.render();
    // Dear ImGui 1.89.2 clears the transient focus-loss flag in EndFrame.
    assert!(!context.io().app_focus_lost);
    widget_frame(
        &mut context,
        &[GuiInputEvent::Focus(true)],
        &mut text,
        false,
    );
    assert!(!context.io().app_focus_lost);
    assert!(!context.io().key_ctrl);
}
