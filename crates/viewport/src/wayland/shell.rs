// SPDX-License-Identifier: GPL-2.0-only
//! Own the xdg-shell lifecycle independently of the renderer's native window lease.
use super::{
    clipboard::ClipboardSelection, geometry::DrawableScale, input::SeatInput, wayland_error,
};
use crate::{ViewportEvent, ViewportWindowAction};
use smithay_client_toolkit::{
    activation::{ActivationHandler, ActivationState, RequestData},
    compositor::{CompositorHandler, CompositorState},
    delegate_activation, delegate_compositor, delegate_output, delegate_registry,
    delegate_xdg_shell, delegate_xdg_window,
    output::{OutputHandler, OutputState},
    registry::{ProvidesRegistryState, RegistryState},
    registry_handlers,
    seat::SeatState,
    shell::{
        WaylandSurface,
        xdg::{
            XdgShell, XdgSurface,
            window::{Window, WindowConfigure, WindowDecorations, WindowHandler},
        },
    },
};
use std::collections::VecDeque;
use wayland_client::{
    Connection, EventQueue, QueueHandle,
    globals::GlobalList,
    protocol::{wl_output, wl_surface},
};

pub(super) struct WindowState {
    pub(super) scale: DrawableScale,
    pub(super) shell: Window,
    pub(super) input: SeatInput,
    pub(super) clipboard: ClipboardSelection,
    pub(super) pending: VecDeque<ViewportEvent>,
    pub(super) visible: bool,
    pub(super) configured: bool,
    pub(super) maximized: bool,
    pub(super) title: String,
    pub(super) application_frame: bool,
    restored_size: [u32; 2],
    registry: RegistryState,
    output: OutputState,
    activation: Option<ActivationState>,
}

impl WindowState {
    pub(super) fn create(
        globals: &GlobalList,
        queue: &EventQueue<Self>,
        title: &str,
        extent: [u32; 2],
        system_frame: bool,
    ) -> Result<Self, String> {
        let qh = queue.handle();
        let compositor = CompositorState::bind(globals, &qh).map_err(wayland_error)?;
        let xdg = XdgShell::bind(globals, &qh).map_err(wayland_error)?;
        let surface = compositor.create_surface(&qh);
        let scale = DrawableScale::new(globals, &qh, &surface, extent);
        let shell = xdg.create_window(surface, decorations(system_frame), &qh);
        let input = SeatInput::new(globals, &qh, compositor)?;
        let clipboard = ClipboardSelection::new(globals, &qh, input.seats.seats().next());
        shell.set_title(title);
        shell.set_app_id("org.oblikovati.Oblikovati");
        shell.set_min_size((!system_frame).then_some((640, 480)));
        shell.commit();
        Ok(Self {
            scale,
            shell,
            input,
            clipboard,
            pending: VecDeque::new(),
            title: title.to_owned(),
            visible: true,
            configured: false,
            maximized: false,
            application_frame: !system_frame,
            restored_size: extent,
            registry: RegistryState::new(globals),
            output: OutputState::new(globals, &qh),
            activation: ActivationState::bind(globals, &qh).ok(),
        })
    }

    pub(super) fn is_our_surface(&self, surface: &wl_surface::WlSurface) -> bool {
        self.shell.wl_surface() == surface
    }

    pub(super) fn set_visible(&mut self, visible: bool) {
        self.visible = visible;
        self.configured = false;
        if !visible {
            self.shell.wl_surface().attach(None, 0, 0);
        }
        self.shell.commit();
        self.pending.push_back(ViewportEvent::Mapped(visible));
    }

    pub(super) fn resize_drawable(&mut self, extent: [u32; 2]) {
        let logical = self.scale.to_logical(extent);
        self.restored_size = logical;
        if !self.maximized {
            self.set_logical_size(logical);
        }
    }

    fn set_logical_size(&mut self, size: [u32; 2]) {
        self.scale.logical = size;
        let [width, height] = size;
        self.shell.set_window_geometry(0, 0, width, height);
        self.scale.apply(self.shell.wl_surface());
    }

