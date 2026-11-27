// SPDX-License-Identifier: GPL-2.0-only
//! Hands the frame's input to Dear ImGui, keeping back what belongs to a command or a shortcut.
use crate::{
    GuiFrameRequest,
    input::{apply_input, sketch_tool_key},
};
use imgui::Context;

/// Keys the engine consumed before Dear ImGui could read them.
#[derive(Default)]
pub(crate) struct FrameKeys {
    pub tool_keyboard: bool,
    /// Characters no widget had the keyboard for, in the order they came.
    pub typed: String,
    pub switch_document: Option<crate::application::document_tabs::DocumentStep>,
}
pub(crate) fn prepare_frame_inputs(
    context: &mut Context,
    request: &GuiFrameRequest,
    held: &mut crate::GuiModifiers,
    popup_was_open: bool,
    key_tips_active: bool,
    retain_ribbon_appearance: bool,
    widget_keys: &mut Vec<crate::GuiKey>,
) -> FrameKeys {
    let io = context.io_mut();
    io.display_size =
        std::array::from_fn(|axis| request.extent[axis] as f32 / request.layout_scale()[axis]);
    io.display_framebuffer_scale = request.layout_scale();
    io.delta_time = request.delta_seconds;
    let mut keys = FrameKeys::default();
    for event in &request.events {
        if let crate::GuiInputEvent::Text(text) = event
            && !io.want_text_input
        {
            keys.typed.push_str(text);
        }
        if let crate::GuiInputEvent::Modifiers(modifiers) = event {
            *held = *modifiers;
        }
        if sketch_tool_key(
            event,
            request.commands.sketch_tool_keyboard && !popup_was_open && !key_tips_active,
            io.want_text_input,
            widget_keys,
        ) {
            keys.tool_keyboard = true;
            continue;
        }
        // Dear ImGui would read Ctrl+Tab as its own window switcher; here it switches documents.
        if document_switch_key(event, held.control) {
            if matches!(event, crate::GuiInputEvent::Key { pressed: true, .. }) {
                keys.switch_document = Some(document_step(*held));
            }
            continue;
        }
        // The sketch Create title's appearance menu stays open until pointer dismissal.
        if retain_ribbon_appearance
            && matches!(
                event,
                crate::GuiInputEvent::Key {
                    key: crate::GuiKey::Escape,
                    pressed: true
                }
            )
        {
            continue;
        }
        apply_input(io, event, request.layout_scale());
    }
    keys
}
fn document_step(held: crate::GuiModifiers) -> crate::application::document_tabs::DocumentStep {
    use crate::application::document_tabs::DocumentStep;
    if held.shift {
        DocumentStep::Previous
    } else {
        DocumentStep::Next
    }
}
fn document_switch_key(event: &crate::GuiInputEvent, control_held: bool) -> bool {
    control_held
        && matches!(
            event,
            crate::GuiInputEvent::Key {
                key: crate::GuiKey::Tab,
                ..
            }
        )
}
