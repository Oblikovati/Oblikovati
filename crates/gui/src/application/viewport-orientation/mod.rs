// SPDX-License-Identifier: GPL-2.0-only
//! The orientation control: a small cube in the viewport's corner that shows and sets the view.
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, StyleColor, Ui, WindowFlags};

#[path = "axis_triad.rs"]
mod axis_triad;
#[path = "cube.rs"]
mod cube;
#[path = "drawing.rs"]
mod drawing;
use cube::{ShownFace, inside, shown_faces};
#[path = "navigation_bar.rs"]
mod navigation_bar;
#[path = "navigation_wheel.rs"]
mod navigation_wheel;
pub use navigation_bar::{GuiNavigationMode, GuiViewCommand};
use navigation_bar::{NAVIGATION_SIZE, NAVIGATION_WINDOW, NavigationBar};
#[path = "options.rs"]
mod options;
use drawing::{draw_compass, draw_control};
use options::{OrientationOptions, ScreenCorner};

const CONTROL_WINDOW: &str = "##orientation-control";
const CONTROL_MENU: &str = "##orientation-menu";
const MARGIN: f32 = 8.;
/// The cube's half width, as a share of the control's side; the arrows use the rest.
const CUBE_SHARE: f32 = 0.3;
/// How much of a face-on square's half width, from its edge, picks an edge or a corner.
const EDGE_BAND: f32 = 0.6;
const FACE_FILL: [f32; 4] = [0.72, 0.75, 0.8, 0.92];
const FACE_EDGE: [f32; 4] = [0.2, 0.22, 0.26, 1.];
const LABEL_COLOR: [f32; 4] = [0.08, 0.09, 0.11, 1.];
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
            step_right: [middle + beyond, middle],
            step_left: [middle - beyond, middle],
            step_up: [middle, middle - beyond],
            step_down: [middle, middle + beyond],
            roll_upper: [side - ARROW_REACH, ARROW_REACH],
            roll_lower: [side - ARROW_REACH, ARROW_REACH * 3.2],
        }
    }
}
/// What a press at this point of the cube picks. Face-on, its rim picks an edge or a corner.
fn cube_pick(
    point: [f32; 2],
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
) -> Option<GuiOrientCommand> {
    let faces = shown_faces(axes, layout, origin);
    let front = faces.first()?;
    if front.facing < FACE_ON {
        let hit = faces.iter().find(|shown| inside(point, &shown.corners))?;
        return Some(GuiOrientCommand::Face(hit.face));
    }
    let across = (point[0] - origin[0] - layout.center[0]) / layout.half;
    let down = (point[1] - origin[1] - layout.center[1]) / layout.half;
    if across.abs() > 1. || down.abs() > 1. {
        return None;
    }
    Some(rim_pick(across, down, axes, front.face))
}
fn rim_pick(across: f32, down: f32, axes: &GuiViewAxes, face: GuiCubeFace) -> GuiOrientCommand {
    match (across.abs() > EDGE_BAND, down.abs() > EDGE_BAND) {
        (true, true) => GuiOrientCommand::Corner {
            right: across > 0.,
            up: down < 0.,
        },
        (false, false) => GuiOrientCommand::Face(face),
        // An edge picks the face beyond it, as the recorded top edge did.
        (true, false) => {
            let side = if across > 0. { 1. } else { -1. };
            GuiOrientCommand::Face(GuiCubeFace::nearest(axes.right.map(|value| value * side)))
        }
        (false, true) => {
            let side = if down < 0. { 1. } else { -1. };
            GuiOrientCommand::Face(GuiCubeFace::nearest(axes.up.map(|value| value * side)))
        }
    }
}
#[derive(Default)]
pub(crate) struct OrientationControl {
    options: OrientationOptions,
    projection: GuiViewProjection,
    navigation: NavigationBar,
}
impl OrientationControl {
    /// Draw over a part's view; example: each frame, once the document area is known.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        request: &crate::GuiFrameRequest,
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let state = &request.commands;
        let Some(axes) = state.view_axes else {
            return;
        };
        if !matches!(
            state.authoring,
            crate::GuiAuthoringContext::Part | crate::GuiAuthoringContext::Sketch
        ) {
            return;
        }
        axis_triad::draw(ui, area, &axes);
        let side = self.options.side();
        let position = self.position(area, side);
        let window = ui
            .window(CONTROL_WINDOW)
            .position(position, imgui::Condition::Always)
            .size([side, side], imgui::Condition::Always)
            .flags(control_flags());
        window.build(|| {
            ui.invisible_button("##orientation-area", [side, side]);
            let layout = GuiOrientationLayout::for_side(side);
            let alpha = if ui.is_item_hovered() {
                1.
            } else {
                self.options.inactive_alpha()
            };
            draw_control(ui, &axes, &layout, position, alpha);
            if self.options.compass_shown() {
                draw_compass(ui, &layout, position, alpha, self.options.north_angle());
            }
            if ui.is_item_clicked_with_button(MouseButton::Left) {
                let fit = self.options.fits_on_change();
                press(ui, &axes, &layout, position, fit, actions);
            }
            if ui.is_item_clicked_with_button(MouseButton::Right) {
                ui.open_popup(CONTROL_MENU);
            }
            if state.orientation_selection_locked {
                let [x, y] = layout.step_down;
                ui.set_cursor_pos([x - layout.half, y - LOCK_CAPTION_RISE]);
                ui.text(LOCK_CAPTION);
            }
            self.context_menu(ui, state, actions);
        });
        self.navigation_bar(
            ui,
            state,
            request.extent,
            area,
            [position, [side, side]],
            actions,
        );
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
    /// The bar keeps to the control's side of the view, under or over it.
    fn navigation_bar(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        extent: [u32; 2],
        area: WorkspaceArea,
        [control, control_size]: [[f32; 2]; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let corner = self.options.corner();
        let x = if corner.is_left() {
            area.left + MARGIN
        } else {
            area.right - NAVIGATION_SIZE[0] - MARGIN
        };
        let below = corner == ScreenCorner::TopLeft || corner == ScreenCorner::TopRight;
        let y = if below {
            control[1] + control_size[1] + MARGIN
        } else {
            control[1] - NAVIGATION_SIZE[1] - MARGIN
        };
        ui.window(NAVIGATION_WINDOW)
            .position([x, y], imgui::Condition::Always)
            .size(NAVIGATION_SIZE, imgui::Condition::Always)
            .flags(control_flags())
            .build(|| self.navigation.buttons(ui, state, extent, actions));
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
        self.options.needs_frame() || self.navigation.needs_frame()
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
    ui: &Ui,
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
    fit: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    let point = ui.io().mouse_pos;
    let local = [point[0] - origin[0], point[1] - origin[1]];
    if near(local, layout.home) {
        actions.push(GuiShellAction::Home);
        return;
    }
    let command = arrow_at(local, axes, layout).or_else(|| cube_pick(point, axes, layout, origin));
    if let Some(command) = command {
        actions.push(GuiShellAction::OrientView { command, fit });
    }
}
fn near(point: [f32; 2], center: [f32; 2]) -> bool {
    (point[0] - center[0]).hypot(point[1] - center[1]) <= ARROW_REACH
}
fn is_face_on(axes: &GuiViewAxes) -> bool {
    GuiCubeFace::ALL
        .iter()
        .any(|face| dot(face.normal(), axes.toward) > FACE_ON)
}
/// The arrows show only face-on: an oblique view has no neighbour to step to.
fn arrow_at(
    point: [f32; 2],
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
) -> Option<GuiOrientCommand> {
    use GuiOrientCommand::{Roll, Step};
    if !is_face_on(axes) {
        return None;
    }
    [
        (layout.step_right, Step(GuiViewStep::Right)),
        (layout.step_left, Step(GuiViewStep::Left)),
        (layout.step_up, Step(GuiViewStep::Up)),
        (layout.step_down, Step(GuiViewStep::Down)),
        (layout.roll_upper, Roll { clockwise: true }),
        (layout.roll_lower, Roll { clockwise: false }),
    ]
    .into_iter()
    .find(|(center, _)| near(point, *center))
    .map(|(_, command)| command)
}
#[cfg(test)]
#[path = "tests.rs"]
mod tests;
