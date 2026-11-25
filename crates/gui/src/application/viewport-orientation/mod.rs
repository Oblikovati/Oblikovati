// SPDX-License-Identifier: GPL-2.0-only
//! The orientation control: a small cube in the viewport's corner that shows and sets the view.
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, StyleColor, Ui, WindowFlags};

#[path = "axis_triad.rs"]
mod axis_triad;
mod hotspots;
mod pick_arrows;
use hotspots::cube_hit;
use pick_arrows::{arrow_at, is_face_on, near};
#[path = "cube.rs"]
mod cube;
#[path = "drawing.rs"]
mod drawing;
use cube::{ShownFace, shown_faces};
#[path = "navigation_bar.rs"]
mod navigation_bar;
#[path = "navigation_wheel.rs"]
mod navigation_wheel;
use navigation_bar::NavigationBar;
pub(crate) use navigation_bar::ZoomAction;
pub use navigation_bar::{GuiNavigationMode, GuiViewCommand};
mod navigation_capsule;
#[path = "options.rs"]
mod options;
use drawing::{draw_compass, draw_control};
use options::{OrientationOptions, ScreenCorner};

#[derive(Clone, Copy, Debug)]
pub(crate) struct ViewControlVisibility {
    pub cube: bool,
    pub navigation: bool,
}

const CONTROL_WINDOW: &str = "##orientation-control";
const CONTROL_MENU: &str = "##orientation-menu";
const MARGIN: f32 = 8.;
/// The cube's half width, as a share of the control's side; the arrows use the rest.
const CUBE_SHARE: f32 = 64. / 240.;
const LABEL_SHOWS_FROM: f32 = 0.3;
const FACING_VIEWER: f32 = 0.02;
const FACE_ON: f32 = 0.999;
const ARROW_REACH: f32 = 9.;
const LOCK_CAPTION: &str = "Locked";
const LOCK_CAPTION_RISE: f32 = 4.;
/// The compass ring's distance outside the cube, past its arrows.
const COMPASS_REACH: f32 = 14.;

