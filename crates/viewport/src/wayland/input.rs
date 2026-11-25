// SPDX-License-Identifier: GPL-2.0-only
//! Translate Wayland seat input without depending on an X11 connection.
mod evdev;
mod pointer;

use super::{shell::WindowState, wayland_error};
use crate::{ViewportEvent, ViewportInput, ViewportModifiers};
use smithay_client_toolkit::{
    compositor::CompositorState,
    delegate_keyboard, delegate_pointer, delegate_seat, delegate_shm,
    seat::{
        Capability, SeatHandler, SeatState,
        keyboard::{
            KeyEvent, KeyboardData, KeyboardHandler, Keysym, Modifiers, RawModifiers, RepeatInfo,
        },
        pointer::{CursorIcon, PointerData, ThemedPointer},
    },
    shm::{Shm, ShmHandler},
};
use std::time::{Duration, Instant};
use wayland_client::{
    Connection, Proxy, QueueHandle,
    globals::GlobalList,
    protocol::{wl_keyboard, wl_seat, wl_surface},
};

pub(super) struct SeatInput {
    pub(super) seats: SeatState,
    compositor: CompositorState,
    shm: Shm,
    keyboard: Option<wl_keyboard::WlKeyboard>,
    pointer: Option<ThemedPointer<PointerData>>,
    modifiers: ViewportModifiers,
    repeat: RepeatInfo,
    repeating: Option<RepeatingKey>,
    cursor: Option<CursorIcon>,
    scroll_remainder: f64,
    resize_grab: bool,
    pub(super) pointer_press: Option<(wl_seat::WlSeat, u32)>,
    pub(super) last_serial: Option<(wl_seat::WlSeat, u32)>,
    pub(super) keyboard_issue: Option<String>,
}

struct RepeatingKey {
    code: u32,
    text: String,
    next: Instant,
    interval: Duration,
}

impl SeatInput {
    pub(super) fn new(
        globals: &GlobalList,
        queue: &QueueHandle<WindowState>,
        compositor: CompositorState,
    ) -> Result<Self, String> {
        Ok(Self {
            seats: SeatState::new(globals, queue),
            shm: Shm::bind(globals, queue).map_err(wayland_error)?,
            compositor,
            keyboard: None,
            pointer: None,
            modifiers: ViewportModifiers::default(),
            repeat: RepeatInfo::Disable,
            repeating: None,
            cursor: None,
            scroll_remainder: 0.,
            resize_grab: false,
            pointer_press: None,
            last_serial: None,
            keyboard_issue: None,
        })
    }
    pub(super) fn next_repeat(&self) -> Option<Duration> {
        self.repeating
            .as_ref()
            .map(|key| key.next.saturating_duration_since(Instant::now()))
    }
}

impl WindowState {
    fn key_event(&mut self, event: &KeyEvent, pressed: bool) {
        let modifiers = self.input.modifiers;
        if let Some(key) = evdev::key(event.raw_code) {
            let symbol = event.keysym.raw();
            self.send_input(ViewportInput::PhysicalKey {
                key,
                layout_key: crate::x11::layout::named_layout_key(key, symbol.into()),
                command: crate::x11::input::xkb_command(symbol, modifiers),
                pressed,
                modifiers,
            });
        }
        if pressed
            && !modifiers.control
            && !modifiers.super_key
            && let Some(text) = event.utf8.as_deref()
        {
            self.send_text(text);
            self.begin_text_repeat(event.raw_code, text);
        }
    }

    fn send_input(&mut self, input: ViewportInput) {
        self.pending.push_back(ViewportEvent::Input(input));
    }

    fn send_text(&mut self, text: &str) {
        for letter in text.chars().filter(|letter| !letter.is_control()) {
            self.send_input(ViewportInput::Text(letter));
        }
    }

    fn begin_text_repeat(&mut self, code: u32, text: &str) {
        if let RepeatInfo::Repeat { rate, delay } = self.input.repeat
            && text.chars().any(|letter| !letter.is_control())
        {
            self.input.repeating = Some(RepeatingKey {
                code,
                text: text.to_owned(),
                next: Instant::now() + Duration::from_millis(u64::from(delay)),
                interval: Duration::from_secs_f64(1. / f64::from(rate.get())),
            });
        }
    }

    pub(super) fn publish_repeat(&mut self) {
        let now = Instant::now();
        let Some(key) = self.input.repeating.as_mut().filter(|key| key.next <= now) else {
            return;
        };
        key.next = now + key.interval;
        let text = key.text.clone();
        self.send_text(&text);
    }

