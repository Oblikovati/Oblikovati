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
    let cap = if tab == RibbonTab::Sketch { 52. } else { 58. };
    let height = (icons.appearance.command_height() - 2.).clamp(18., cap);
    let stacked = height >= 42.;
    let size = [width, icons.cell([0., height])[1]];
    let glyph = compact_group_icon(panel);
    let clicked = crate::widget_abi::icon_button(
        ui,
        &format!("{}{identity}", panel.label()),
        crate::widget_abi::IconButtonPresentation {
            texture: icons.key(glyph, crate::icons::IconSize::Large),
            size,
            icon: icons.cell([if stacked { 24. } else { 14. }; 2]),
            large: stacked,
            show_caption: true,
        },
    );
    draw_disclosure(ui, panel.label(), stacked);
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
        RibbonPanel::ManageInsert => IconGlyph::InsertModel,
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
        RibbonPanel::AnnotateDimension | RibbonPanel::PartToleranceAnalysis => IconGlyph::Dimension,
        RibbonPanel::AnnotateText | RibbonPanel::PartNotes | RibbonPanel::PartGeneralAnnotation => {
            IconGlyph::Annotation
        }
        RibbonPanel::AnnotateSymbols | RibbonPanel::PartGeometricAnnotation => {
            IconGlyph::DrawingSurface
        }
        RibbonPanel::PartCreate => IconGlyph::Extrude,
        RibbonPanel::PartPrimitives => IconGlyph::CreateBlock,
        RibbonPanel::PartModify => IconGlyph::Fillet,
        RibbonPanel::PartWorkFeatures => IconGlyph::WorkPlane,
        RibbonPanel::PartSimplification | RibbonPanel::PartShapeGenerator => IconGlyph::Simplify,
        RibbonPanel::PartFreeform => IconGlyph::Envelope,
        RibbonPanel::UserCommands | RibbonPanel::PartAnnotationManage => IconGlyph::Customize,
    }
}
