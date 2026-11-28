// SPDX-License-Identifier: GPL-2.0-only
//! Go chrome_ribbon.go button styles through the existing scalar widget ABI.
use super::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::{GuiCommandState, GuiShellAction, GuiSketchConstraint, GuiSketchTool};
use imgui::Ui;

pub(crate) const FLYOUT_ROW_HEIGHT: f32 = 28.;

struct TransparentCommandChrome<'a> {
    _bg: Option<imgui::ColorStackToken<'a>>,
    _border: Option<imgui::StyleStackToken<'a>>,
}

fn transparent_command_chrome(ui: &Ui) -> TransparentCommandChrome<'_> {
    let active = ui.style_color(imgui::StyleColor::Button)
        == ui.style_color(imgui::StyleColor::ButtonActive);
    if active {
        return TransparentCommandChrome {
            _bg: None,
            _border: None,
        };
    }
    TransparentCommandChrome {
        _bg: Some(ui.push_style_color(imgui::StyleColor::Button, [0.; 4])),
        _border: Some(ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.))),
    }
}

impl GuiRibbonIcons {
    pub(crate) fn flyout_button(&self, ui: &Ui, label: &str, glyph: IconGlyph) -> bool {
        let _chrome = transparent_command_chrome(ui);
        crate::widget_abi::icon_button(
            ui,
            label,
            crate::widget_abi::IconButtonPresentation {
                texture: self.key(glyph, IconSize::Small),
                size: [ui.content_region_avail()[0].max(1.), FLYOUT_ROW_HEIGHT],
                icon: [16.; 2],
                large: false,
                show_caption: true,
            },
        )
    }

    pub(crate) fn brand_mark(&self, ui: &Ui, beside_toolbar: bool) {
        let texture = imgui::TextureId::new(self.key(IconGlyph::BrandMark, IconSize::Small).raw());
        let [start_x, start_y] = ui.cursor_pos();
        if beside_toolbar {
            crate::widget_abi::cursor_position(ui, [start_x, start_y + 4.]);
        }
        imgui::Image::new(texture, [16.; 2]).build(ui);
        if !beside_toolbar {
            ui.same_line();
            return;
        }
        let [right, _] = ui.item_rect_max();
        let top = ui.item_rect_min()[1];
        let x = (right + 5.).floor() + 0.5;
        ui.get_window_draw_list()
            .add_line(
                [x, top + 1.],
                [x, top + 15.],
                crate::theme::border_subtle(ui),
            )
            .build();
        ui.same_line_with_spacing(0., 0.);
        ui.dummy([10., 16.]);
        ui.same_line_with_spacing(0., 0.);
        crate::widget_abi::cursor_position(ui, [ui.cursor_pos()[0], start_y]);
    }

    pub(crate) fn brand_wordmark(&self, ui: &Ui, available: f32) {
        let width = available.min(216.);
        let texture =
            imgui::TextureId::new(self.key(IconGlyph::BrandWordmark, IconSize::Large).raw());
        imgui::Image::new(texture, [width, width * 40. / 216.]).build(ui);
    }

    pub(crate) fn navigation_button(
        &self,
        ui: &Ui,
        label: &str,
        glyph: IconGlyph,
        size: [f32; 2],
    ) -> bool {
        let _chrome = transparent_command_chrome(ui);
        let pressed = crate::widget_abi::icon_button(
            ui,
            label,
            crate::widget_abi::IconButtonPresentation {
                texture: self.key(glyph, IconSize::Small),
                size,
                icon: [16.; 2],
                large: false,
                show_caption: false,
            },
        );
        if ui.is_item_hovered() {
            ui.tooltip_text(label);
        }
        pressed
    }
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
        let _chrome = transparent_command_chrome(ui);
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
        let _chrome = transparent_command_chrome(ui);
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
            // Glyphs can overhang their measured size, including descenders (#4062).
            ui.tooltip(|| {
                ui.text(format!("{label} "));
                ui.dummy([0., 2.]);
            });
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
        let show_caption = self.appearance == RibbonAppearance::Normal;
        let mut cell = self.cell(size);
        if show_caption {
            cell[0] = cell[0].max(self.caption_width(ui, label, style));
        }
        crate::widget_abi::IconButtonPresentation {
            texture: self.key(glyph, style),
            size: cell,
            icon,
            large,
            show_caption,
        }
    }

    pub(crate) fn normal_button_width(&self, ui: &Ui, label: &str, size: [f32; 2]) -> f32 {
        self.cell(size)[0].max(self.caption_width(ui, label, IconSize::for_height(size[1])))
    }

    fn caption_width(&self, ui: &Ui, label: &str, style: IconSize) -> f32 {
        let caption = ui.calc_text_size_with_opts(label, true, -1.)[0];
        let inset = if style == IconSize::Large {
            10.
        } else {
            self.logical[style.index()][0] + 18.
        };
        caption + inset
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
pub(crate) fn action_glyph(action: &GuiShellAction) -> Option<IconGlyph> {
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
        GuiShellAction::UpdatePart => IconGlyph::Update,
        GuiShellAction::OpenIProperties => IconGlyph::Preferences,
        GuiShellAction::Undo => IconGlyph::Undo,
        GuiShellAction::Redo => IconGlyph::Redo,
        GuiShellAction::OpenShellFeature => IconGlyph::Shell,
        GuiShellAction::Revolve(crate::GuiRevolveAction::Open) => IconGlyph::Revolve,
        GuiShellAction::Sweep(crate::GuiSweepAction::Open) => IconGlyph::Sweep,
        GuiShellAction::Loft(crate::GuiLoftAction::Open) => IconGlyph::Loft,
        GuiShellAction::Coil(crate::GuiCoilAction::Open) => IconGlyph::Coil,
        GuiShellAction::Emboss(crate::GuiEmbossAction::Open) => IconGlyph::Emboss,
        GuiShellAction::Rib(crate::GuiRibAction::Open) => IconGlyph::Rib,
        GuiShellAction::Decal(crate::GuiDecalAction::Open) => IconGlyph::Decal,
        GuiShellAction::Import(crate::GuiImportAction::Open) => IconGlyph::Import,
        GuiShellAction::Unwrap(crate::GuiUnwrapAction::Open) => IconGlyph::Unwrap,
        GuiShellAction::Derive(crate::GuiDeriveAction::Open) => IconGlyph::Derive,
        _ => return None,
    })
}
fn tool_glyph(tool: GuiSketchTool) -> IconGlyph {
    match tool {
        GuiSketchTool::Line => IconGlyph::Line,
        GuiSketchTool::MidpointLine => IconGlyph::MidpointLine,
        GuiSketchTool::Rectangle => IconGlyph::Rectangle,
        GuiSketchTool::CenterRectangle => IconGlyph::CenterRectangle,
        GuiSketchTool::ThreePointRectangle => IconGlyph::ThreePointRectangle,
        GuiSketchTool::ThreePointCenterRectangle => IconGlyph::ThreePointCenterRectangle,
        GuiSketchTool::Circle | GuiSketchTool::ThreePointCircle => IconGlyph::Circle,
        GuiSketchTool::CenterPointArc => IconGlyph::CenterPointArc,
        GuiSketchTool::ThreePointArc => IconGlyph::Arc,
        GuiSketchTool::Point => IconGlyph::Point,
        GuiSketchTool::Ellipse => IconGlyph::Ellipse,
        GuiSketchTool::Fillet => IconGlyph::SketchFillet,
        GuiSketchTool::Text => IconGlyph::SketchText,
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
