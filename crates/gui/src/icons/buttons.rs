// SPDX-License-Identifier: GPL-2.0-only
//! Go chrome_ribbon.go button styles through the existing scalar widget ABI.
use super::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::{GuiCommandState, GuiShellAction, GuiSketchConstraint, GuiSketchTool};
use imgui::Ui;

impl GuiRibbonIcons {
    pub(crate) fn layout_factor(&self) -> f32 {
        self.logical
            .iter()
            .zip([18., 40.])
            .flat_map(|(axes, base)| axes.iter().map(move |edge| edge / base))
            .fold(1., f32::max)
    }
    pub(crate) fn cell(&self, authored: [f32; 2]) -> [f32; 2] {
        authored.map(|length| length * self.layout_factor())
    }
    pub(crate) fn cursor(&self, ui: &Ui, origin: [f32; 2], offset: [f32; 2]) {
        let offset = self.cell(offset);
        crate::widget_abi::cursor_position(
            ui,
            std::array::from_fn(|axis| origin[axis] + offset[axis]),
        );
    }
    pub(crate) fn button(&self, ui: &Ui, label: &str, glyph: IconGlyph, size: [f32; 2]) -> bool {
        let presentation = self.presentation(ui, label, glyph, size);
        let show_caption = presentation.show_caption;
        let pressed = crate::widget_abi::icon_button(ui, label, presentation);
        if !show_caption && ui.is_item_hovered() {
            ui.tooltip_text(label);
        }
        pressed
    }
    /// Draw an icon-only application-bar command with its accessible tooltip.
    /// Example: `icons.compact_button(ui, "Open", IconGlyph::Open)`.
    pub(crate) fn compact_button(&self, ui: &Ui, label: &str, glyph: IconGlyph) -> bool {
        let icon = self.logical[IconSize::Small.index()];
        let pressed = crate::widget_abi::icon_button(
            ui,
            label,
            crate::widget_abi::IconButtonPresentation {
                texture: self.key(glyph, IconSize::Small),
                size: self.cell([24., 24.]),
                icon,
                large: false,
                show_caption: false,
            },
        );
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            // Glyphs can overhang their advance; the extra space prevents clipping (#4062).
            ui.tooltip_text(format!("{label} "));
        }
        pressed
    }
    fn presentation(
        &self,
        ui: &Ui,
        label: &str,
        glyph: IconGlyph,
        size: [f32; 2],
    ) -> crate::widget_abi::IconButtonPresentation {
        use crate::ribbon_command_layout::RibbonAppearance;
        let style = match self.appearance {
            RibbonAppearance::Normal => IconSize::for_height(size[1]),
            RibbonAppearance::TextOff | RibbonAppearance::Large => IconSize::Large,
            _ => IconSize::Small,
        };
        let large = style == IconSize::Large;
        let icon = self.logical[style.index()];
        let mut cell = self.cell(size);
        let show_caption = self.appearance == RibbonAppearance::Normal;
        if show_caption {
            let caption = ui.calc_text_size(label)[0];
            cell[0] = cell[0].max(caption + if large { 4. } else { icon[0] + 7. });
        }
        crate::widget_abi::IconButtonPresentation {
            texture: self.key(glyph, style),
            size: cell,
            icon,
            large,
            show_caption,
        }
    }
    pub(crate) fn command_button(
        &self,
        ui: &Ui,
        label: &str,
        action: GuiShellAction,
        state: &GuiCommandState,
        size: [f32; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        crate::chrome::command_control(ui, label, action.clone(), state, actions, || {
            match action_glyph(&action) {
                Some(glyph) => self.button(ui, label, glyph, size),
                None => crate::widget_abi::button(ui, label, self.cell(size)),
            }
        });
    }
}
fn action_glyph(action: &GuiShellAction) -> Option<IconGlyph> {
    Some(match action {
        GuiShellAction::NewPart => IconGlyph::NewPart,
        GuiShellAction::NewDocumentWindow => IconGlyph::NewWindow,
        GuiShellAction::DeleteSketchSelection => IconGlyph::Delete,
        GuiShellAction::SketchMove(crate::GuiSketchMoveAction::Start) => IconGlyph::Move,
        GuiShellAction::ChooseSketchPlane => IconGlyph::CreateSketch,
        GuiShellAction::FinishSketch => IconGlyph::FinishSketch,
        GuiShellAction::PlaceSketchTool(tool) => tool_glyph(*tool),
        GuiShellAction::ConstrainSketch(constraint) => constraint_glyph(*constraint),
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start) => IconGlyph::Dimension,
        GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure) => IconGlyph::Measure,
        GuiShellAction::Home => IconGlyph::Home,
        GuiShellAction::Save => IconGlyph::Save,
        GuiShellAction::Undo => IconGlyph::Undo,
        GuiShellAction::Redo => IconGlyph::Redo,
        _ => return None,
    })
}
fn tool_glyph(tool: GuiSketchTool) -> IconGlyph {
    match tool {
        GuiSketchTool::Line => IconGlyph::Line,
        GuiSketchTool::Rectangle
        | GuiSketchTool::CenterRectangle
        | GuiSketchTool::ThreePointRectangle
        | GuiSketchTool::ThreePointCenterRectangle => IconGlyph::Rectangle,
        GuiSketchTool::Circle | GuiSketchTool::ThreePointCircle => IconGlyph::Circle,
        GuiSketchTool::CenterPointArc | GuiSketchTool::ThreePointArc => IconGlyph::Arc,
        GuiSketchTool::Point => IconGlyph::Point,
        GuiSketchTool::Ellipse => IconGlyph::Ellipse,
    }
}
fn constraint_glyph(constraint: GuiSketchConstraint) -> IconGlyph {
    match constraint {
        GuiSketchConstraint::Horizontal => IconGlyph::Horizontal,
        GuiSketchConstraint::Vertical => IconGlyph::Vertical,
        GuiSketchConstraint::Coincident => IconGlyph::Coincident,
        GuiSketchConstraint::Fix => IconGlyph::Fix,
        GuiSketchConstraint::Parallel => IconGlyph::Parallel,
        GuiSketchConstraint::Perpendicular => IconGlyph::Perpendicular,
        GuiSketchConstraint::Collinear => IconGlyph::Collinear,
        GuiSketchConstraint::Equal => IconGlyph::Equal,
        GuiSketchConstraint::Concentric => IconGlyph::Concentric,
        GuiSketchConstraint::Tangent => IconGlyph::Tangent,
    }
}
