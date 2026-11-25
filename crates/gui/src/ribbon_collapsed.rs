// SPDX-License-Identifier: GPL-2.0-only
//! Keep compact groups identifiable by their caption and emblem.
use super::*;

pub(super) fn minimum_width(tab: RibbonTab) -> f32 {
    match tab {
        RibbonTab::Sketch => 64.,
        RibbonTab::Model => 92.,
        _ => 108.,
    }
}

pub(super) fn collapsed_button(
    context: &PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    identity: &str,
    width: f32,
) -> bool {
    let ui = context.ui;
    let icons = context.owners.icons;
    if !matches!(tab, RibbonTab::Model | RibbonTab::Sketch) {
        let height = (icons.appearance.command_height() - 2.).max(18.) * icons.layout_factor();
        return crate::widget_abi::button(ui, &format!("v{identity}"), [width, height]);
    }
    let height = if tab == RibbonTab::Sketch {
        52.
    } else {
        icons.appearance.command_height() - 2.
    };
    let size = [width, icons.cell([0., height])[1]];
    let glyph = compact_group_icon(panel);
    let clicked = crate::widget_abi::icon_button(
        ui,
        &format!("{}{identity}", panel.label()),
        crate::widget_abi::IconButtonPresentation {
            texture: icons.key(glyph, crate::icons::IconSize::Large),
            size,
            icon: icons.cell([24.; 2]),
            large: true,
            show_caption: true,
        },
    );
    let [right, top] = [ui.item_rect_max()[0] - 5., ui.item_rect_min()[1] + 8.];
    ui.get_window_draw_list()
        .add_triangle(
            [right - 6., top],
            [right, top],
            [right - 3., top + 4.],
            ui.style_color(imgui::StyleColor::Text),
        )
        .filled(true)
        .build();
    clicked
}
fn compact_group_icon(panel: RibbonPanel) -> crate::icons::IconGlyph {
    use crate::icons::IconGlyph;
    match panel {
        RibbonPanel::Create => IconGlyph::CreateSketch,
        RibbonPanel::Constrain => IconGlyph::Coincident,
        RibbonPanel::Modify => IconGlyph::Move,
        RibbonPanel::SketchPattern => IconGlyph::RectangularPattern,
        RibbonPanel::Insert => IconGlyph::InsertImage,
        RibbonPanel::Format => IconGlyph::Construction,
        RibbonPanel::Parameters => IconGlyph::Parameters,
        RibbonPanel::Sketch => IconGlyph::CreateSketch,
        RibbonPanel::PartCreate => IconGlyph::Extrude,
        RibbonPanel::PartModify => IconGlyph::Fillet,
        RibbonPanel::PartExplore => IconGlyph::Measure,
        RibbonPanel::PartWorkFeatures => IconGlyph::WorkPlane,
        RibbonPanel::PartPattern => IconGlyph::RectangularPattern,
        RibbonPanel::PartSimplification => IconGlyph::Simplify,
        _ => IconGlyph::Customize,
    }
}
