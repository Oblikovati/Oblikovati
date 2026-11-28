// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiShellAction, GuiThemeState,
    dockable_window::DockableWindow,
    theme_editor_rows::{THEME_COLOR_ROWS, ThemeColorRow},
};
use imgui::{StyleColor, StyleVar, Ui};
#[derive(Default)]
pub(crate) struct GuiThemeEditor {
    pub(crate) new_name: String,
    requested: bool,
    open: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for GuiThemeEditor {}
impl GuiThemeEditor {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(THEME_EDITOR_POPUP, &self.open)
    }
}
const THEME_EDITOR_POPUP: &str = "Theme editor";

pub(crate) fn launcher(
    ui: &Ui,
    editor: &mut GuiThemeEditor,
    icons: &crate::icons::GuiRibbonIcons,
) -> bool {
    ui.same_line();
    launcher_icon(ui, editor, icons, [108., 20.])
}
pub(crate) fn launcher_icon(
    ui: &Ui,
    editor: &mut GuiThemeEditor,
    icons: &crate::icons::GuiRibbonIcons,
    size: [f32; 2],
) -> bool {
    let clicked = icons.button(ui, "Edit Theme", crate::icons::IconGlyph::Preferences, size);
    editor.requested |= clicked;
    clicked
}
pub(crate) fn panel(
    ui: &Ui,
    theme: Option<&GuiThemeState>,
    editor: &mut GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) {
    // #3849: the launcher belongs to a flyout; draw the editor from its stable owner.
    if std::mem::take(&mut editor.requested) {
        editor.open.set_open(true);
    }
    let Some(theme) = theme.filter(|_| editor.open.is_open()) else {
        editor.open.set_open(false);
        return;
    };
    popup_geometry(ui);
    let _border = crate::theme::property_window_border(ui);
    let mut open = true;
    let mut close = false;
    GuiThemeEditor::dockable_window(ui, THEME_EDITOR_POPUP, &mut open).build(|| {
        crate::theme::floating_property_shadow(ui);
        {
            let _title = crate::ui_scale::dialog_title_font(ui);
            ui.text(format!("Theme: {}", theme.active_name()));
        }
        close = editor_actions(ui, theme, actions)
            || ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        duplicate_controls(ui, theme, editor, actions);
        ui.text_colored(
            crate::theme::text_secondary(ui),
            "Edits preview immediately. Save keeps them after restart.",
        );
        if !theme.editable() {
            ui.text_colored(crate::theme::text_muted(ui), "Built-in theme is read-only.");
        }
        color_rows(ui, theme, actions);
    });
    editor.open.set_open(open && !close);
}
fn popup_geometry(ui: &Ui) {
    let [width, height] = ui.io().display_size;
    let top_clear = 148.;
    let right_clear = if width > 720. { 136. } else { 16. };
    let size = [
        (width - right_clear - 16.).clamp(1., 560.),
        (height - top_clear - 24.).clamp(1., 600.),
    ];
    let x = ((width - right_clear - size[0]) * 0.5).max(16.);
    let y = (top_clear + ((height - top_clear - size[1]) * 0.5).max(0.)).min(height - size[1]);
    GuiThemeEditor::fit_dockable(ui, THEME_EDITOR_POPUP, [x, y], size);
}
fn editor_actions(ui: &Ui, theme: &GuiThemeState, actions: &mut Vec<GuiShellAction>) -> bool {
    let disabled = ui.begin_disabled(!theme.save_available());
    if crate::theme::primary_button(ui, "Save theme", [100., 26.]) {
        actions.push(GuiShellAction::SaveTheme {
            name: theme.active_name().into(),
        });
    }
    drop(disabled);
    ui.same_line_with_spacing(0., 6.);
    let close = crate::theme::secondary_button(ui, "Close", [70., 26.]);
    ui.same_line_with_spacing(0., 6.);
    let _disabled = ui.begin_disabled(!theme.deletion_available());
    if crate::theme::secondary_button(ui, "Delete", [70., 26.]) {
        actions.push(GuiShellAction::DeleteTheme {
            name: theme.active_name().into(),
        });
    }
    close
}
fn duplicate_controls(
    ui: &Ui,
    theme: &GuiThemeState,
    editor: &mut GuiThemeEditor,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.align_text_to_frame_padding();
    ui.text("New name");
    ui.same_line();
    ui.set_next_item_width((ui.content_region_avail()[0] - 108.).clamp(1., 230.));
    ui.input_text("##theme-name", &mut editor.new_name).build();
    ui.same_line_with_spacing(0., 6.);
    let _disabled = ui.begin_disabled(!theme.creation_available());
    if crate::theme::secondary_button(ui, "Duplicate", [100., 24.]) {
        actions.push(GuiShellAction::DuplicateTheme {
            base: theme.active_name().into(),
            name: editor.new_name.clone(),
        });
    }
}
fn color_rows(ui: &Ui, theme: &GuiThemeState, actions: &mut Vec<GuiShellAction>) {
    let alpha = ui.clone_style().alpha;
    let light_read_only = !theme.editable() && ui.style_color(StyleColor::WindowBg)[0] > 0.5;
    let _disabled = ui.begin_disabled(!theme.editable());
    let _readable = light_read_only.then(|| ui.push_style_var(StyleVar::Alpha(alpha)));
    let _ink = light_read_only
        .then(|| ui.push_style_color(StyleColor::Text, crate::theme::hex_rgb(0x4B5263)));
    let _outline = ui.push_style_var(StyleVar::FrameBorderSize(1.));
    let mut group = "";
    for row in THEME_COLOR_ROWS {
        if row.group != group {
            ui.separator();
            {
                let _semi = crate::ui_scale::ui_semibold_font(ui);
                ui.text(row.group);
            }
            group = row.group;
        }
        color_row(ui, theme, row, actions);
    }
}
fn color_row(
    ui: &Ui,
    theme: &GuiThemeState,
    row: &ThemeColorRow,
    actions: &mut Vec<GuiShellAction>,
) {
    let value = &theme.palette()[&row.token];
    let mut channels = [value.r, value.g, value.b, value.a];
    let label = format!("{}##{}", row.label, row.token.0);
    if ui.color_edit4(label, &mut channels) {
        let [r, g, b, a] = channels;
        actions.push(GuiShellAction::EditThemeColor {
            name: theme.active_name().into(),
            token: row.token.clone(),
            value: oblikovati_api::types::Rgba { r, g, b, a },
        });
    }
}
