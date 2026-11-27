// SPDX-License-Identifier: GPL-2.0-only
//! The navigation bar beside the orientation control: Pan, Zoom and Orbit as running commands.
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
#[path = "navigation_buttons.rs"]
mod navigation_buttons;
use imgui::{Key, MouseButton, StyleColor, Ui};

#[path = "rotate_dialog.rs"]
mod rotate_dialog;
use rotate_dialog::DEFAULT_INCREMENT;

pub(super) const NAVIGATION_WINDOW: &str = "##navigation-bar";
pub(super) const NAVIGATION_SIZE: [f32; 2] = [30., 176.];
const ZOOM_MENU: &str = "##zoom-menu";
const ORBIT_MENU: &str = "##orbit-menu";
const VIEW_MENU: &str = "##navigation-menu";
const REPEAT_MENU: &str = "##navigation-repeat";
const BUTTON: [f32; 2] = [20., 22.];
/// A press that travels less than this is a click, not a drag.
const CLICK_REACH: f32 = 4.;
/// A pixel of drag, as a zoom factor and as an orbit turn in radians.
const ZOOM_PER_PIXEL: f64 = 1.01;
const ORBIT_PER_PIXEL: f64 = 0.01;
const GUIDE_SHARE: f32 = 0.35;
const GUIDE_MARK: f32 = 10.;
const GUIDE_COLOR: [f32; 4] = [0.85, 0.85, 0.9, 0.9];

