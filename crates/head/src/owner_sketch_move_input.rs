// SPDX-License-Identifier: GPL-2.0-only
//! Classify the pointer and cancel events admitted by sketch Move.
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};

/// Recognize a pressed cancel key; example: `is_cancel(event)` before handling a Move pick.
pub(super) fn is_cancel(event: ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true
        }) | ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Escape,
            pressed: true,
            ..
        })
    )
}
/// Admit sketch Move pointer input; example: `move_pointer(event)` returns a position and click flag.
pub(super) fn move_pointer(event: ViewportEvent) -> Option<([i32; 2], bool)> {
    match event {
        ViewportEvent::Input(ViewportInput::Motion { position, .. }) => Some((position, false)),
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position,
            modifiers,
        }) if !modifiers.alt && !modifiers.super_key => Some((position, true)),
        _ => None,
    }
}
