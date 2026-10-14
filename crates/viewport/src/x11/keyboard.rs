// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportInput, ViewportKey, ViewportModifiers, ViewportPhysicalKey};
use std::cell::RefCell;
use std::collections::BTreeMap;
#[derive(Clone, Copy)]
struct HeldKeyboardKey {
    key: Option<ViewportPhysicalKey>,
    layout_key: Option<ViewportPhysicalKey>,
    command: Option<ViewportKey>,
}
pub(super) struct NativeKeyboard {
    input: text_channel::NativeTextChannel,
    keymap: RefCell<keyboard_map::NativeKeymap>,
    held: RefCell<BTreeMap<u32, HeldKeyboardKey>>,
}
impl NativeKeyboard {
    pub(super) fn open(
        api: Rc<NativeX11>,
        display: Handle<AspectDisplayConnection>,
        window: u64,
    ) -> Result<Self, String> {
        let owner = display.get_display_aspect() as *mut xlib::Display;
        require_detectable_repeat(&api.api, owner)?;
        let keymap = keyboard_map::NativeKeymap::load(&api.api, owner)?;
        Ok(Self {
            input: text_channel::NativeTextChannel::admit(
                api.input_locale
                    .clone()
                    .and_then(|_| text::NativeInputMethod::open(api, display, window)),
            ),
            keymap: RefCell::new(keymap),
            held: RefCell::new(BTreeMap::new()),
        })
    }
    pub(super) fn focus(&self, focused: bool) -> Result<(), String> {
        if !focused {
            self.held.borrow_mut().clear();
        }
        self.input.focus(focused);
        Ok(())
    }
    pub(super) fn text_input_issue(&self) -> Option<String> {
        self.input.issue()
    }
    pub(super) fn reload(&self, api: &xlib::Xlib, owner: *mut xlib::Display) -> Result<(), String> {
        let keymap = keyboard_map::NativeKeymap::load(api, owner)?;
        *self.keymap.borrow_mut() = keymap;
        Ok(())
    }
    pub(super) fn events(
        &self,
        api: &xlib::Xlib,
        original: xlib::XKeyEvent,
        translated: &mut xlib::XKeyEvent,
        filtered: bool,
    ) -> Result<Vec<ViewportEvent>, String> {
        let mut events = Vec::new();
        if let Some(input) = self.physical_input(api, original) {
            events.push(ViewportEvent::Input(input));
        }
        if original.type_ != xlib::KeyPress || filtered || !plain_commit(original.state) {
            return Ok(events);
        }
        events.extend(
            self.input
                .committed_text(translated)
                .chars()
                .map(|character| ViewportEvent::Input(ViewportInput::Text(character))),
        );
        Ok(events)
    }
    fn physical_input(&self, api: &xlib::Xlib, event: xlib::XKeyEvent) -> Option<ViewportInput> {
        // GLFW 3.4 XIM commits can carry keycode zero without a physical key transition.
        if event.keycode == 0 {
            return None;
        }
        let pressed = event.type_ == xlib::KeyPress;
        let retained = self.retain_key(event.keycode, pressed, self.resolve_key(api, event));
        let modifiers = keyboard_modifiers(event.state, retained.key, pressed, &self.held.borrow());
        match retained.key {
            Some(key) => Some(ViewportInput::PhysicalKey {
                key,
                layout_key: retained.layout_key.unwrap_or(key),
                command: retained.command,
                pressed,
                modifiers,
            }),
            None => retained
                .command
                .map(|key| ViewportInput::Key { key, pressed }),
        }
    }
    fn resolve_key(&self, api: &xlib::Xlib, event: xlib::XKeyEvent) -> HeldKeyboardKey {
        let key = self.keymap.borrow().key(event.keycode);
        // XKB.h XkbGroupForCoreState: group occupies state bits 13..14; level zero ignores Shift.
        let group = ((event.state >> 13) & 3) as i32;
        let symbol =
            unsafe { (api.XkbKeycodeToKeysym)(event.display, event.keycode as u8, group, 0) };
        HeldKeyboardKey {
            key,
            layout_key: key.map(|key| layout::named_layout_key(key, symbol)),
            command: input::copied_key(symbol as u32, event.state),
        }
    }
    fn retain_key(&self, code: u32, pressed: bool, current: HeldKeyboardKey) -> HeldKeyboardKey {
        let mut held = self.held.borrow_mut();
        if !pressed {
            return held.remove(&code).unwrap_or(HeldKeyboardKey {
                command: None,
                ..current
            });
        }
        if let Some(previous) = held.get(&code) {
            return HeldKeyboardKey {
                command: None,
                ..*previous
            };
        }
        held.insert(code, current);
        current
    }
}
fn require_detectable_repeat(api: &xlib::Xlib, owner: *mut xlib::Display) -> Result<(), String> {
    let mut supported = 0;
    let admitted = unsafe { (api.XkbSetDetectableAutoRepeat)(owner, 1, &mut supported) };
    if admitted == 0 || supported == 0 {
        return Err(format!(
            "XKB repeat admission {admitted}, support {supported}: expected detectable native key repeat"
        ));
    }
    Ok(())
}
fn plain_commit(state: u32) -> bool {
    // GLFW 3.4 permits plain characters separately from physical shortcut events; AltGr commits remain text.
    state & xlib::Mod5Mask != 0
        || state & (xlib::ControlMask | xlib::Mod1Mask | xlib::Mod4Mask) == 0
}
fn keyboard_modifiers(
    state: u32,
    key: Option<ViewportPhysicalKey>,
    pressed: bool,
    held: &BTreeMap<u32, HeldKeyboardKey>,
) -> ViewportModifiers {
    let mut modifiers = input::copied_modifiers(state);
    let Some(key) = key else {
        return modifiers;
    };
    use ViewportPhysicalKey::*;
    let remaining = |keys: &[ViewportPhysicalKey]| {
        held.values()
            .any(|held| held.key.is_some_and(|key| keys.contains(&key)))
    };
    match key {
        LeftCtrl | RightCtrl => modifiers.control = pressed || remaining(&[LeftCtrl, RightCtrl]),
        LeftShift | RightShift => modifiers.shift = pressed || remaining(&[LeftShift, RightShift]),
        LeftAlt | RightAlt => modifiers.alt = pressed || remaining(&[LeftAlt, RightAlt]),
        LeftSuper | RightSuper => {
            modifiers.super_key = pressed || remaining(&[LeftSuper, RightSuper])
        }
        _ => {}
    }
    modifiers
}

