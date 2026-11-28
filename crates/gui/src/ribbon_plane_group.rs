// SPDX-License-Identifier: GPL-2.0-only
//! Work Plane's offset action and reference constructor group (#4105, #3918).
use crate::{
    GuiCommandState, GuiShellAction, GuiWorkPlaneAction, GuiWorkPlaneMode,
    icons::{GuiRibbonIcons, IconGlyph, IconSize},
    widget_abi::{self, IconButtonPresentation},
};
use imgui::{StyleColor, Ui};

const REFERENCE_VARIANTS: [(&str, GuiWorkPlaneMode); 5] = [
    (
        "Midplane between Two Planes",
        GuiWorkPlaneMode::MidplaneTwoPlanes,
    ),
    ("Three Points", GuiWorkPlaneMode::ThreePoints),
    (
        "Parallel to Plane through Point",
        GuiWorkPlaneMode::ParallelThroughPoint,
    ),
    ("Through Axis and Point", GuiWorkPlaneMode::LineThroughPoint),
    ("Two Coplanar Edges", GuiWorkPlaneMode::TwoCoplanarEdges),
];
const ORIENTATION_VARIANTS: [(&str, GuiWorkPlaneMode); 6] = [
    (
        "Angle to Plane around Edge",
        GuiWorkPlaneMode::AngleToPlaneAroundEdge,
    ),
    (
        "Tangent to Surface and Parallel to Plane",
        GuiWorkPlaneMode::TangentParallelToPlane,
    ),
    (
        "Tangent to Surface through Point",
        GuiWorkPlaneMode::TangentThroughPoint,
    ),
    (
        "Tangent to Surface through Edge",
        GuiWorkPlaneMode::TangentThroughEdge,
    ),
    (
        "Normal to Axis through Point",
        GuiWorkPlaneMode::NormalToAxisThroughPoint,
    ),
    ("Midplane of Torus", GuiWorkPlaneMode::MidplaneOfTorus),
];
pub(super) const COMMAND_COUNT: usize = REFERENCE_VARIANTS.len() + ORIENTATION_VARIANTS.len() + 2;

pub(super) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    ungrouped: bool,
    expanded: bool,
) {
    let origin = ui.cursor_pos();
    let (_, size) = icons.appearance.cell(0);
    if ungrouped {
        draw_offset_button(ui, state, icons, actions, "Plane", size);
        let (position, size) = variant_cell(icons, 1, expanded);
        icons.cursor(ui, origin, position);
        draw_offset_button(ui, state, icons, actions, "Offset from Plane", size);
        draw_individual_variants(ui, state, icons, actions, origin, expanded);
    } else {
        let host = [size[0] + trigger_column_extra(icons), size[1]];
        draw_offset_button(
            ui,
            state,
            icons,
            actions,
            "Plane",
            crate::ribbon_split::primary_size(host),
        );
        if crate::ribbon_split::trigger(ui, icons, "##plane-variants", host[1]) {
            ui.open_popup("##plane-variant-menu");
        }
        draw_variants_popup(ui, state, icons, actions);
    }
}

pub(super) fn trigger_column_extra(icons: &GuiRibbonIcons) -> f32 {
    use crate::ribbon_command_layout::RibbonAppearance;
    if icons.appearance == RibbonAppearance::Normal {
        0.
    } else {
        crate::ribbon_split::TRIGGER_SPACE
    }
}

fn draw_offset_button(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    label: &str,
    size: [f32; 2],
) {
    crate::chrome::command_control(
        ui,
        label,
        GuiShellAction::OpenOffsetPlane,
        state,
        actions,
        || icons.button(ui, label, IconGlyph::WorkPlane, size),
    );
}

fn draw_variants_popup(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_popup) = ui.begin_popup("##plane-variant-menu") else {
        return;
    };
    let _button_color =
        ui.push_style_color(StyleColor::Button, ui.style_color(StyleColor::PopupBg));
    draw_popup_action(
        ui,
        state,
        icons,
        "Plane",
        GuiShellAction::WorkPlane(GuiWorkPlaneAction::Start(GuiWorkPlaneMode::General)),
        actions,
    );
    draw_popup_action(
        ui,
        state,
        icons,
        "Offset from Plane",
        GuiShellAction::OpenOffsetPlane,
        actions,
    );
    for group in [
        REFERENCE_VARIANTS.as_slice(),
        ORIENTATION_VARIANTS.as_slice(),
    ] {
        ui.separator();
        for &(name, mode) in group {
            draw_popup_action(
                ui,
                state,
                icons,
                name,
                GuiShellAction::WorkPlane(GuiWorkPlaneAction::Start(mode)),
                actions,
            );
        }
    }
}

fn draw_popup_action(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    name: &str,
    action: GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if popup_choice(ui, icons, name) {
        actions.push(action);
        ui.close_current_popup();
    }
    drop(disabled);
}

fn popup_choice(ui: &Ui, icons: &GuiRibbonIcons, name: &str) -> bool {
    widget_abi::icon_button(
        ui,
        name,
        IconButtonPresentation {
            texture: icons.key(IconGlyph::WorkPlane, IconSize::Small),
            size: icons.cell([285., 22.]),
            icon: icons.logical[IconSize::Small.index()],
            large: false,
            show_caption: true,
        },
    )
}

fn draw_individual_variants(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    origin: [f32; 2],
    expanded: bool,
) {
    for (index, (name, mode)) in REFERENCE_VARIANTS
        .into_iter()
        .chain(ORIENTATION_VARIANTS)
        .enumerate()
    {
        let (position, size) = variant_cell(icons, index + 2, expanded);
        icons.cursor(ui, origin, position);
        crate::chrome::command_control(
            ui,
            name,
            GuiShellAction::WorkPlane(GuiWorkPlaneAction::Start(mode)),
            state,
            actions,
            || icons.button(ui, name, IconGlyph::WorkPlane, size),
        );
    }
}

fn variant_cell(icons: &GuiRibbonIcons, index: usize, expanded: bool) -> ([f32; 2], [f32; 2]) {
    if expanded {
        ([0., index as f32 * 30.], [170., 26.])
    } else {
        icons.appearance.cell(index)
    }
}
