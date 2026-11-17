// SPDX-License-Identifier: GPL-2.0-only
//! Presentation-owned lifecycle for tool windows whose content comes from the host.
use crate::dockable_window::DockableWindow;

pub(crate) struct ExternalDockWindow {
    title: &'static str,
    state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for ExternalDockWindow {}
impl ExternalDockWindow {
    fn new(title: &'static str) -> Self {
        Self {
            title,
            state: Default::default(),
        }
    }
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(self.title, &self.state)
    }
    pub(crate) fn is_open(&self) -> bool {
        self.state.is_open()
    }
    pub(crate) fn synchronize(&self, open: bool) {
        self.state.set_open(open);
    }
}

pub(crate) struct ExternalDockWindows {
    pub face: ExternalDockWindow,
    pub measure: ExternalDockWindow,
    pub region: ExternalDockWindow,
    pub sketch_move: ExternalDockWindow,
}
impl Default for ExternalDockWindows {
    fn default() -> Self {
        Self {
            face: ExternalDockWindow::new("Face Properties"),
            measure: ExternalDockWindow::new("Measure"),
            region: ExternalDockWindow::new("Region Properties"),
            sketch_move: ExternalDockWindow::new("Move"),
        }
    }
}
impl ExternalDockWindows {
    pub(crate) fn synchronize(&self, state: &crate::GuiCommandState) {
        self.face.synchronize(state.face_properties.is_some());
        self.measure.synchronize(state.measure.is_some());
        self.region.synchronize(state.region_properties.is_some());
        self.sketch_move.synchronize(state.sketch_move.is_some());
    }
    pub(crate) fn observe_actions(&self, actions: &[crate::GuiShellAction]) {
        for action in actions {
            match action {
                crate::GuiShellAction::CloseFaceProperties => self.face.synchronize(false),
                crate::GuiShellAction::Inspect(crate::GuiInspectAction::FinishMeasure) => {
                    self.measure.synchronize(false)
                }
                crate::GuiShellAction::Inspect(crate::GuiInspectAction::FinishRegionProperties) => {
                    self.region.synchronize(false)
                }
                crate::GuiShellAction::SketchMove(crate::GuiSketchMoveAction::Done) => {
                    self.sketch_move.synchronize(false)
                }
                _ => {}
            }
        }
    }
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.face.register_dockable(registry)?;
        self.measure.register_dockable(registry)?;
        self.region.register_dockable(registry)?;
        self.sketch_move.register_dockable(registry)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{GuiInspectAction, GuiShellAction, GuiSketchMoveAction};

    #[test]
    fn host_close_actions_only_retire_their_own_windows() {
        let docks = ExternalDockWindows::default();
        docks.face.synchronize(true);
        docks.measure.synchronize(true);
        docks.region.synchronize(true);
        docks.sketch_move.synchronize(true);
        docks.observe_actions(&[GuiShellAction::Inspect(GuiInspectAction::FinishMeasure)]);
        assert!(!docks.measure.is_open());
        assert!(docks.face.is_open() && docks.region.is_open() && docks.sketch_move.is_open());
        docks.observe_actions(&[
            GuiShellAction::CloseFaceProperties,
            GuiShellAction::Inspect(GuiInspectAction::FinishRegionProperties),
            GuiShellAction::SketchMove(GuiSketchMoveAction::Done),
        ]);
        assert!(!docks.face.is_open() && !docks.region.is_open() && !docks.sketch_move.is_open());
    }
}
