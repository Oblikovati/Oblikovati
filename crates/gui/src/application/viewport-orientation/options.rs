// SPDX-License-Identifier: GPL-2.0-only
//! The control's options: where it sits and how big it is. Cancel forgets what was changed.
use crate::dockable_window::DockableWindow;
use imgui::Ui;

const TITLE: &str = "Orientation Options";
const SIZE: [f32; 2] = [440., 560.];
const OPACITY_STEPS: [u8; 5] = [0, 25, 50, 75, 100];

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(super) enum ScreenCorner {
    #[default]
    TopRight,
    BottomRight,
    TopLeft,
    BottomLeft,
}
impl ScreenCorner {
    const ALL: [Self; 4] = [
        Self::TopRight,
        Self::BottomRight,
        Self::TopLeft,
        Self::BottomLeft,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::TopRight => "Top Right",
            Self::BottomRight => "Bottom Right",
            Self::TopLeft => "Top Left",
            Self::BottomLeft => "Bottom Left",
        }
    }
    pub(super) fn is_left(self) -> bool {
        matches!(self, Self::TopLeft | Self::BottomLeft)
    }
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum ControlSize {
    #[default]
    Automatic,
    Tiny,
    Small,
    Normal,
    Large,
}
impl ControlSize {
    const ALL: [Self; 5] = [
        Self::Automatic,
        Self::Tiny,
        Self::Small,
        Self::Normal,
        Self::Large,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::Automatic => "Automatic",
            Self::Tiny => "Tiny",
            Self::Small => "Small",
            Self::Normal => "Normal",
            Self::Large => "Large",
        }
    }
    fn side(self) -> f32 {
        match self {
            Self::Tiny => 80.,
            Self::Small => 100.,
            Self::Automatic | Self::Normal => 120.,
            Self::Large => 170.,
        }
    }
}
/// A model plane and which way it faces, as a named view can look at it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum ViewPlane {
    XyPlus,
    XyMinus,
    XzPlus,
    XzMinus,
    YzPlus,
    YzMinus,
}
impl ViewPlane {
    const ALL: [Self; 6] = [
        Self::XyPlus,
        Self::XyMinus,
        Self::XzPlus,
        Self::XzMinus,
        Self::YzPlus,
        Self::YzMinus,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::XyPlus => "XY(+Z)",
            Self::XyMinus => "XY(-Z)",
            Self::XzPlus => "XZ(+Y)",
            Self::XzMinus => "XZ(-Y)",
            Self::YzPlus => "YZ(+X)",
            Self::YzMinus => "YZ(-X)",
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq)]
struct OptionValues {
    show_on_new_window: bool,
    all_views: bool,
    front_plane: ViewPlane,
    top_plane: ViewPlane,
    corner: ScreenCorner,
    size: ControlSize,
    inactive_opacity: u8,
    snap_to_closest: bool,
    fit_on_change: bool,
    animate_transitions: bool,
    keep_upright: bool,
    compass: bool,
    north_angle: f32,
}
impl Default for OptionValues {
    fn default() -> Self {
        Self {
            show_on_new_window: true,
            all_views: false,
            front_plane: ViewPlane::XyPlus,
            top_plane: ViewPlane::XzPlus,
            corner: ScreenCorner::default(),
            size: ControlSize::default(),
            inactive_opacity: 50,
            snap_to_closest: true,
            fit_on_change: false,
            animate_transitions: false,
            keep_upright: false,
            compass: false,
            north_angle: 0.,
        }
    }
}
#[derive(Default)]
pub(super) struct OrientationOptions {
    applied: OptionValues,
    editing: OptionValues,
    opening: bool,
    open: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for OrientationOptions {}
impl OrientationOptions {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.open)
    }
    pub(super) fn set_suspended(&self, suspended: bool) {
        self.open.set_suspended(suspended);
    }
    pub(super) fn request(&mut self) {
        self.opening = true;
        self.editing = self.applied;
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.opening
    }
    pub(super) fn corner(&self) -> ScreenCorner {
        self.applied.corner
    }
    pub(super) fn side(&self) -> f32 {
        self.applied.size.side()
    }
    /// How much of the control shows while the pointer is elsewhere, from 0 to 1.
    pub(super) fn inactive_alpha(&self) -> f32 {
        f32::from(self.applied.inactive_opacity) / 100.
    }
    pub(super) fn fits_on_change(&self) -> bool {
        self.applied.fit_on_change
    }
    pub(super) fn north_angle(&self) -> f32 {
        self.applied.north_angle
    }
    pub(super) fn compass_shown(&self) -> bool {
        self.applied.compass
    }
    pub(super) fn panel(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.opening) {
            self.open.set_open(true);
        }
        if !self.open.is_open() {
            return;
        }
        Self::center_dockable(ui, TITLE, SIZE);
        let mut open = true;
        let mut close = false;
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            self.visibility_rows(ui);
            self.placement_rows(ui);
            self.behavior_rows(ui);
            self.view_plane_rows(ui);
            self.compass_rows(ui);
            ui.separator();
            if crate::widget_abi::button(ui, "OK", [80., 0.]) {
                self.applied = self.editing;
                close = true;
            }
            ui.same_line();
            close |= crate::widget_abi::button(ui, "Cancel", [80., 0.])
                || ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        });
        self.open.set_open(open && !close);
    }
    fn visibility_rows(&mut self, ui: &Ui) {
        ui.checkbox(
            "Show on window creation",
            &mut self.editing.show_on_new_window,
        );
        let _off = ui.begin_disabled(!self.editing.show_on_new_window);
        if ui.radio_button_bool("All 3D Views", self.editing.all_views) {
            self.editing.all_views = true;
        }
        if ui.radio_button_bool("Only in Current View", !self.editing.all_views) {
            self.editing.all_views = false;
        }
    }
    /// Two names for one plane cannot stand: picking the other's plane swaps them.
    fn view_plane_rows(&mut self, ui: &Ui) {
        let (front, top) = (self.editing.front_plane, self.editing.top_plane);
        choice_list(
            ui,
            "Front View Plane",
            &ViewPlane::ALL,
            ViewPlane::label,
            &mut self.editing.front_plane,
        );
        if self.editing.front_plane == top {
            self.editing.top_plane = front;
        }
        let (front, top) = (self.editing.front_plane, self.editing.top_plane);
        choice_list(
            ui,
            "Top View Plane",
            &ViewPlane::ALL,
            ViewPlane::label,
            &mut self.editing.top_plane,
        );
        if self.editing.top_plane == front {
            self.editing.front_plane = top;
        }
    }
    fn placement_rows(&mut self, ui: &Ui) {
        choice_list(
            ui,
            "On-screen Position",
            &ScreenCorner::ALL,
            ScreenCorner::label,
            &mut self.editing.corner,
        );
        choice_list(
            ui,
            "Size",
            &ControlSize::ALL,
            ControlSize::label,
            &mut self.editing.size,
        );
        self.opacity_list(ui);
    }
    fn opacity_list(&mut self, ui: &Ui) {
        let shown = format!("{}%", self.editing.inactive_opacity);
        let Some(_list) = ui.begin_combo("Inactive Opacity", shown) else {
            return;
        };
        for percent in OPACITY_STEPS {
            let row = ui
                .selectable_config(format!("{percent}%"))
                .selected(percent == self.editing.inactive_opacity)
                .build();
            if row {
                self.editing.inactive_opacity = percent;
            }
        }
    }
    fn behavior_rows(&mut self, ui: &Ui) {
        ui.checkbox("Snap to closest view", &mut self.editing.snap_to_closest);
        ui.checkbox("Fit to view on change", &mut self.editing.fit_on_change);
        ui.checkbox(
            "Use animated transitions",
            &mut self.editing.animate_transitions,
        );
        ui.checkbox("Keep model upright", &mut self.editing.keep_upright);
    }
    fn compass_rows(&mut self, ui: &Ui) {
        ui.checkbox("Show the compass", &mut self.editing.compass);
        let _off = ui.begin_disabled(!self.editing.compass);
        ui.input_float("Angle of North", &mut self.editing.north_angle)
            .build();
    }
}
fn choice_list<T: Copy + PartialEq>(
    ui: &Ui,
    label: &str,
    all: &[T],
    name: fn(T) -> &'static str,
    chosen: &mut T,
) {
    let Some(_list) = ui.begin_combo(label, name(*chosen)) else {
        return;
    };
    for item in all {
        if ui
            .selectable_config(name(*item))
            .selected(item == chosen)
            .build()
        {
            *chosen = *item;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn cancelled_changes_are_forgotten_and_ok_keeps_them() {
        let mut options = OrientationOptions::default();
        options.request();
        options.editing.compass = true;
        options.request();
        assert!(!options.editing.compass);
        options.editing.compass = true;
        options.applied = options.editing;
        assert!(options.compass_shown());
        assert_eq!(options.inactive_alpha(), 0.5);
    }
}
