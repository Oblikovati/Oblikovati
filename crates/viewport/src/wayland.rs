// SPDX-License-Identifier: GPL-2.0-only
//! Oblikovati owns the Wayland shell surface and Martensite owns its Vulkan presentation.
pub(crate) mod clipboard;
mod geometry;
mod input;
mod shell;

use crate::{ViewportEvent, ViewportWindowAction, types::valid_extent};
use martensite_tkernel::standard::Handle;
use martensite_tkservice::{aspect::AspectDisplayConnection, wl::WlWindow};
use smithay_client_toolkit::shell::WaylandSurface;
use std::{
    cell::RefCell,
    os::fd::{AsFd, BorrowedFd},
    rc::Rc,
    time::Duration,
};
use wayland_client::{Connection, EventQueue, Proxy, globals::registry_queue_init};

pub(crate) struct NativeWaylandWindow {
    pub(crate) window: Handle<WlWindow>,
    events: Rc<RefCell<WaylandEvents>>,
    connection: Connection,
}

struct WaylandEvents {
    queue: EventQueue<shell::WindowState>,
    state: shell::WindowState,
}

impl NativeWaylandWindow {
    pub(crate) fn create(
        connection: Connection,
        title: &str,
        extent: [u32; 2],
        system_frame: bool,
    ) -> Result<Self, String> {
        valid_extent(extent)?;
        crate::x11::validate_title(title)?;
        let (globals, mut queue) = registry_queue_init(&connection).map_err(wayland_error)?;
        let mut state = shell::WindowState::create(&globals, &queue, title, extent, system_frame)?;
        queue.roundtrip(&mut state).map_err(wayland_error)?;
        let window = Handle::new(wrap_surface(&connection, &state)?);
        let native = Self {
            window,
            events: Rc::new(RefCell::new(WaylandEvents { queue, state })),
            connection,
        };
        native.refresh_drawable();
        Ok(native)
    }

    pub(crate) fn backing_scale(&self) -> f32 {
        self.events.borrow().state.scale.factor()
    }

    pub(crate) fn is_mapped(&self) -> bool {
        self.window.fields.read().is_mapped
    }

    pub(crate) fn mapped(&self, mapped: bool) -> Result<(), String> {
        let mut events = self.events.borrow_mut();
        if events.state.visible == mapped {
            return Ok(());
        }
        events.state.set_visible(mapped);
        events.roundtrip()?;
        drop(events);
        self.refresh_drawable();
        Ok(())
    }

    pub(crate) fn resize(&self, extent: [u32; 2]) -> Result<(), String> {
        valid_extent(extent)?;
        self.events.borrow_mut().state.resize_drawable(extent);
        self.refresh_drawable();
        self.flush()
    }

    pub(crate) fn poll(&self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.events.borrow_mut();
        events.receive()?;
        let count = limit.min(events.state.pending.len());
        let pending = events.state.pending.drain(..count).collect();
        drop(events);
        self.refresh_drawable();
        Ok(pending)
    }

    pub(crate) fn has_pending_input(&self) -> Result<bool, String> {
        let mut events = self.events.borrow_mut();
        events.dispatch()?;
        Ok(!events.state.pending.is_empty())
    }

    pub(crate) fn connection_descriptor(&self) -> BorrowedFd<'_> {
        self.connection.as_fd()
    }

    pub(crate) fn next_input_timeout(&self) -> Option<Duration> {
        let events = self.events.borrow();
        events
            .state
            .input
            .next_repeat()
            .into_iter()
            .chain(events.state.clipboard.write_timeout())
            .min()
    }

    pub(crate) fn request_accessible_focus(&self) -> Result<(), String> {
        let events = self.events.borrow();
        events.state.request_focus(&events.queue.handle());
        self.flush()
    }

    pub(crate) fn window_action(&self, action: ViewportWindowAction) -> Result<(), String> {
        self.events.borrow().state.action(action);
        self.flush()
    }

    pub(crate) fn window_maximized(&self) -> Result<bool, String> {
        Ok(self.events.borrow().state.maximized)
    }

    pub(crate) fn window_screen_bounds(&self) -> Result<[i32; 4], String> {
        // Wayland reports coordinates relative to the surface.
        let [width, height] = self.events.borrow().state.scale.drawable();
        Ok([0, 0, width as i32, height as i32])
    }

    pub(crate) fn set_title(&self, title: &str) -> Result<(), String> {
        crate::x11::validate_title(title)?;
        let mut events = self.events.borrow_mut();
        events.state.shell.set_title(title);
        title.clone_into(&mut events.state.title);
        self.flush()
    }

    pub(crate) fn title(&self) -> Result<String, String> {
        Ok(self.events.borrow().state.title.clone())
    }

    pub(crate) fn text_input_issue(&self) -> Option<String> {
        self.events.borrow().state.input.keyboard_issue.clone()
    }

    fn refresh_drawable(&self) {
        let events = self.events.borrow();
        let [width, height] = events.state.scale.drawable();
        let mut fields = self.window.fields.write();
        fields.width = width as i32;
        fields.height = height as i32;
        fields.is_mapped = events.state.visible && events.state.configured;
    }

    fn flush(&self) -> Result<(), String> {
        self.connection.flush().or_else(ignore_would_block)
    }
}

fn wrap_surface(connection: &Connection, state: &shell::WindowState) -> Result<WlWindow, String> {
    let display = connection.backend().display_ptr() as usize;
    let surface = state.shell.wl_surface().id().as_ptr() as usize as u64;
    let display =
        Handle::new(AspectDisplayConnection::with_wayland_display(display).map_err(wayland_error)?);
    let [width, height] = valid_extent(state.scale.drawable())?;
    WlWindow::wrap(&display, surface, width, height).map_err(wayland_error)
}

impl WaylandEvents {
    fn dispatch(&mut self) -> Result<(), String> {
        self.queue
            .dispatch_pending(&mut self.state)
            .map_err(wayland_error)?;
        Ok(())
    }

    fn roundtrip(&mut self) -> Result<(), String> {
        self.queue
            .roundtrip(&mut self.state)
            .map_err(wayland_error)?;
        Ok(())
    }

    fn receive(&mut self) -> Result<(), String> {
        self.dispatch()?;
        self.queue.flush().or_else(ignore_would_block)?;
        if let Some(read) = self.queue.prepare_read() {
            let fd = read.connection_fd();
            let mut fds = [rustix::event::PollFd::new(
                &fd,
                rustix::event::PollFlags::IN,
            )];
            let ready = rustix::event::poll(&mut fds, Some(&rustix::event::Timespec::default()))
                .map_err(wayland_error)?;
            if ready > 0 {
                read.read().map(|_| ()).or_else(ignore_would_block)?;
            }
        }
        self.dispatch()?;
        self.state.publish_repeat();
        self.state.clipboard.service_writes();
        Ok(())
    }
}

fn ignore_would_block(error: wayland_client::backend::WaylandError) -> Result<(), String> {
    use wayland_client::backend::WaylandError;
    match error {
        WaylandError::Io(cause) if cause.kind() == std::io::ErrorKind::WouldBlock => Ok(()),
        error => Err(wayland_error(error)),
    }
}

fn wayland_error(cause: impl std::fmt::Display) -> String {
    format!("Wayland desktop: {cause}")
}
