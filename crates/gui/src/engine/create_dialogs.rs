// SPDX-License-Identifier: GPL-2.0-only
//! Bundled dockable dialogs for Part Create features to keep GuiFrameEngine modular.
use crate::{GuiCommandState, GuiShellAction, icons::GuiRibbonIcons};

#[derive(Default)]
pub(super) struct CreateFeatureDialogs {
    pub(super) revolve: crate::revolve_dialog::RevolveDialog,
    pub(super) sweep: crate::sweep_dialog::SweepDialog,
    pub(super) loft: crate::loft_dialog::LoftDialog,
    pub(super) coil: crate::coil_dialog::CoilDialog,
    pub(super) emboss: crate::emboss_dialog::EmbossDialog,
    pub(super) derive: crate::derive_dialog::DeriveDialog,
    pub(super) rib: crate::rib_dialog::RibDialog,
    pub(super) decal: crate::decal_dialog::DecalDialog,
    pub(super) import: crate::import_dialog::ImportDialog,
    pub(super) unwrap: crate::unwrap_dialog::UnwrapDialog,
    pub(super) thread: crate::thread_dialog::ThreadDialog,
    pub(super) mark: crate::mark_dialog::MarkDialog,
    pub(super) finish: crate::finish_dialog::FinishDialog,
}

impl CreateFeatureDialogs {
    pub(super) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.sweep.register_dockable(registry)?;
        self.revolve.register_dockable(registry)?;
        self.coil.register_dockable(registry)?;
        self.emboss.register_dockable(registry)?;
        self.derive.register_dockable(registry)?;
        self.rib.register_dockable(registry)?;
        self.decal.register_dockable(registry)?;
        self.import.register_dockable(registry)?;
        self.unwrap.register_dockable(registry)?;
        self.thread.register_dockable(registry)?;
        self.mark.register_dockable(registry)?;
        self.finish.register_dockable(registry)?;
        self.loft.register_dockable(registry)
    }

    pub(super) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.revolve.synchronize(state, actions);
        self.coil.synchronize(state, actions);
        self.emboss.synchronize(state, actions);
        self.derive.synchronize(state, actions);
        self.rib.synchronize(state, actions);
        self.decal.synchronize(state, actions);
        self.import.synchronize(state, actions);
        self.unwrap.synchronize(state, actions);
        self.thread.synchronize(state, actions);
        self.mark.synchronize(state, actions);
        self.finish.synchronize(state, actions);
        self.sweep.synchronize(state, actions);
        self.loft.synchronize(state, actions);
    }

    pub(super) fn status_prompt<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        self.unwrap
            .status_prompt()
            .or_else(|| state.ucs.status_prompt())
    }

    pub(super) fn take_sketch_plane_request(&mut self) -> bool {
        self.revolve.take_sketch_plane_request()
            || self.coil.take_sketch_plane_request()
            || self.emboss.take_sketch_plane_request()
    }

    pub(super) fn is_opening(&self) -> bool {
        self.derive.is_opening() || self.import.is_opening()
    }

    pub(super) fn draw(
        &mut self,
        ui: &imgui::Ui,
        commands: &GuiCommandState,
        (area, icons, parameters, picked_axis): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
            &mut crate::parameters::GuiParameterEditor,
            Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
        ),
        selected: &mut Vec<GuiShellAction>,
    ) {
        self.revolve
            .panel(ui, commands, (area, icons, parameters), selected);
        self.sweep.panel(ui, commands, (area, icons), selected);
        self.loft.panel(ui, commands, area, selected);
        self.coil
            .panel(ui, commands, (area, icons, picked_axis), selected);
        self.emboss.panel(ui, commands, (area, icons), selected);
        self.derive.panel(ui, commands, (area, icons), selected);
        self.rib.panel(ui, commands, (area, icons), selected);
        self.decal.panel(ui, commands, (area, icons), selected);
        self.import.panel(ui, commands, (area, icons), selected);
        self.unwrap.panel(ui, commands, (area, icons), selected);
        self.thread.panel(ui, commands, (area, icons), selected);
        self.mark.panel(ui, commands, (area, icons), selected);
        self.finish.panel(ui, commands, (area, icons), selected);
    }
}