/// How the view looks at the model: unit vectors along the screen and toward the viewer.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GuiViewAxes {
    pub right: [f32; 3],
    pub up: [f32; 3],
    pub toward: [f32; 3],
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiCubeFace {
    Front,
    Back,
    Right,
    Left,
    Top,
    Bottom,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiViewStep {
    Right,
    Left,
    Up,
    Down,
}
/// How the view projects the model; the hybrid keeps a face-on view flat.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiViewProjection {
    #[default]
    Orthographic,
    Perspective,
    PerspectiveWithOrthoFaces,
}
impl GuiViewProjection {
    const ALL: [Self; 3] = [
        Self::Orthographic,
        Self::Perspective,
        Self::PerspectiveWithOrthoFaces,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::Orthographic => "Orthographic",
            Self::Perspective => "Perspective",
            Self::PerspectiveWithOrthoFaces => "Perspective with Ortho Faces",
        }
    }
}
/// What a press on the control asks of the view.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiOrientCommand {
    Face(GuiCubeFace),
    /// Look from the corner of the current face on this side of the screen.
    Corner {
        right: bool,
        up: bool,
    },
    /// Model-space normal of an edge or corner picked from an oblique view.
    Bevel([i8; 3]),
    Step(GuiViewStep),
    Roll {
        clockwise: bool,
    },
}
impl GuiCubeFace {
    const ALL: [Self; 6] = [
        Self::Front,
        Self::Back,
        Self::Right,
        Self::Left,
        Self::Top,
        Self::Bottom,
    ];
    /// The name the face carries; example: `GuiCubeFace::Top.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Front => "Front",
            Self::Back => "Back",
            Self::Right => "Right",
            Self::Left => "Left",
            Self::Top => "Top",
            Self::Bottom => "Bottom",
        }
    }
    /// The model direction that points out of this face.
    pub fn normal(self) -> [f32; 3] {
        match self {
            Self::Front => [0., 0., 1.],
            Self::Back => [0., 0., -1.],
            Self::Right => [1., 0., 0.],
            Self::Left => [-1., 0., 0.],
            Self::Top => [0., 1., 0.],
            Self::Bottom => [0., -1., 0.],
        }
    }
    /// The direction its label reads along.
    fn reading(self) -> [f32; 3] {
        match self {
            Self::Back => [-1., 0., 0.],
            Self::Right => [0., 0., -1.],
            Self::Left => [0., 0., 1.],
            _ => [1., 0., 0.],
        }
    }
    /// The face a model direction points out of most.
    fn nearest(direction: [f32; 3]) -> Self {
        Self::ALL
            .into_iter()
            .max_by(|first, second| {
                dot(first.normal(), direction).total_cmp(&dot(second.normal(), direction))
            })
            .unwrap_or(Self::Front)
    }
    fn corners(self) -> [[f32; 3]; 4] {
        let normal = self.normal();
        let reading = self.reading();
        let upward = cross(normal, reading);
        [(-1., -1.), (1., -1.), (1., 1.), (-1., 1.)].map(|(along, over)| {
            std::array::from_fn(|axis| normal[axis] + reading[axis] * along + upward[axis] * over)
        })
    }
}
fn dot(first: [f32; 3], second: [f32; 3]) -> f32 {
    (0..3).map(|axis| first[axis] * second[axis]).sum()
}
fn cross(a: [f32; 3], b: [f32; 3]) -> [f32; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}
/// Where each of the control's parts sits inside its window, by the window's side.
/// Example: `GuiOrientationLayout::for_side(110.).home` for the Home action's centre.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GuiOrientationLayout {
    pub center: [f32; 2],
    pub half: f32,
    pub home: [f32; 2],
    pub menu: [f32; 2],
    pub step_right: [f32; 2],
    pub step_left: [f32; 2],
    pub step_up: [f32; 2],
    pub step_down: [f32; 2],
    pub roll_upper: [f32; 2],
    pub roll_lower: [f32; 2],
}
impl GuiOrientationLayout {
    pub fn for_side(side: f32) -> Self {
        let middle = side / 2.;
        let half = side * CUBE_SHARE;
        let beyond = half + ARROW_REACH + 2.;
        Self {
            center: [middle, middle],
            half,
            home: [ARROW_REACH, ARROW_REACH],
            menu: [side - ARROW_REACH, side - ARROW_REACH],
            step_right: [middle + beyond, middle],
            step_left: [middle - beyond, middle],
            step_up: [middle, middle - beyond],
            step_down: [middle, middle + beyond],
            roll_upper: [side - ARROW_REACH, ARROW_REACH],
            roll_lower: [side - ARROW_REACH, ARROW_REACH * 3.2],
        }
    }
}
#[derive(Default)]
pub(crate) struct OrientationControl {
    options: OrientationOptions,
    projection: GuiViewProjection,
    navigation: NavigationBar,
    chrome_alpha: f32,
    fading: bool,
}
impl OrientationControl {
    pub(crate) fn cube_bounds(&self, area: WorkspaceArea) -> [f32; 4] {
        let side = self.options.side();
        let [left, top] = self.position(area, side);
        [left, top, left + side, top + side]
    }
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.options.register_dockable(registry)?;
        self.navigation.register_dockable(registry)
    }
    pub(crate) fn set_suspended(&mut self, suspended: bool) {
        if suspended {
            self.chrome_alpha = 0.;
            self.fading = false;
        }
        self.options.set_suspended(suspended);
        self.navigation.set_suspended(suspended);
    }
    /// Draw over a part's view; example: each frame, once the document area is known.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        icons: &crate::icons::GuiRibbonIcons,
        request: &crate::GuiFrameRequest,
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
        visibility: ViewControlVisibility,
    ) {
        let state = &request.commands;
        let Some(axes) = state.view_axes else {
            self.set_suspended(true);
            return;
        };
        let model_workspace = matches!(
            state.authoring,
            crate::GuiAuthoringContext::Part | crate::GuiAuthoringContext::Sketch
        ) || state.document_type
            == Some(oblikovati_api::types::DOCUMENT_ASSEMBLY);
        if !model_workspace {
            self.set_suspended(true);
            return;
        }
        self.set_suspended(false);
        axis_triad::draw(ui, area, &axes);
        let side = self.options.side();
        let position = self.position(area, side);
        if visibility.cube {
            let window = ui
                .window(CONTROL_WINDOW)
                .position(position, imgui::Condition::Always)
                .size([side, side], imgui::Condition::Always)
                .flags(control_flags());
            window.build(|| {
                ui.invisible_button("##orientation-area", [side, side]);
                let layout = GuiOrientationLayout::for_side(side);
                let hovered = ui.is_item_hovered();
                let step = ui.io().delta_time / 0.08;
                self.chrome_alpha = if hovered {
                    (self.chrome_alpha + step).min(1.)
                } else {
                    (self.chrome_alpha - step).max(0.)
                };
                self.fading = self.chrome_alpha != if hovered { 1. } else { 0. };
                let idle = self.options.inactive_alpha();
                let alpha = idle + (1. - idle) * self.chrome_alpha;
                let point = ui.io().mouse_pos;
                let local = [point[0] - position[0], point[1] - position[1]];
                let faces = shown_faces(&axes, &layout, position);
                let hit = hovered
                    .then(|| cube_hit(point, &axes, &layout, &faces))
                    .flatten();
                let text = ui.style_color(StyleColor::Text);
                let ink = state
                    .color_scheme
                    .as_ref()
                    .map_or(text, |scheme| scheme.contrasting_ink(text));
                let overlay_text = ui.push_style_color(StyleColor::Text, ink);
                draw_control(
                    ui,
                    &axes,
                    &layout,
                    position,
                    &faces,
                    hit.as_ref(),
                    [alpha, self.chrome_alpha],
                );
                if self.options.compass_shown() {
                    draw_compass(ui, &layout, position, alpha, self.options.north_angle());
                }
                if ui.is_item_clicked_with_button(MouseButton::Left) {
                    if near(local, layout.menu) {
                        ui.open_popup(CONTROL_MENU);
                    } else {
                        press(
                            local,
                            &axes,
                            &layout,
                            hit.map(|hit| hit.command),
                            &self.options,
                            actions,
                        );
                    }
                }
                if ui.is_item_clicked_with_button(MouseButton::Right) {
                    ui.open_popup(CONTROL_MENU);
                }
                if state.orientation_selection_locked {
                    let [x, y] = layout.step_down;
                    ui.set_cursor_pos([x - layout.half, y - LOCK_CAPTION_RISE]);
                    ui.text(LOCK_CAPTION);
                }
                drop(overlay_text);
                self.context_menu(ui, state, actions);
            });
        } else {
            self.chrome_alpha = 0.;
            self.fading = false;
        }
        if visibility.navigation {
            self.navigation_bar(ui, icons, request, area, [position, [side, side]], actions);
        }
        self.navigation.follow_view(ui, request, actions);
        let center = [(area.left + area.right) / 2., (area.top + area.bottom) / 2.];
        let reach = (area.right - area.left).min(area.bottom - area.top);
        self.navigation.draw_guide(ui, center, reach);
        self.navigation.rotate_dialog(ui, state, actions);
        self.navigation.wheel_panel(ui, state, actions);
        self.options.panel(ui);
    }
    fn position(&self, area: WorkspaceArea, side: f32) -> [f32; 2] {
        let corner = self.options.corner();
        let x = if corner.is_left() {
            area.left + MARGIN
        } else {
            area.right - side - MARGIN
        };
        let y = if corner == ScreenCorner::BottomLeft || corner == ScreenCorner::BottomRight {
            area.bottom - side - MARGIN
        } else {
            area.top + MARGIN
        };
        [x, y]
    }
    /// Available with something to lock to; once locked, it stays to let the lock go.
    fn lock_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let locked = state.orientation_selection_locked;
        let row = ui
            .menu_item_config("Lock to Current Selection")
            .selected(locked)
            .enabled(state.orientation_selection_available || locked)
            .build();
        if row {
            actions.push(GuiShellAction::SetOrientationSelectionLock(!locked));
        }
    }
    fn context_menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = ui.begin_popup(CONTROL_MENU) else {
            return;
        };
        if ui.menu_item("Home") {
            actions.push(GuiShellAction::Home);
        }
        for mode in GuiViewProjection::ALL {
            let row = ui
                .menu_item_config(mode.label())
                .selected(self.projection == mode)
                .build();
            if row {
                self.projection = mode;
                actions.push(GuiShellAction::SetViewProjection(mode));
            }
        }
        Self::lock_row(ui, state, actions);
        ui.menu("Set Current View as", || {
            for row in ["Top", "Front"] {
                muted_row(ui, row);
            }
        });
        ui.menu("Set Current View as Home", || {
            for row in ["Fixed Distance", "Fit to View"] {
                muted_row(ui, row);
            }
        });
        if ui.menu_item("Options") {
            self.options.request();
        }
        // Dear ImGui lets one Escape take back one level; here it leaves the whole menu.
        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            ui.close_current_popup();
        }
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn navigation_owns_viewport(&self) -> bool {
        self.navigation.owns_viewport_input()
    }
    pub(crate) fn can_repeat_wheel(&self) -> bool {
        self.navigation.can_repeat_wheel()
    }
    pub(crate) fn repeat_wheel(&mut self) {
        self.navigation.repeat_wheel();
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.fading || self.options.needs_frame() || self.navigation.needs_frame()
    }
    /// A navigation command is running, so a click on the view belongs to it.
    pub(crate) fn owns_view_clicks(&self) -> bool {
        self.navigation.is_running()
    }
}
/// Listed as observed; no view definition can be stored in this build.
fn muted_row(ui: &Ui, label: &str) {
    let _muted = ui.begin_disabled(true);
    ui.menu_item(label);
}
fn control_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_NAV
        | WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS
}
fn press(
    local: [f32; 2],
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    picked: Option<GuiOrientCommand>,
    options: &OrientationOptions,
    actions: &mut Vec<GuiShellAction>,
) {
    if near(local, layout.home) {
        actions.push(GuiShellAction::Home);
        return;
    }
    let command = arrow_at(local, axes, layout).or(picked);
    if let Some(command) = command {
        actions.push(GuiShellAction::OrientView {
            command,
            fit: options.fits_on_change(),
            animate: options.animates_transitions(),
        });
    }
}
#[cfg(test)]
#[path = "tests.rs"]
mod tests;