impl NativeWindow {
    pub(super) fn copy_event(&self, mut event: xlib::XEvent) -> Result<Vec<ViewportEvent>, String> {
        if !self.admit_input_event(&mut event)? {
            return Ok(Vec::new());
        }
        if let Some(focused) = keyboard_focus_transition(event.get_type()) {
            self.keyboard.focus(focused)?;
        }
        let original = event;
        let filtered =
            unsafe { (self.x11.api.XFilterEvent)(&mut event, self.window.native_handle()) } != 0;
        if matches!(original.get_type(), xlib::KeyPress | xlib::KeyRelease) {
            return self.keyboard.events(
                &self.x11.api,
                unsafe { original.key },
                unsafe { &mut event.key },
                filtered,
            );
        }
        Ok(self.translate(original).into_iter().collect())
    }
    fn admit_input_event(&self, event: &mut xlib::XEvent) -> Result<bool, String> {
        if unsafe { event.any.display } != self.pointer() {
            return Ok(false);
        }
        if event.get_type() == xlib::MappingNotify {
            unsafe { (self.x11.api.XRefreshKeyboardMapping)(&mut event.mapping) };
            self.keyboard.reload(&self.x11.api, self.pointer())?;
            return Ok(false);
        }
        if unsafe { event.any.window } != self.window.native_handle() {
            return Ok(false);
        }
        require_native_keycode(event)?;
        Ok(true)
    }
}
fn require_native_keycode(event: &xlib::XEvent) -> Result<(), String> {
    if !matches!(event.get_type(), xlib::KeyPress | xlib::KeyRelease) {
        return Ok(());
    }
    let code = unsafe { event.key.keycode };
    if code > u32::from(u8::MAX) {
        return Err(format!(
            "native keycode {code}: expected an 8-bit XKB keycode"
        ));
    }
    Ok(())
}

fn keyboard_focus_transition(kind: i32) -> Option<bool> {
    match kind {
        xlib::FocusIn => Some(true),
        xlib::FocusOut | xlib::UnmapNotify => Some(false),
        _ => None,
    }
}