/// A command the bar keeps running until it ends or is cancelled.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiNavigationMode {
    Pan,
    Zoom,
    Orbit,
    ConstrainedOrbit,
    ZoomWindow,
    ZoomSelected,
}
impl GuiNavigationMode {
    fn prompt(self) -> &'static str {
        match self {
            Self::Pan => "Pan: drag to pan the current view. Escape cancels.",
            Self::Zoom => "Zoom: drag to zoom in or out. Escape cancels.",
            Self::Orbit => "Free Rotation: drag to rotate the view. Escape cancels.",
            Self::ConstrainedOrbit => "Constrained Orbit: drag to turn the view. Escape cancels.",
            Self::ZoomWindow => "Zoom Window: select the window's first corner. Escape cancels.",
            Self::ZoomSelected => "Zoom Selected: select an entity to zoom. Escape cancels.",
        }
    }
}
/// What the bar asks of the view or the status area.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiViewCommand {
    /// The status area reads this prompt, or Ready when there is none.
    Prompt(Option<&'static str>),
    Pan([i32; 2]),
    Zoom(f64),
    Orbit {
        yaw: f64,
        pitch: f64,
    },
    /// Turn the screen about the viewer, in radians.
    Roll(f64),
    /// Remember this view, for a dialog that may put it back.
    MarkView,
    /// Put the remembered view back.
    RestoreMarkedView,
    FitAll,
    Previous,
    /// Frame the view on the rectangle between these pixels.
    ZoomWindow {
        corners: [[i32; 2]; 2],
        extent: [u32; 2],
    },
    /// Frame the view on the chosen face.
    ZoomSelected {
        extent: [u32; 2],
    },
    /// Frame the view on the face under this pixel, without choosing it.
    ZoomFaceAt {
        pixel: [i32; 2],
        extent: [u32; 2],
    },
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum ZoomAction {
    #[default]
    Zoom,
    All,
    Window,
    Selected,
}
impl ZoomAction {
    pub(crate) const ALL: [Self; 4] = [Self::Zoom, Self::All, Self::Window, Self::Selected];

    pub(crate) const fn label(self) -> &'static str {
        match self {
            Self::Zoom => "Zoom",
            Self::All => "Zoom All",
            Self::Window => "Zoom Window",
            Self::Selected => "Zoom Selected",
        }
    }
}
impl super::OrientationControl {
    pub(crate) fn run_zoom(
        &mut self,
        zoom: ZoomAction,
        request: &GuiFrameRequest,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.navigation
            .run_zoom_action(zoom, &request.commands, request.extent, actions);
    }
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum OrbitAction {
    #[default]
    Orbit,
    Constrained,
    RotateAtAngle,
}
#[derive(Default)]
pub(super) struct NavigationBar {
    wheel: super::navigation_wheel::NavigationWheel,
    mode: Option<GuiNavigationMode>,
    zoom_action: ZoomAction,
    orbit_action: OrbitAction,
    rotate_opening: bool,
    rotate_document: Option<u64>,
    rotate_dock: crate::dockable_lifecycle::DockableState,
    increment: Option<f32>,
    /// Where the current press began, and whether it has travelled far enough to be a drag.
    press: Option<([f32; 2], bool)>,
    window_corner: Option<[i32; 2]>,
}
impl crate::dockable_window::DockableWindow for NavigationBar {}
impl NavigationBar {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.wheel.register_dockable(registry)?;
        registry.register::<Self>("Incremental View Rotate", &self.rotate_dock)
    }
    pub(super) fn set_suspended(&self, suspended: bool) {
        self.wheel.set_suspended(suspended);
        self.rotate_dock.set_suspended(suspended);
    }
    fn is_rotating(&self) -> bool {
        matches!(
            self.mode,
            Some(GuiNavigationMode::Orbit | GuiNavigationMode::ConstrainedOrbit)
        )
    }
    /// A command is running, so a click on the view belongs to it; example: before selection.
    pub(super) fn is_running(&self) -> bool {
        self.mode.is_some() || self.wheel.is_open()
    }
    /// The wheel over the view; example: each frame, after the bar.
    pub(super) fn wheel_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.wheel.panel(ui, state, actions);
    }
    fn start(&mut self, mode: GuiNavigationMode, actions: &mut Vec<GuiShellAction>) {
        self.mode = Some(mode);
        self.window_corner = None;
        self.press = None;
        actions.push(GuiShellAction::ViewCommand(GuiViewCommand::Prompt(Some(
            mode.prompt(),
        ))));
    }
    fn end(&mut self, actions: &mut Vec<GuiShellAction>) {
        self.mode = None;
        self.window_corner = None;
        self.press = None;
        actions.push(GuiShellAction::ViewCommand(GuiViewCommand::Prompt(None)));
    }
    fn orbit_menu(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let Some(_menu) = ui.begin_popup(ORBIT_MENU) else {
            return;
        };
        for (label, action) in [
            ("Orbit", OrbitAction::Orbit),
            ("Constrained Orbit", OrbitAction::Constrained),
            ("Rotate at Angle", OrbitAction::RotateAtAngle),
        ] {
            let row = ui
                .menu_item_config(label)
                .selected(self.orbit_action == action)
                .build();
            if row {
                self.orbit_action = action;
                self.run_orbit_action(actions);
            }
        }
    }
    fn run_orbit_action(&mut self, actions: &mut Vec<GuiShellAction>) {
        match self.orbit_action {
            OrbitAction::Orbit => self.start(GuiNavigationMode::Orbit, actions),
            OrbitAction::Constrained => self.start(GuiNavigationMode::ConstrainedOrbit, actions),
            OrbitAction::RotateAtAngle => {
                self.rotate_opening = true;
                self.increment = Some(DEFAULT_INCREMENT);
                actions.push(GuiShellAction::ViewCommand(GuiViewCommand::MarkView));
            }
        }
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(super) fn needs_frame(&self) -> bool {
        self.rotate_opening || self.wheel.needs_frame()
    }
    fn zoom_menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        extent: [u32; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = ui.begin_popup(ZOOM_MENU) else {
            return;
        };
        for action in ZoomAction::ALL {
            let row = ui
                .menu_item_config(action.label())
                .selected(self.zoom_action == action)
                .build();
            if row {
                self.zoom_action = action;
                self.run_zoom_action(action, state, extent, actions);
            }
        }
    }
    fn run_zoom_action(
        &mut self,
        action: ZoomAction,
        state: &GuiCommandState,
        extent: [u32; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        match action {
            ZoomAction::Zoom => self.start(GuiNavigationMode::Zoom, actions),
            ZoomAction::Window => self.start(GuiNavigationMode::ZoomWindow, actions),
            ZoomAction::All => {
                actions.push(GuiShellAction::ViewCommand(GuiViewCommand::FitAll));
                self.end(actions);
            }
            ZoomAction::Selected if state.model_face_chosen => {
                let command = GuiViewCommand::ZoomSelected { extent };
                actions.push(GuiShellAction::ViewCommand(command));
                self.end(actions);
            }
            ZoomAction::Selected => self.start(GuiNavigationMode::ZoomSelected, actions),
        }
    }
    /// What the running command does with the pointer and keys over the view.
    pub(super) fn follow_view(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(mode) = self.mode else {
            if request.commands.authoring != crate::GuiAuthoringContext::Part {
                self.repeat_menu(ui);
            }
            return;
        };
        self.view_menu(ui, mode, actions);
        if crate::widget_abi::any_popup_open(ui) {
            return;
        }
        if ui.is_key_pressed_no_repeat(Key::Escape) {
            self.end(actions);
            return;
        }
        let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
        if !over_panel && ui.is_mouse_released(MouseButton::Right) {
            ui.open_popup(VIEW_MENU);
            return;
        }
        match mode {
            GuiNavigationMode::ZoomWindow => self.window_click(ui, request, over_panel, actions),
            GuiNavigationMode::ZoomSelected => self.face_click(ui, request, over_panel, actions),
            _ => self.drag(ui, mode, over_panel, actions),
        }
    }
    pub(super) fn owns_viewport_input(&self) -> bool {
        self.mode.is_some()
    }
    pub(super) fn can_repeat_wheel(&self) -> bool {
        self.wheel.can_repeat()
    }
    pub(super) fn repeat_wheel(&mut self) {
        self.wheel.request();
    }
    /// With nothing running, a right-click on the view offers the wheel that was last up.
    fn repeat_menu(&mut self, ui: &Ui) {
        let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
        if self.wheel.can_repeat()
            && !over_panel
            && !crate::widget_abi::any_popup_open(ui)
            && ui.is_mouse_released(MouseButton::Right)
        {
            ui.open_popup(REPEAT_MENU);
        }
        let Some(_menu) = ui.begin_popup(REPEAT_MENU) else {
            return;
        };
        if ui.menu_item("Repeat Navigation Wheel") {
            self.wheel.request();
        }
    }
    fn drag(
        &mut self,
        ui: &Ui,
        mode: GuiNavigationMode,
        over_panel: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let pointer = ui.io().mouse_pos;
        if !over_panel && ui.is_mouse_clicked(MouseButton::Left) {
            self.press = Some((pointer, false));
        }
        let Some((start, travelled)) = self.press else {
            return;
        };
        let moved = (pointer[0] - start[0]).hypot(pointer[1] - start[1]) >= CLICK_REACH;
        self.press = Some((start, travelled || moved));
        let [dx, dy] = ui.io().mouse_delta;
        if (dx, dy) != (0., 0.) && ui.is_mouse_down(MouseButton::Left) {
            actions.push(GuiShellAction::ViewCommand(drag_command(mode, [dx, dy])));
        }
        if ui.is_mouse_released(MouseButton::Left) {
            self.press = None;
            // A click without a drag ends Pan, as the recording shows.
            if mode == GuiNavigationMode::Pan && !(travelled || moved) {
                self.end(actions);
            }
        }
    }
    fn window_click(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        over_panel: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if over_panel || !ui.is_mouse_clicked(MouseButton::Left) {
            return;
        }
        let pixel = pointer_pixel(ui, request);
        let Some(first) = self.window_corner else {
            self.window_corner = Some(pixel);
            return;
        };
        let command = GuiViewCommand::ZoomWindow {
            corners: [first, pixel],
            extent: request.extent,
        };
        actions.push(GuiShellAction::ViewCommand(command));
        self.end(actions);
    }
    fn face_click(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        over_panel: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if over_panel || !ui.is_mouse_clicked(MouseButton::Left) {
            return;
        }
        let command = GuiViewCommand::ZoomFaceAt {
            pixel: pointer_pixel(ui, request),
            extent: request.extent,
        };
        actions.push(GuiShellAction::ViewCommand(command));
        self.end(actions);
    }
    fn view_menu(&mut self, ui: &Ui, mode: GuiNavigationMode, actions: &mut Vec<GuiShellAction>) {
        let Some(_menu) = ui.begin_popup(VIEW_MENU) else {
            return;
        };
        if ui.menu_item("Done") || ui.menu_item("OK") {
            self.end(actions);
        }
        if ui.menu_item("Home View") {
            actions.push(GuiShellAction::Home);
        }
        for (label, other) in [
            ("Pan", GuiNavigationMode::Pan),
            ("Zoom", GuiNavigationMode::Zoom),
            ("Orbit", GuiNavigationMode::Orbit),
        ] {
            let running = other == mode
                || (other == GuiNavigationMode::Orbit
                    && mode == GuiNavigationMode::ConstrainedOrbit);
            if !running && ui.menu_item(label) {
                self.start(other, actions);
            }
        }
        if ui.menu_item("Previous View") {
            actions.push(GuiShellAction::ViewCommand(GuiViewCommand::Previous));
        }
    }
    /// Free rotation shows its guide over the view: a ring, a cross and radial marks.
    pub(super) fn draw_guide(&self, ui: &Ui, center: [f32; 2], reach: f32) {
        if !self.is_rotating() {
            return;
        }
        let list = ui.get_background_draw_list();
        let radius = reach * GUIDE_SHARE;
        list.add_circle(center, radius, GUIDE_COLOR)
            .num_segments(64)
            .build();
        let [x, y] = center;
        list.add_line([x - GUIDE_MARK, y], [x + GUIDE_MARK, y], GUIDE_COLOR)
            .build();
        list.add_line([x, y - GUIDE_MARK], [x, y + GUIDE_MARK], GUIDE_COLOR)
            .build();
        for [across, down] in [[1., 0.], [-1., 0.], [0., 1.], [0., -1.]] {
            let outer = [x + across * radius, y + down * radius];
            let inner = [outer[0] - across * GUIDE_MARK, outer[1] - down * GUIDE_MARK];
            list.add_line(inner, outer, GUIDE_COLOR).build();
        }
    }
}
fn pointer_pixel(ui: &Ui, request: &GuiFrameRequest) -> [i32; 2] {
    let scale = request.layout_scale();
    let [x, y] = ui.io().mouse_pos;
    [(x * scale[0]) as i32, (y * scale[1]) as i32]
}
pub(super) fn drag_command(mode: GuiNavigationMode, [dx, dy]: [f32; 2]) -> GuiViewCommand {
    match mode {
        GuiNavigationMode::Pan => GuiViewCommand::Pan([dx as i32, dy as i32]),
        GuiNavigationMode::Orbit => GuiViewCommand::Orbit {
            yaw: f64::from(dx) * ORBIT_PER_PIXEL,
            pitch: f64::from(dy) * ORBIT_PER_PIXEL,
        },
        GuiNavigationMode::ConstrainedOrbit => GuiViewCommand::Orbit {
            yaw: f64::from(dx) * ORBIT_PER_PIXEL,
            pitch: 0.,
        },
        _ => GuiViewCommand::Zoom(ZOOM_PER_PIXEL.powf(f64::from(dy))),
    }
}

#[cfg(test)]
#[path = "navigation_bar_tests.rs"]
mod tests;