    pub(super) fn action(&self, action: ViewportWindowAction) {
        match action {
            ViewportWindowAction::Minimize => self.shell.set_minimized(),
            ViewportWindowAction::ToggleMaximize if self.maximized => self.shell.unset_maximized(),
            ViewportWindowAction::ToggleMaximize => self.shell.set_maximized(),
            ViewportWindowAction::Move => {
                if let Some((seat, serial)) = &self.input.pointer_press {
                    self.shell.move_(seat, *serial);
                }
            }
        }
    }

    pub(super) fn request_focus(&self, queue: &QueueHandle<Self>) {
        if let Some(activation) = &self.activation {
            activation.request_token(
                queue,
                RequestData {
                    seat_and_serial: self.input.last_serial.clone(),
                    surface: Some(self.shell.wl_surface().clone()),
                    app_id: Some("org.oblikovati.Oblikovati".to_owned()),
                },
            );
        }
    }
}

fn decorations(system_frame: bool) -> WindowDecorations {
    if system_frame {
        WindowDecorations::RequestServer
    } else {
        WindowDecorations::RequestClient
    }
}

impl WindowHandler for WindowState {
    fn request_close(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &Window) {
        self.pending.push_back(ViewportEvent::CloseRequested);
    }

    fn configure(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &Window,
        configure: WindowConfigure,
        _: u32,
    ) {
        let maximized = configure.is_maximized();
        let fallback = if self.maximized && !maximized {
            self.restored_size
        } else {
            self.scale.logical
        };
        let extent = [
            configure.new_size.0.map_or(fallback[0], |side| side.get()),
            configure.new_size.1.map_or(fallback[1], |side| side.get()),
        ];
        let changed =
            !self.configured || self.maximized != maximized || self.scale.logical != extent;
        self.maximized = maximized;
        self.configured = true;
        self.set_logical_size(extent);
        if !maximized && !configure.is_fullscreen() && !configure.is_tiled() {
            self.restored_size = extent;
        }
        if changed {
            self.pending
                .push_back(ViewportEvent::Resized(self.scale.drawable()));
        }
    }
}

impl CompositorHandler for WindowState {
    fn scale_factor_changed(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        surface: &wl_surface::WlSurface,
        factor: i32,
    ) {
        if self.is_our_surface(surface) {
            self.change_integer_scale(factor);
        }
    }
    fn transform_changed(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_surface::WlSurface,
        _: wl_output::Transform,
    ) {
    }
    fn frame(&mut self, _: &Connection, _: &QueueHandle<Self>, _: &wl_surface::WlSurface, _: u32) {}
    fn surface_enter(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_surface::WlSurface,
        _: &wl_output::WlOutput,
    ) {
    }
    fn surface_leave(
        &mut self,
        _: &Connection,
        _: &QueueHandle<Self>,
        _: &wl_surface::WlSurface,
        _: &wl_output::WlOutput,
    ) {
    }
}

impl OutputHandler for WindowState {
    fn output_state(&mut self) -> &mut OutputState {
        &mut self.output
    }
    fn new_output(&mut self, _: &Connection, _: &QueueHandle<Self>, _: wl_output::WlOutput) {}
    fn update_output(&mut self, _: &Connection, _: &QueueHandle<Self>, _: wl_output::WlOutput) {}
    fn output_destroyed(&mut self, _: &Connection, _: &QueueHandle<Self>, _: wl_output::WlOutput) {}
}

impl ActivationHandler for WindowState {
    type RequestData = RequestData;

    fn new_token(&mut self, token: String, _: &Self::RequestData) {
        if let Some(activation) = &self.activation {
            activation.activate::<Self>(self.shell.wl_surface(), token);
        }
    }
}

impl ProvidesRegistryState for WindowState {
    fn registry(&mut self) -> &mut RegistryState {
        &mut self.registry
    }

    registry_handlers![OutputState, SeatState];
}

delegate_compositor!(WindowState);
delegate_output!(WindowState);
delegate_xdg_shell!(WindowState);
delegate_xdg_window!(WindowState);
delegate_activation!(WindowState);
delegate_registry!(WindowState);
