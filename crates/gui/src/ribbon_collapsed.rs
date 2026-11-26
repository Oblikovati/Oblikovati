// SPDX-License-Identifier: GPL-2.0-only
//! Keep compact groups identifiable by their caption and emblem.
use super::*;

pub(super) fn row_widths(
    context: &PanelDrawContext<'_>,
    tab: RibbonTab,
    panels: &[RibbonPanel],
    layout: &RibbonPanelLayout,
    available: f32,
    natural: f32,
) -> (Vec<f32>, usize) {
    let factor = context.owners.icons.layout_factor();
    let normal = |panel| normal_group_width(context, tab, panel, natural);
    let room = compact_room(tab, panels, layout, available, factor);
    let (minimum, count) = compact_row_minimum(context, panels, factor);
    // Keep the scrollable ribbon when the groups would have too little room for captions.
    let caption_floor = count * (minimum + context.ui.calc_text_size("M...")[0]);
    if widths_without_exit(panels, normal).sum::<f32>() <= room || room < caption_floor {
        let expanded = layout.expanded_count(tab, panels, natural, available, factor);
        return (panels.iter().copied().map(normal).collect(), expanded);
    }
    (
        compressed_group_widths(context, panels, room, minimum, count),
        0,
    )
}

fn normal_group_width(
    context: &PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    natural: f32,
) -> f32 {
    if tab == RibbonTab::Manage {
        caption_width(context, tab, panel)
    } else {
        natural
    }
}

fn compact_row_minimum(
    context: &PanelDrawContext<'_>,
    panels: &[RibbonPanel],
    factor: f32,
) -> (f32, f32) {
    let minimum = if context.owners.icons.appearance.command_height() >= 44. {
        28.
    } else {
        30.
    } * factor;
    let count = panels
        .iter()
        .filter(|&&panel| panel != RibbonPanel::Exit)
        .count() as f32;
    (minimum, count)
}

fn compressed_group_widths(
    context: &PanelDrawContext<'_>,
    panels: &[RibbonPanel],
    room: f32,
    minimum: f32,
    count: f32,
) -> Vec<f32> {
    let tight = |panel: RibbonPanel| {
        (context.ui.calc_text_size(panel.label())[0] + caption_padding(context)).max(minimum)
    };
    let floor = count * minimum;
    let total = widths_without_exit(panels, tight).sum::<f32>();
    let share = ((room - floor) / (total - floor).max(1.)).clamp(0., 1.);
    panels
        .iter()
        .copied()
        .map(|panel| minimum + (tight(panel) - minimum) * share)
        .collect()
}

fn compact_room(
    tab: RibbonTab,
    panels: &[RibbonPanel],
    layout: &RibbonPanelLayout,
    available: f32,
    factor: f32,
) -> f32 {
    let exit = if panels.contains(&RibbonPanel::Exit) {
        layout
            .panel_width(tab, RibbonPanel::Exit, factor)
            .unwrap_or(132. * factor)
    } else {
        0.
    };
    available - panels.len().saturating_sub(1) as f32 * 8. - exit - 2.
}

fn widths_without_exit(
    panels: &[RibbonPanel],
    width: impl Fn(RibbonPanel) -> f32,
) -> impl Iterator<Item = f32> {
    panels
        .iter()
        .copied()
        .filter(|&panel| panel != RibbonPanel::Exit)
        .map(width)
}

fn caption_padding(context: &PanelDrawContext<'_>) -> f32 {
    if context.owners.icons.appearance.command_height() >= 44. {
        20.
    } else {
        34.
    }
}

pub(super) fn minimum_width(tab: RibbonTab) -> f32 {
    match tab {
        RibbonTab::Sketch => 64.,
        RibbonTab::Model => 92.,
        _ => 108.,
    }
}

pub(super) fn caption_width(
    context: &PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
) -> f32 {
    let caption_room = caption_padding(context);
    (context.ui.calc_text_size(panel.label())[0] + caption_room).max(minimum_width(tab))
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
    let cap = if tab == RibbonTab::Sketch { 52. } else { 58. };
    let height = (icons.appearance.command_height() - 2.).clamp(18., cap);
    let stacked = height >= 42.;
    let size = [width, icons.cell([0., height])[1]];
    let glyph = compact_group_icon(panel);
    let room = if stacked { 16. } else { 34. };
    let caption = crate::application::home_recent_documents::recent_rows::fit_recent_cell(
        panel.label(),
        width - room,
        |text| ui.calc_text_size(text)[0],
    );
    let clicked = crate::widget_abi::icon_button(
        ui,
        &format!("{caption}#{identity}"),
        crate::widget_abi::IconButtonPresentation {
            texture: icons.key(glyph, crate::icons::IconSize::Large),
            size,
            icon: icons.cell([if stacked { 24. } else { 14. }; 2]),
            large: stacked,
            show_caption: true,
        },
    );
    draw_disclosure(ui, &caption, stacked);
    clicked
}