    fn observe_keyboard_serial(&mut self, keyboard: &wl_keyboard::WlKeyboard, serial: u32) {
        if let Some(data) = keyboard.data::<KeyboardData<Self>>() {
            self.input.last_serial = Some((data.seat().clone(), serial));
        }
    }
}

impl SeatHandler for WindowState {
    fn seat_state(&mut self) -> &mut SeatState {
        &mut self.input.seats
    }
    fn new_seat(&mut self, _: &Connection, queue: &QueueHandle<Self>, seat: wl_seat::WlSeat) {
        self.clipboard.register_seat(queue, seat);
    }
    fn remove_seat(&mut self, _: &Connection, _: &QueueHandle<Self>, seat: wl_seat::WlSeat) {
        self.clipboard.remove_seat(&seat);
    }

    fn new_capability(
        &mut self,
        _: &Connection,
        queue: &QueueHandle<Self>,
        seat: wl_seat::WlSeat,
        capability: Capability,
    ) {
        match capability {
            Capability::Keyboard if self.input.keyboard.is_none() => {
                match self.input.seats.get_keyboard(queue, &seat, None) {
                    Ok(keyboard) => self.input.keyboard = Some(keyboard),
                    Err(error) => self.input.keyboard_issue = Some(error.to_string()),
                }
            }
            Capability::Pointer if self.input.pointer.is_none() => {
                self.input.pointer = self
                    .input
                    .seats
                    .get_pointer_with_theme(
                        queue,
                        &seat,
                        self.input.shm.wl_shm(),
                        self.input.compositor.create_surface(queue),
                        smithay_client_toolkit::seat::pointer::ThemeSpec::System,
                    )
                    .ok();
            }
            _ => {}
        }
    }

    fn remove_capability(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        seat: wl_seat::WlSeat,
        capability: Capability,
    ) {
        if capability == Capability::Keyboard
            && let Some(keyboard) = self.input.keyboard.as_ref()
            && keyboard
                .data::<KeyboardData<Self>>()
                .is_some_and(|data| data.seat() == &seat)
        {
            if keyboard.version() >= 3 {
                keyboard.release();
            }
            self.input.keyboard = None;
            self.input.repeating = None;
        }
        if capability == Capability::Pointer
            && let Some(pointer) = self.input.pointer.as_ref()
            && pointer
                .pointer()
                .data::<PointerData>()
                .is_some_and(|data| data.seat() == &seat)
        {
            self.input.pointer = None;
            self.input.pointer_press = None;
        }
    }
}

impl KeyboardHandler for WindowState {
    fn enter(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        surface: &wl_surface::WlSurface,
        _: u32,
        _: &[u32],
        _: &[Keysym],
    ) {
        if self.is_our_surface(surface) {
            self.pending.push_back(ViewportEvent::Focus(true));
        }
    }

    fn leave(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        surface: &wl_surface::WlSurface,
        _: u32,
    ) {
        if self.is_our_surface(surface) {
            self.input.repeating = None;
            self.input.modifiers = ViewportModifiers::default();
            self.pending.push_back(ViewportEvent::Focus(false));
        }
    }

    fn press_key(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        keyboard: &wl_keyboard::WlKeyboard,
        serial: u32,
        event: KeyEvent,
    ) {
        self.observe_keyboard_serial(keyboard, serial);
        self.input.repeating = None;
        self.key_event(&event, true);
    }

    fn repeat_key(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        _: u32,
        event: KeyEvent,
    ) {
        if let Some(text) = event.utf8 {
            self.send_text(&text);
        }
    }

    fn release_key(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        _: u32,
        event: KeyEvent,
    ) {
        if self
            .input
            .repeating
            .as_ref()
            .is_some_and(|key| key.code == event.raw_code)
        {
            self.input.repeating = None;
        }
        self.key_event(&event, false);
    }

    fn update_modifiers(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        _: u32,
        mods: Modifiers,
        _: RawModifiers,
        _: u32,
    ) {
        self.input.modifiers = ViewportModifiers {
            shift: mods.shift,
            control: mods.ctrl,
            alt: mods.alt,
            super_key: mods.logo,
        };
    }

    fn update_repeat_info(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_keyboard::WlKeyboard,
        info: RepeatInfo,
    ) {
        self.input.repeat = info;
        self.input.repeating = None;
    }
}

impl ShmHandler for WindowState {
    fn shm_state(&mut self) -> &mut Shm {
        &mut self.input.shm
    }
}

delegate_seat!(WindowState);
delegate_keyboard!(WindowState);
delegate_pointer!(WindowState);
delegate_shm!(WindowState);
