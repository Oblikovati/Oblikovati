// SPDX-License-Identifier: GPL-2.0-only
//! Shared construction for ordinary, dockable, and tabbed ImGui windows.
use imgui::{Ui, Window as ImGuiWindow, WindowFlags};

/// A standard non-dockable application window.
/// Example: `Self::window(ui, "Preferences", &mut open).build(|| {})`.
pub(crate) trait Window {
    /// Construct a standard window that cannot be docked into another window.
    fn window<'ui, 'open, Label: AsRef<str>>(
        ui: &'ui Ui,
        title: Label,
        open: &'open mut bool,
    ) -> ImGuiWindow<'ui, 'open, Label>
    where
        'ui: 'open,
    {
        ui.window(title)
            .opened(open)
            .collapsible(false)
            .flags(WindowFlags::NO_DOCKING)
    }

    /// Fit a floating window while leaving later placement to the user.
    /// Example: `Self::fit_window(ui, "Preferences", [20., 20.], [400., 300.])`.
    fn fit_window(ui: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
        crate::widget_abi::fit_dialog(ui, title, origin, size);
    }

    /// Position a new window at the center of the current display.
    /// Example: `Self::center_window(ui, "Preferences", [400., 300.])`.
    fn center_window(ui: &Ui, title: &str, size: [f32; 2]) {
        let display = ui.io().display_size;
        let origin = std::array::from_fn(|axis| (display[axis] - size[axis]) * 0.5);
        Self::fit_window(ui, title, origin, size);
    }
}

impl<T> Window for T {}

/// A standard application window that can be docked and has a title-bar close control.
/// Example: `Self::dockable_window(ui, "Tool", &mut open).build(|| {})`.
pub(crate) trait DockableWindow: Window {
    fn dockable_window<'ui, 'open, Label: AsRef<str>>(
        ui: &'ui Ui,
        title: Label,
        open: &'open mut bool,
    ) -> ImGuiWindow<'ui, 'open, Label>
    where
        'ui: 'open,
    {
        ui.window(title).opened(open).collapsible(false)
    }

    /// Compatibility helper retained while callers migrate to the base `Window` API.
    fn fit_dockable(ui: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
        Self::fit_window(ui, title, origin, size);
    }

    /// Compatibility helper retained while callers migrate to the base `Window` API.
    fn center_dockable(ui: &Ui, title: &str, size: [f32; 2]) {
        Self::center_window(ui, title, size);
    }
}

/// A dockable window whose tab selection and reordering belong to ImGui's dock node.
/// Implementors provide one dockable window per tab and place them in the same node.
pub(crate) trait TabbedDockableWindow: DockableWindow {
    /// Construct a window intended to participate in an ImGui dock tab group.
    /// Example: `Self::tabbed_dockable_window(ui, "Model Browser", "Model", &mut open).build(|| {})`.
    fn tabbed_dockable_window<'ui, 'open, Label: AsRef<str>>(
        ui: &'ui Ui,
        title: Label,
        group: &str,
        open: &'open mut bool,
    ) -> ImGuiWindow<'ui, 'open, Label>
    where
        'ui: 'open,
    {
        crate::widget_abi::ensure_tabbed_window(ui, title.as_ref(), group);
        Self::dockable_window(ui, title, open)
    }
}

impl<T: DockableWindow> TabbedDockableWindow for T {}
