// SPDX-License-Identifier: GPL-2.0-only
//! Bounded UTF-8 transfers over the focused Wayland seat's data device.
use super::{NativeWaylandWindow, WaylandEvents, ignore_would_block, shell::WindowState};
use crate::ViewportClipboard;
use rustix::{
    event::{PollFd, PollFlags, Timespec, poll},
    fs::{OFlags, fcntl_getfl, fcntl_setfl},
};
use smithay_client_toolkit::{
    data_device_manager::{
        DataDeviceManagerState, ReadPipe, WritePipe,
        data_device::{DataDevice, DataDeviceHandler},
        data_offer::{DataOfferHandler, DragOffer},
        data_source::{CopyPasteSource, DataSourceHandler},
    },
    delegate_data_device,
};
use std::{
    cell::RefCell,
    io::{self, Read, Write},
    os::fd::AsFd,
    rc::{Rc, Weak},
    time::{Duration, Instant},
};
use wayland_client::{
    Connection, QueueHandle,
    globals::GlobalList,
    protocol::{
        wl_data_device::WlDataDevice, wl_data_device_manager::DndAction,
        wl_data_source::WlDataSource, wl_seat::WlSeat, wl_surface::WlSurface,
    },
};

const TEXT_TARGETS: &[&str] = &["text/plain;charset=utf-8", "UTF8_STRING", "text/plain"];
const TRANSFER_TIMEOUT: Duration = Duration::from_secs(3);
const TRANSFER_LIMIT: usize = 16;

pub(super) struct ClipboardSelection {
    manager: Option<DataDeviceManagerState>,
    device: Option<(WlSeat, DataDevice)>,
    published: Option<(CopyPasteSource, Rc<str>)>,
    writes: Vec<PendingWrite>,
}

struct PendingWrite {
    pipe: WritePipe,
    text: Rc<str>,
    sent: usize,
    deadline: Instant,
}

impl ClipboardSelection {
    pub(super) fn new(
        globals: &GlobalList,
        queue: &QueueHandle<WindowState>,
        seat: Option<WlSeat>,
    ) -> Self {
        let mut selection = Self {
            manager: DataDeviceManagerState::bind(globals, queue).ok(),
            device: None,
            published: None,
            writes: Vec::new(),
        };
        if let Some(seat) = seat {
            selection.register_seat(queue, seat);
        }
        selection
    }

    pub(super) fn register_seat(&mut self, queue: &QueueHandle<WindowState>, seat: WlSeat) {
        if self.device.is_none()
            && let Some(manager) = &self.manager
        {
            self.device = Some((seat.clone(), manager.get_data_device(queue, &seat)));
        }
    }

    pub(super) fn remove_seat(&mut self, seat: &WlSeat) {
        if self
            .device
            .as_ref()
            .is_some_and(|(active, _)| active == seat)
        {
            self.device = None;
            self.published = None;
        }
    }

    fn offer_pipe(&self) -> Result<Option<ReadPipe>, String> {
        let Some((_, device)) = &self.device else {
            return Err("Wayland clipboard: expected a data device on the active seat".into());
        };
        let Some(offer) = device.data().selection_offer() else {
            return Ok(None);
        };
        let mime = offer.with_mime_types(preferred_text_target);
        mime.map(|mime| offer.receive(mime).map_err(clipboard_error))
            .transpose()
    }

    fn publish(
        &mut self,
        queue: &QueueHandle<WindowState>,
        seat: &WlSeat,
        serial: u32,
        text: &str,
    ) -> Result<(), String> {
        let Some(manager) = &self.manager else {
            return Err(
                "Wayland clipboard: compositor did not advertise wl_data_device_manager".into(),
            );
        };
        let Some((_, device)) = self.device.as_ref().filter(|(owner, _)| owner == seat) else {
            return Err("Wayland clipboard: expected a data device on the active seat".into());
        };
        let source = manager.create_copy_paste_source(queue, TEXT_TARGETS.iter().copied());
        source.set_selection(device, serial);
        self.published = Some((source, Rc::from(text)));
        Ok(())
    }

    pub(super) fn service_writes(&mut self) {
        self.writes
            .retain_mut(|transfer| transfer.advance(Instant::now()));
    }

    pub(super) fn write_timeout(&self) -> Option<Duration> {
        (!self.writes.is_empty()).then_some(Duration::from_millis(5))
    }
}

impl PendingWrite {
    fn advance(&mut self, now: Instant) -> bool {
        if now >= self.deadline {
            return false;
        }
        match self.pipe.write(&self.text.as_bytes()[self.sent..]) {
            Ok(written) => {
                self.sent += written;
                written > 0 && self.sent < self.text.len()
            }
            Err(cause) => matches!(
                cause.kind(),
                io::ErrorKind::WouldBlock | io::ErrorKind::Interrupted
            ),
        }
    }
}

/// An owner-thread clipboard handle for a live Wayland viewport.
pub struct WaylandClipboard {
    events: Weak<RefCell<WaylandEvents>>,
    connection: Connection,
    byte_limit: usize,
}

impl NativeWaylandWindow {
    pub(crate) fn clipboard(&self, byte_limit: usize) -> WaylandClipboard {
        WaylandClipboard {
            events: Rc::downgrade(&self.events),
            connection: self.connection.clone(),
            byte_limit,
        }
    }
}