fn draw_disclosure(ui: &Ui, label: &str, stacked: bool) {
    let [left, top] = ui.item_rect_min();
    let [right, bottom] = ui.item_rect_max();
    let [label_width, label_height] = ui.calc_text_size(label);
    let x = if stacked {
        ((left + right + label_width) * 0.5 + 5.).min(right - 4.)
    } else {
        right - 5.
    };
    let y = if stacked {
        bottom - label_height * 0.5 - 4.
    } else {
        (top + bottom) * 0.5 - 2.
    };
    ui.get_window_draw_list()
        .add_triangle(
            [x - 3., y],
            [x + 3., y],
            [x, y + 4.],
            ui.style_color(imgui::StyleColor::Text),
        )
        .filled(true)
        .build();
}
fn compact_group_icon(panel: RibbonPanel) -> crate::icons::IconGlyph {
    use crate::icons::IconGlyph;
    match panel {
        RibbonPanel::Launch => IconGlyph::NewPart,
        RibbonPanel::Create | RibbonPanel::Sketch | RibbonPanel::DrawingSketch => {
            IconGlyph::CreateSketch
        }
        RibbonPanel::Exit => IconGlyph::FinishSketch,
        RibbonPanel::Constrain => IconGlyph::Coincident,
        RibbonPanel::Layout => IconGlyph::CreateBlock,
        RibbonPanel::Modify | RibbonPanel::PresentationModel => IconGlyph::Move,
        RibbonPanel::SketchPattern | RibbonPanel::PartPattern => IconGlyph::RectangularPattern,
        RibbonPanel::Insert => IconGlyph::InsertImage,
        RibbonPanel::ManageUpdate => IconGlyph::Update,
        RibbonPanel::ManageDocument => IconGlyph::Preferences,
        RibbonPanel::ManageStyles => IconGlyph::AppearanceBrowser,
        RibbonPanel::ManageInsert | RibbonPanel::ManageContentCenter => IconGlyph::InsertModel,
        RibbonPanel::ManagePointCloud => IconGlyph::Point,
        RibbonPanel::ManageLayout => IconGlyph::NewAssembly,
        RibbonPanel::ManageAuthor => IconGlyph::NewPart,
        RibbonPanel::ManageRules => IconGlyph::Parameters,
        RibbonPanel::Format => IconGlyph::Construction,
        RibbonPanel::Parameters => IconGlyph::Parameters,
        RibbonPanel::ToolOptions => IconGlyph::Preferences,
        RibbonPanel::ToolAssets => IconGlyph::Materials,
        RibbonPanel::ToolMeasure | RibbonPanel::PartExplore => IconGlyph::Measure,
        RibbonPanel::ToolFind => IconGlyph::Find,
        RibbonPanel::ToolClipboard => IconGlyph::Copy,
        RibbonPanel::ToolTutorials => IconGlyph::TemplateDocument,
        RibbonPanel::ToolTeamWeb => IconGlyph::TeamWeb,
        RibbonPanel::Navigate => IconGlyph::Home,
        RibbonPanel::Appearance => IconGlyph::AppearanceBrowser,
        RibbonPanel::Visibility => IconGlyph::SketchVisibility,
        RibbonPanel::Windows => IconGlyph::SwitchWindow,
        RibbonPanel::Component => IconGlyph::PlaceComponent,
        RibbonPanel::SheetMetalTools => IconGlyph::SheetFlange,
        RibbonPanel::DrawingViews => IconGlyph::DrawingBase,
        RibbonPanel::DrawingModify => IconGlyph::DrawingBreak,
        RibbonPanel::DrawingSheets => IconGlyph::DrawingSheet,
        RibbonPanel::AnnotateDimension
        | RibbonPanel::PartToleranceAnalysis
        | RibbonPanel::PartSimulation => IconGlyph::Dimension,
        RibbonPanel::AnnotateText | RibbonPanel::PartNotes | RibbonPanel::PartGeneralAnnotation => {
            IconGlyph::Annotation
        }
        RibbonPanel::AnnotateSymbols
        | RibbonPanel::PartGeometricAnnotation
        | RibbonPanel::PartSurface => IconGlyph::DrawingSurface,
        RibbonPanel::PartCreate => IconGlyph::Extrude,
        RibbonPanel::PartPrimitives => IconGlyph::CreateBlock,
        RibbonPanel::PartModify => IconGlyph::Fillet,
        RibbonPanel::PartWorkFeatures => IconGlyph::WorkPlane,
        RibbonPanel::PartSimplification | RibbonPanel::PartShapeGenerator => IconGlyph::Simplify,
        RibbonPanel::PartFreeform => IconGlyph::Envelope,
        RibbonPanel::PartPlastic => IconGlyph::Shell,
        RibbonPanel::PartHarness => IconGlyph::WorkAxis,
        RibbonPanel::PartFactory => IconGlyph::NewAssembly,
        RibbonPanel::PartConvert => IconGlyph::Import,
        RibbonPanel::PartPrint => IconGlyph::Print,
        RibbonPanel::UserCommands | RibbonPanel::PartAnnotationManage => IconGlyph::Customize,
    }
}
