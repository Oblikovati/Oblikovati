// SPDX-License-Identifier: GPL-2.0-only
//! The navigation wheel: a floating ring of view tools, opened from the navigation bar.
use super::navigation_bar::{GuiNavigationMode, GuiViewCommand, drag_command};
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, MouseButton, StyleColor, Ui, WindowFlags};
#[path = "wheel_decoration.rs"]
mod wheel_decoration;

const WINDOW: &str = "##navigation-wheel";
const MENU: &str = "##wheel-menu";
const OPTIONS_TITLE: &str = "Navigation Wheel Options";
const OPTIONS_SIZE: [f32; 2] = [420., 420.];
/// The ring's side at Normal; the other big sizes scale from it.
const FULL_SIDE: f32 = 220.;
const BASIC_SIDE: f32 = 240.;
const MINI_SIDE: f32 = 64.;
const BIG_SCALES: [f32; 3] = [0.75, 1., 1.3];
const MINI_SCALES: [f32; 4] = [0.75, 1., 1.3, 1.7];
const OUTER_REACH: f32 = 0.38;
const INNER_REACH: f32 = 0.16;
const RING_SHARE: f32 = 0.48;
const HUB_SHARE: f32 = 0.27;
const RING_COLOR: [f32; 3] = [0.2, 0.23, 0.28];
const HOVER_FILL: [f32; 4] = [0.2, 0.55, 0.3, 0.95];
const SEGMENT_REACH: f32 = 0.5;
const MINI_SIZES: [&str; 4] = ["Small", "Normal", "Large", "Extra Large"];
const BIG_SIZES: [&str; 3] = ["Small", "Normal", "Large"];
const OPACITIES: [&str; 4] = ["25%", "50%", "75%", "90%"];
const OPACITY_ALPHAS: [f32; 4] = [0.25, 0.5, 0.75, 0.9];