impl ViewportClipboard for WaylandClipboard {
    fn read_text(&self) -> Result<Option<String>, String> {
        let events = self
            .events
            .upgrade()
            .ok_or("Wayland clipboard: expected a live window")?;
        let mut window = events.borrow_mut();
        window.receive()?;
        if let Some((_, text)) = &window.state.clipboard.published {
            return Ok(Some(text.to_string()));
        }
        let Some(pipe) = window.state.clipboard.offer_pipe()? else {
            return Ok(None);
        };
        self.connection.flush().or_else(ignore_would_block)?;
        drop(window);
        read_offer(pipe, self.byte_limit).map(Some)
    }

    fn write_text(&self, text: &str) -> Result<(), String> {
        if text.len() > self.byte_limit {
            return Err(format!(
                "Wayland clipboard text length {} exceeds {} bytes",
                text.len(),
                self.byte_limit
            ));
        }
        let events = self
            .events
            .upgrade()
            .ok_or("Wayland clipboard: expected a live window")?;
        let mut window = events.borrow_mut();
        let (seat, serial) = window
            .state
            .input
            .last_serial
            .clone()
            .ok_or("Wayland clipboard: expected a compositor input serial")?;
        let queue = window.queue.handle();
        window
            .state
            .clipboard
            .publish(&queue, &seat, serial, text)?;
        self.connection.flush().or_else(ignore_would_block)
    }
}

fn preferred_text_target(mimes: &[String]) -> Option<String> {
    TEXT_TARGETS.iter().find_map(|target| {
        mimes
            .iter()
            .find(|mime| mime.eq_ignore_ascii_case(target))
            .cloned()
    })
}

fn nonblocking(pipe: impl AsFd) -> Result<(), String> {
    let flags = fcntl_getfl(&pipe).map_err(clipboard_error)?;
    fcntl_setfl(&pipe, flags | OFlags::NONBLOCK).map_err(clipboard_error)
}

fn read_offer(mut pipe: impl Read + AsFd, limit: usize) -> Result<String, String> {
    nonblocking(&pipe)?;
    let deadline = Instant::now() + TRANSFER_TIMEOUT;
    let mut text = Vec::new();
    let mut chunk = [0; 8192];
    loop {
        match pipe.read(&mut chunk) {
            Ok(0) => return String::from_utf8(text).map_err(clipboard_error),
            Ok(count) if count <= limit.saturating_sub(text.len()) => {
                text.extend_from_slice(&chunk[..count])
            }
            Ok(_) => return Err(format!("Wayland clipboard selection exceeds {limit} bytes")),
            Err(cause) if cause.kind() == io::ErrorKind::WouldBlock => {
                wait_for_pipe(&pipe, deadline)?
            }
            Err(cause) if cause.kind() == io::ErrorKind::Interrupted => continue,
            Err(cause) => return Err(clipboard_error(cause)),
        }
    }
}

fn wait_for_pipe(pipe: impl AsFd, deadline: Instant) -> Result<(), String> {
    let remaining = deadline.saturating_duration_since(Instant::now());
    let timeout = Timespec {
        tv_sec: remaining.as_secs() as i64,
        tv_nsec: remaining.subsec_nanos().into(),
    };
    let mut fds = [PollFd::new(&pipe, PollFlags::IN)];
    if remaining.is_zero() || poll(&mut fds, Some(&timeout)).map_err(clipboard_error)? == 0 {
        return Err("Wayland clipboard selection timed out".into());
    }
    Ok(())
}

fn clipboard_error(cause: impl std::fmt::Display) -> String {
    format!("Wayland clipboard: {cause}")
}

impl DataSourceHandler for WindowState {
    fn accept_mime(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &WlDataSource,
        _: Option<String>,
    ) {
    }

    fn send_request(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        source: &WlDataSource,
        mime: String,
        pipe: WritePipe,
    ) {
        let Some((owner, text)) = &self.clipboard.published else {
            return;
        };
        if owner.inner() == source
            && TEXT_TARGETS.contains(&mime.as_str())
            && self.clipboard.writes.len() < TRANSFER_LIMIT
            && nonblocking(&pipe).is_ok()
        {
            self.clipboard.writes.push(PendingWrite {
                pipe,
                text: text.clone(),
                sent: 0,
                deadline: Instant::now() + TRANSFER_TIMEOUT,
            });
        }
    }

    fn cancelled(&mut self, _: &Connection, _: &QueueHandle<Self>, source: &WlDataSource) {
        if self
            .clipboard
            .published
            .as_ref()
            .is_some_and(|(owner, _)| owner.inner() == source)
        {
            self.clipboard.published = None;
        }
    }
    fn dnd_dropped(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataSource) {}
    fn dnd_finished(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataSource) {}
    fn action(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataSource, _: DndAction) {}
}

impl DataDeviceHandler for WindowState {
    fn enter(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &WlDataDevice,
        _: f64,
        _: f64,
        _: &WlSurface,
    ) {
    }
    fn leave(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataDevice) {}
    fn motion(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataDevice, _: f64, _: f64) {}
    fn selection(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataDevice) {}
    fn drop_performed(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &WlDataDevice) {}
}

impl DataOfferHandler for WindowState {
    fn source_actions(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &mut DragOffer,
        _: DndAction,
    ) {
    }
    fn selected_action(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &mut DragOffer,
        _: DndAction,
    ) {
    }
}

delegate_data_device!(WindowState);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pipe_reads_whole_utf8_payloads_and_rejects_oversized_transfers() {
        for (limit, expected) in [(4, None), (5, Some("Café"))] {
            let (read, write) = rustix::pipe::pipe().unwrap();
            std::fs::File::from(write)
                .write_all("Café".as_bytes())
                .unwrap();
            let result = read_offer(std::fs::File::from(read), limit);
            assert_eq!(result.ok().as_deref(), expected);
        }
    }
}
