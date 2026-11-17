// SPDX-License-Identifier: GPL-2.0-only
//! Shared construction for dockable tool and preference windows.
use imgui::{Ui, Window};

/// A tool window with a title-bar close control and the standard docking behavior.
/// Example: `Self::dockable_window(ui, "Tool", &mut open).build(|| {})`.
pub(crate) trait DockableWindow {
    /// Fit a floating window while leaving docked placement to its node.
    /// Example: `Self::fit_dockable(ui, "Tool", [20., 20.], [400., 300.])`.
    fn fit_dockable(ui: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
        crate::widget_abi::fit_dialog(ui, title, origin, size);
    }
    /// Start centered and retain later user placement; example: `Self::center_dockable(ui, "Tool", size)`.
    fn center_dockable(ui: &Ui, title: &str, size: [f32; 2]) {
        let display = ui.io().display_size;
        let origin = std::array::from_fn(|axis| (display[axis] - size[axis]) * 0.5);
        Self::fit_dockable(ui, title, origin, size);
    }
    /// Notify the docking system when window content requests closure (#4072).
    /// Example: `Self::close_dockable(ui, "Tool", requested)`.
    fn close_dockable(ui: &Ui, title: &str, requested: bool) {
        if !requested {
            return;
        }
        crate::widget_abi::close_dock_tab(ui, title);
    }
    fn dockable_window<'ui, 'open, Label: AsRef<str>>(
        ui: &'ui Ui,
        title: Label,
        open: &'open mut bool,
    ) -> Window<'ui, 'open, Label>
    where
        'ui: 'open,
    {
        ui.window(title).opened(open).collapsible(false)
    }
}
