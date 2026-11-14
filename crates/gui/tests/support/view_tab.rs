// SPDX-License-Identifier: GPL-2.0-only
//! Select the View ribbon without depending on the toolbar width.
use oblikovati_gui::{GuiFrameEngine, GuiFrameRequest, GuiInputEvent, GuiKey};

/// Open View through its key tip; example: `select_view_tab(&mut engine, &mut request)`.
pub fn select_view_tab(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    for key in [GuiKey::LeftAlt, GuiKey::V] {
        for pressed in [true, false] {
            request.events = vec![GuiInputEvent::Key { key, pressed }];
            engine.frame(request).unwrap();
        }
    }
    request.events.clear();
}
