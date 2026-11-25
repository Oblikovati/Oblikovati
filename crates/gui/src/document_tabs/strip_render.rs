// SPDX-License-Identifier: GPL-2.0-only
//! Rendering and interaction for document strip tabs.
use super::*;
use crate::icons::{IconGlyph, IconSize};

const TAB_ICON_SIZE: f32 = 14.;

/// Home first, then one width per window, as they will be drawn.
pub(super) fn tab_widths(ui: &Ui, tabs: &[StripTab<'_>]) -> Vec<f32> {
    let spacing = ui.clone_style().item_inner_spacing[0];
    tabs.iter()
        .map(|tab| match tab {
            StripTab::Home => ui.calc_text_size("Home")[0] + HOME_TAB_ROOM,
            StripTab::Document(tab) => ui.calc_text_size(tab_label(tab))[0] + TAB_ROOM,
        })
        .map(|width| width + spacing)
        .collect()
}
pub(super) fn active_tab_index(
    state: &crate::GuiCommandState,
    tabs: &[StripTab<'_>],
    home_active: bool,
) -> Option<usize> {
    tabs.iter().position(|tab| match tab {
        StripTab::Home => home_active,
        StripTab::Document(tab) => !home_active && is_active_window(state, tab),
    })
}
/// Home is a workspace, not a document: it has no close control and is chosen only by a click.
pub(super) fn home_tab(
    ui: &Ui,
    owner: &mut StripOwner<'_>,
    state: &crate::GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    use crate::application::document_tabs::previews::{HoveredTab, PreviewKey};
    let flags = if owner.home.active {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    ui.set_next_item_width(ui.calc_text_size("Home")[0] + HOME_TAB_ROOM);
    let shown = ui.tab_item_with_flags("###home-workspace-tab", None, flags);
    draw_tab_label(ui, owner.icons, IconGlyph::Home, "Home");
    if owner.home.active {
        crate::theme::active_tab_underline(ui);
    }
    owner.order.note(
        TabKey::Home,
        ui.item_rect_min()[0],
        ui.is_item_active() && ui.is_mouse_clicked(imgui::MouseButton::Left),
    );
    let through_popup = owner.home_press.released(ui, ());
    let clicked = ui.is_item_clicked() || through_popup;
    if ui.is_item_hovered() {
        owner.hovered = Some(HoveredTab {
            key: PreviewKey::Home,
            bounds: [ui.item_rect_min(), ui.item_rect_max()],
            labels: vec!["Home".to_owned()],
        });
    }
    drop(shown);
    owner.menu.home_popup(ui, state, actions);
    owner.home.active |= clicked;
}
pub(super) fn document_strip_window(
    ui: &Ui,
    span: (f32, f32),
    strip_top: f32,
    contents: impl FnOnce(),
) {
    let _minimum = ui.push_style_var(imgui::StyleVar::WindowMinSize([1., 1.]));
    let window_padding = ui.push_style_var(imgui::StyleVar::WindowPadding([6., 0.]));
    let _tab_padding = ui.push_style_var(imgui::StyleVar::FramePadding([6., 5.]));
    crate::widget_abi::window_geometry(
        ui,
        [span.0, strip_top],
        [span.1, TAB_STRIP_HEIGHT],
        Condition::Always,
    );
    let mut flags = imgui::WindowFlags::NO_DOCKING;
    if crate::widget_abi::moving_window(ui) {
        flags |= imgui::WindowFlags::NO_INPUTS;
    }
    let strip = ui
        .window("##open-document-tabs")
        .flags(flags)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .scroll_bar(false)
        .scrollable(false)
        .collapsible(false)
        .begin();
    drop(window_padding);
    if let Some(_strip) = strip {
        contents();
    }
}
/// A tab is chosen by its own click: the strip may scroll, or Home may stand in front, without
/// the owner's document changing.
pub(super) fn document_tab(
    ui: &Ui,
    request: &GuiFrameRequest,
    tab: &TabRow<'_>,
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let state = &request.commands;
    let behind = owner.home.active
        || state
            .document
            .is_some_and(|id| owner.detached.contains(&id));
    let selected = !behind && is_active_window(state, tab);
    let keep = draw_document_tab(ui, owner.icons, tab, selected);
    if selected {
        crate::theme::active_tab_underline(ui);
    }
    #[cfg(test)]
    record_test_tab_bounds(ui, tab.row.id, owner.test_tab_bounds);
    owner.order.note(
        TabKey::Window(tab.window),
        ui.item_rect_min()[0],
        ui.is_item_active() && ui.is_mouse_clicked(imgui::MouseButton::Left),
    );
    let clicked = ui.is_item_clicked();
    note_hovered_tab(ui, request, tab, owner);
    owner.menu.popup(ui, state, tab.row, actions);
    if ui.is_item_active() {
        *owner.pressed = Some(tab.row.id);
    }
    if !keep {
        actions.push(GuiShellAction::CloseDocument(CloseDocumentArgs {
            id: tab.row.id,
            ..Default::default()
        }));
        return;
    }
    if clicked {
        owner.home.reached_from_home = owner.home.active;
        owner.home.active = false;
        actions.extend(activation(state, tab).filter(|action| action.dispatchable(state)));
    }
}
fn note_hovered_tab(
    ui: &Ui,
    request: &GuiFrameRequest,
    tab: &TabRow<'_>,
    owner: &mut StripOwner<'_>,
) {
    use crate::application::document_tabs::previews::{HoveredTab, PreviewKey};
    if !ui.is_item_hovered() {
        return;
    }
    let saved = request
        .commands
        .document_storage
        .get(&tab.row.id)
        .and_then(|storage| storage.path.clone());
    owner.hovered = Some(HoveredTab {
        key: PreviewKey::Window(tab.window),
        bounds: [ui.item_rect_min(), ui.item_rect_max()],
        labels: saved.into_iter().chain([tab.name.clone()]).collect(),
    });
}
fn tab_label(tab: &TabRow<'_>) -> String {
    format!("{}{}", tab.name, if tab.row.dirty { " *" } else { "" })
}
fn draw_document_tab(
    ui: &Ui,
    icons: &crate::icons::GuiRibbonIcons,
    tab: &TabRow<'_>,
    selected: bool,
) -> bool {
    let flags = if selected {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    let label = tab_label(tab);
    ui.set_next_item_width(ui.calc_text_size(&label)[0] + TAB_ROOM);
    let mut keep = true;
    let contents = ui.tab_item_with_flags(
        format!("###open-document-{}", tab.window),
        Some(&mut keep),
        flags,
    );
    let glyph = match tab.row.type_.as_str() {
        "assembly" => IconGlyph::NewAssembly,
        "drawing" => IconGlyph::NewDrawing,
        "presentation" => IconGlyph::TemplateDocument,
        _ => IconGlyph::NewPart,
    };
    draw_tab_label(ui, icons, glyph, &label);
    drop(contents);
    keep
}
fn draw_tab_label(ui: &Ui, icons: &crate::icons::GuiRibbonIcons, glyph: IconGlyph, label: &str) {
    let [left, top] = ui.item_rect_min();
    let bottom = ui.item_rect_max()[1].min(crate::widget_abi::window_draw_clip_bottom(ui)) - 2.;
    let center = (top + bottom) / 2.;
    let texture = imgui::TextureId::new(icons.key(glyph, IconSize::Small).raw());
    ui.get_window_draw_list()
        .add_image(
            texture,
            [left + 8., center - TAB_ICON_SIZE / 2.],
            [left + 8. + TAB_ICON_SIZE, center + TAB_ICON_SIZE / 2.],
        )
        .build();
    crate::widget_abi::literal_text(
        ui,
        [
            left + 8. + TAB_ICON_SIZE + 5.,
            (center - ui.current_font_size() / 2.).floor(),
        ],
        ui.style_color(imgui::StyleColor::Text),
        label,
    );
}

#[cfg(test)]
fn record_test_tab_bounds(ui: &Ui, id: u64, bounds: &mut TestTabBounds) {
    let min = ui.item_rect_min();
    let max = ui.item_rect_max();
    // ImGui places the close mark inside the tab's right padding.
    let close = [
        max[0] - ui.clone_style().frame_padding[0] * 2. - ui.current_font_size() / 2.,
        (min[1] + max[1]) / 2.,
    ];
    bounds.insert(id, ([min, max], close));
}