/// Which ring of tools the wheel shows.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum WheelLayout {
    MiniViewObject,
    MiniTourBuilding,
    MiniFull,
    #[default]
    Full,
    BasicViewObject,
    BasicTourBuilding,
}
impl WheelLayout {
    const ALL: [Self; 6] = [
        Self::MiniViewObject,
        Self::MiniTourBuilding,
        Self::MiniFull,
        Self::Full,
        Self::BasicViewObject,
        Self::BasicTourBuilding,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::MiniViewObject => "Mini View Object Wheel",
            Self::MiniTourBuilding => "Mini Tour Building Wheel",
            Self::MiniFull => "Mini Full Navigation Wheel",
            Self::Full => "Full Navigation Wheel",
            Self::BasicViewObject => "Basic View Object Wheel",
            Self::BasicTourBuilding => "Basic Tour Building Wheel",
        }
    }
    fn side(self) -> f32 {
        match self {
            Self::Full => FULL_SIDE,
            Self::BasicViewObject | Self::BasicTourBuilding => BASIC_SIDE,
            _ => MINI_SIDE,
        }
    }
    fn is_mini(self) -> bool {
        self.side() == MINI_SIDE
    }
    /// The ring's tools, clockwise from the top: at the rim, then round the hub.
    fn tools(self) -> (&'static [&'static str], &'static [&'static str]) {
        match self {
            Self::Full => (
                &["Zoom", "Orbit", "Pan", "Rewind"],
                &["Center", "Walk", "Look", "Up Down"],
            ),
            Self::MiniFull => (
                &[
                    "Zoom", "Pan", "Orbit", "Rewind", "Center", "Walk", "Look", "Up Down",
                ],
                &[],
            ),
            Self::BasicViewObject => (&["Center", "Zoom", "Rewind", "Orbit"], &[]),
            Self::BasicTourBuilding => (&["Forward", "Look", "Rewind", "Up Down"], &[]),
            Self::MiniViewObject | Self::MiniTourBuilding => {
                (&["Zoom", "Rewind", "Pan", "Orbit"], &[])
            }
        }
    }
}
/// Which way a tool sits on the ring, by its place among `count`, clockwise from the top.
fn tool_direction(place: usize, count: usize) -> [f32; 2] {
    let angle = std::f32::consts::TAU * place as f32 / count as f32;
    [angle.sin(), -angle.cos()]
}
/// The tool at a pointer offset from the middle, as a share of the wheel's side.
fn tool_at(layout: WheelLayout, [x, y]: [f32; 2]) -> Option<&'static str> {
    let reach = x.hypot(y);
    if reach > SEGMENT_REACH {
        return None;
    }
    let (rim, hub) = layout.tools();
    let in_hub = reach < HUB_SHARE && !hub.is_empty();
    let tools = if in_hub { hub } else { rim };
    let angle = x.atan2(-y).rem_euclid(std::f32::consts::TAU);
    let share = std::f32::consts::TAU / tools.len() as f32;
    let place = if in_hub {
        angle / share
    } else {
        (angle + share / 2.) / share
    };
    let place = place as usize % tools.len();
    Some(tools[place])
}
#[derive(Clone, Copy, Debug, PartialEq)]
struct WheelOptions {
    tool_messages: bool,
    tool_tips: bool,
    mini_size: usize,
    big_size: usize,
    opacity: usize,
    invert_look: bool,
    zoom_to_pointer: bool,
    keep_up_while_orbiting: bool,
    walk_on_plane: bool,
    walk_speed: f32,
}
impl DockableWindow for NavigationWheel {}
impl Default for WheelOptions {
    fn default() -> Self {
        Self {
            tool_messages: true,
            tool_tips: true,
            mini_size: 1,
            big_size: 1,
            opacity: 1,
            invert_look: false,
            zoom_to_pointer: false,
            keep_up_while_orbiting: true,
            walk_on_plane: true,
            walk_speed: 1.,
        }
    }
}
#[derive(Default)]
pub(super) struct NavigationWheel {
    open: bool,
    used: bool,
    /// Follows the pointer until a click sets it down.
    placed: Option<[f32; 2]>,
    layout: WheelLayout,
    options: WheelOptions,
    editing: WheelOptions,
    options_opening: bool,
    options_visible: crate::dockable_lifecycle::DockableState,
    dragging: Option<GuiNavigationMode>,
    popup_was_open: bool,
}
impl NavigationWheel {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(OPTIONS_TITLE, &self.options_visible)
    }
    pub(super) fn set_suspended(&self, suspended: bool) {
        self.options_visible.set_suspended(suspended);
    }
    pub(super) fn request(&mut self) {
        self.open = true;
        self.used = true;
        self.placed = None;
        self.options_visible.set_open(false);
        self.options_opening = false;
        self.popup_was_open = false;
    }
    /// It has been up since the part opened, so the view's menu can bring it back.
    pub(super) fn can_repeat(&self) -> bool {
        self.used && !self.open
    }
    /// The wheel is up, so a click on the view belongs to it.
    pub(super) fn is_open(&self) -> bool {
        self.open
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.options_opening
    }
    /// Draw while open; example: each frame, over the view.
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open {
            return;
        }
        let pointer = ui.io().mouse_pos;
        let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
        if self.placed.is_none() && !over_panel && ui.is_mouse_clicked(MouseButton::Left) {
            self.placed = Some(pointer);
        }
        let center = self.placed.unwrap_or(pointer);
        let side = self.side();
        // While it follows the pointer, a click must reach whatever lies under it.
        let flags = if self.placed.is_some() {
            wheel_flags()
        } else {
            wheel_flags() | WindowFlags::NO_INPUTS
        };
        ui.window(WINDOW)
            .position(
                [center[0] - side / 2., center[1] - side / 2.],
                imgui::Condition::Always,
            )
            .size([side, side], imgui::Condition::Always)
            .flags(flags)
            .build(|| self.contents(ui, state, center, side, actions));
        let options_were_open = self.options_visible.is_open() || self.options_opening;
        self.options_dialog(ui);
        let popup_open = crate::widget_abi::any_popup_open(ui);
        if !options_were_open && !self.popup_was_open && !popup_open {
            self.close_on_escape(ui);
        }
        self.popup_was_open = popup_open;
        if !self.open {
            actions.push(GuiShellAction::ViewCommand(GuiViewCommand::Prompt(None)));
        }
    }
    fn side(&self) -> f32 {
        let scale = if self.layout.is_mini() {
            MINI_SCALES[self.options.mini_size]
        } else {
            BIG_SCALES[self.options.big_size]
        };
        self.layout.side() * scale
    }
    fn close_on_escape(&mut self, ui: &Ui) {
        if !crate::widget_abi::any_popup_open(ui) && ui.is_key_pressed_no_repeat(Key::Escape) {
            self.open = false;
        }
    }
    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        center: [f32; 2],
        side: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let offset = [
            (ui.io().mouse_pos[0] - center[0]) / side,
            (ui.io().mouse_pos[1] - center[1]) / side,
        ];
        let under = ui
            .is_window_hovered()
            .then(|| tool_at(self.layout, offset))
            .flatten();
        self.draw(ui, center, side, under);
        let cursor = ui.cursor_screen_pos();
        if !self.layout.is_mini() {
            self.wheel_controls(ui, center, side);
        }
        ui.set_cursor_screen_pos(cursor);
        ui.invisible_button("##wheel-area", [side, side]);
        let hovered = ui.is_item_hovered();
        let right_clicked = ui.is_item_clicked_with_button(MouseButton::Right);
        let under = hovered.then(|| tool_at(self.layout, offset)).flatten();
        self.press_tools(ui, under, actions);
        if right_clicked {
            ui.open_popup(MENU);
        }
        self.context_menu(ui, state, actions);
    }
    fn draw(&self, ui: &Ui, center: [f32; 2], side: f32, under: Option<&str>) {
        let list = ui.get_window_draw_list();
        let [red, green, blue] = RING_COLOR;
        let fill = [red, green, blue, OPACITY_ALPHAS[self.options.opacity]];
        list.add_circle(center, side * RING_SHARE, fill)
            .num_segments(48)
            .filled(true)
            .build();
        let (rim, hub) = self.layout.tools();
        if self.layout.is_mini() {
            wheel_decoration::draw_mini_segments(
                &list,
                center,
                side,
                rim.len(),
                under.and_then(|tool| rim.iter().position(|row| *row == tool)),
            );
            drop(list);
            if let Some(tool) = under.filter(|_| self.options.tool_tips) {
                ui.tooltip_text(tool);
            }
            return;
        }
        wheel_decoration::draw_partitions(&list, center, side, rim.len(), !hub.is_empty());
        let text = ui.style_color(StyleColor::Text);
        for (tools, reach) in [(rim, OUTER_REACH), (hub, INNER_REACH)] {
            for (place, tool) in tools.iter().enumerate() {
                let [across, down] = if reach == INNER_REACH {
                    wheel_decoration::hub_direction(place, tools.len())
                } else {
                    tool_direction(place, tools.len())
                };
                let at = [
                    center[0] + across * side * reach,
                    center[1] + down * side * reach,
                ];
                let [width, height] = ui.calc_text_size(tool);
                if under == Some(*tool) {
                    list.add_circle(at, height, HOVER_FILL)
                        .num_segments(16)
                        .filled(true)
                        .build();
                    list.add_circle(at, height + 2., text)
                        .num_segments(16)
                        .build();
                }
                list.add_text([at[0] - width / 2., at[1] - height / 2.], text, tool);
            }
        }
    }
    fn wheel_controls(&mut self, ui: &Ui, center: [f32; 2], side: f32) {
        let top = center[1] - side / 2. + 4.;
        ui.set_cursor_screen_pos([center[0] + side / 2. - 48., top]);
        if ui.small_button("v##wheel-menu-button") {
            ui.open_popup(MENU);
        }
        ui.same_line();
        if ui.small_button("x##wheel-close-button") {
            self.open = false;
        }
    }
    /// A press on a view tool drags as that tool does; the rest have nothing to move yet.
    fn press_tools(&mut self, ui: &Ui, under: Option<&str>, actions: &mut Vec<GuiShellAction>) {
        if ui.is_mouse_clicked(MouseButton::Left) {
            self.dragging = match under {
                Some("Pan") => Some(GuiNavigationMode::Pan),
                Some("Zoom") => Some(GuiNavigationMode::Zoom),
                Some("Orbit") => Some(GuiNavigationMode::Orbit),
                _ => None,
            };
            if under == Some("Rewind") {
                actions.push(GuiShellAction::ViewCommand(GuiViewCommand::Previous));
            }
            if under == Some("Center") {
                actions.push(GuiShellAction::ViewCommand(GuiViewCommand::FitAll));
            }
        }
        if !ui.is_mouse_down(MouseButton::Left) {
            self.dragging = None;
            return;
        }
        let [dx, dy] = ui.io().mouse_delta;
        if let (Some(mode), true) = (self.dragging, (dx, dy) != (0., 0.)) {
            actions.push(GuiShellAction::ViewCommand(drag_command(mode, [dx, dy])));
        }
    }
    fn context_menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = ui.begin_popup(MENU) else {
            return;
        };
        for layout in WheelLayout::ALL {
            let row = ui
                .menu_item_config(layout.label())
                .selected(self.layout == layout)
                .build();
            if row {
                self.layout = layout;
            }
        }
        ui.separator();
        if ui.menu_item("Go Home") && state.document.is_some() {
            actions.push(GuiShellAction::Home);
        }
        if ui.menu_item("Fit to Window") {
            actions.push(GuiShellAction::ViewCommand(GuiViewCommand::FitAll));
        }
        muted_row(ui, "Restore Original Center");
        ui.separator();
        if ui.menu_item("Options") {
            self.editing = self.options;
            self.options_opening = true;
        }
        if ui.menu_item("Close Wheel") {
            self.open = false;
        }
        muted_row(ui, "Help Topics");
    }
    fn options_dialog(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.options_opening) {
            self.options_visible.set_open(true);
        }
        if !self.options_visible.is_open() {
            return;
        }
        Self::center_dockable(ui, OPTIONS_TITLE, OPTIONS_SIZE);
        let mut open = true;
        let mut close = false;
        Self::dockable_window(ui, OPTIONS_TITLE, &mut open).build(|| {
            ui.checkbox("Show Tool Messages", &mut self.editing.tool_messages);
            ui.checkbox("Show Tool Tips", &mut self.editing.tool_tips);
            choice_list(
                ui,
                "Mini Wheel Size",
                &MINI_SIZES,
                &mut self.editing.mini_size,
            );
            choice_list(ui, "Big Wheel Size", &BIG_SIZES, &mut self.editing.big_size);
            choice_list(ui, "Wheel Opacity", &OPACITIES, &mut self.editing.opacity);
            ui.checkbox(
                "Invert vertical axis for Look",
                &mut self.editing.invert_look,
            );
            ui.checkbox("Zoom to the pointer", &mut self.editing.zoom_to_pointer);
            ui.checkbox(
                "Keep scene upright while orbiting",
                &mut self.editing.keep_up_while_orbiting,
            );
            ui.checkbox(
                "Constrain Walk to the ground plane",
                &mut self.editing.walk_on_plane,
            );
            ui.slider("Walk Speed Factor", 0.1, 10., &mut self.editing.walk_speed);
            ui.separator();
            close = self.options_buttons(ui)
                || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape);
        });
        self.options_visible.set_open(open && !close);
    }
    fn options_buttons(&mut self, ui: &Ui) -> bool {
        let mut close = false;
        if crate::widget_abi::button(ui, "Restore Defaults", [0.; 2]) {
            self.editing = WheelOptions::default();
        }
        ui.same_line();
        {
            let _muted = ui.begin_disabled(self.editing == self.options);
            if crate::widget_abi::button(ui, "OK##wheel-options", [80., 0.]) {
                self.options = self.editing;
                close = true;
            }
        }
        ui.same_line();
        close || crate::widget_abi::button(ui, "Cancel##wheel-options", [80., 0.])
    }
}
fn wheel_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_NAV
}
fn muted_row(ui: &Ui, label: &str) {
    let _muted = ui.begin_disabled(true);
    ui.menu_item(label);
}
fn choice_list(ui: &Ui, label: &str, choices: &[&str], chosen: &mut usize) {
    let Some(_list) = ui.begin_combo(label, choices[*chosen]) else {
        return;
    };
    for (index, row) in choices.iter().enumerate() {
        if ui.selectable_config(row).selected(index == *chosen).build() {
            *chosen = index;
        }
    }
}

#[cfg(test)]
#[path = "navigation_wheel_tests.rs"]
mod tests;
