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
    pub(super) stitch: crate::stitch_dialog::StitchDialog,
    pub(super) surface_trim: crate::surface_trim_dialog::SurfaceTrimDialog,
    pub(super) grounded_point: crate::work_point::overlay::GroundedPointHud,
    pub(super) work_plane: crate::work_plane_dialog::WorkPlaneDialog,
    pub(super) extend_surface: crate::extend_surface_dialog::ExtendSurfaceDialog,
    pub(super) boundary_patch: crate::boundary_patch_dialog::BoundaryPatchDialog,
    pub(super) ruled_surface: crate::ruled_surface_dialog::RuledSurfaceDialog,
    pub(super) replace_face: crate::replace_face_dialog::ReplaceFaceDialog,
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
        self.stitch.register_dockable(registry)?;
        self.surface_trim.register_dockable(registry)?;
        self.grounded_point.register_dockable(registry)?;
        self.extend_surface.register_dockable(registry)?;
        self.boundary_patch.register_dockable(registry)?;
        self.ruled_surface.register_dockable(registry)?;
        self.replace_face.register_dockable(registry)?;
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
        self.extend_surface.synchronize(state, actions);
        self.boundary_patch.synchronize(state, actions);
        self.ruled_surface.synchronize(state, actions);
        self.replace_face.synchronize(state, actions);
        self.stitch.synchronize(state, actions);
        self.surface_trim.synchronize(state, actions);
        self.sweep.synchronize(state, actions);
        self.loft.synchronize(state, actions);
        self.grounded_point.synchronize(state);
    }

    pub(super) fn status_prompt<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        self.work_plane
            .status_prompt()
            .or_else(|| self.unwrap.status_prompt())
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

    pub(super) fn open_extend_surface(
        &mut self,
        state: &GuiCommandState,
        action: crate::GuiExtendSurfaceAction,
        actions: &mut Vec<GuiShellAction>,
    ) {
        match action {
            crate::GuiExtendSurfaceAction::Open => self.extend_surface.open(state, None, actions),
            crate::GuiExtendSurfaceAction::OpenEdit(feature) => {
                self.extend_surface.open(state, Some(feature), actions)
            }
            _ => {}
        }
    }

    pub(super) fn close_all(&mut self, actions: &mut Vec<GuiShellAction>) {
        self.revolve.close(actions);
        self.coil.close(actions);
        self.emboss.close(actions);
        self.derive.close(actions);
        self.rib.close(actions);
        self.decal.close(actions);
        self.import.close(actions);
        self.unwrap.close(actions);
        self.sweep.close(actions);
        self.loft.close(actions);
        self.thread.close(actions);
        self.mark.close(actions);
        self.finish.close(actions);
        self.extend_surface.close(actions);
        self.boundary_patch.close(actions);
        self.ruled_surface.close(actions);
        self.replace_face.close(actions);
        self.stitch.close(actions);
        self.surface_trim.close(actions);
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
        self.stitch.panel(ui, commands, (area, icons), selected);
        self.grounded_point.panel(ui, commands, area, selected);
        self.extend_surface
            .panel(ui, commands, (area, icons), selected);
        self.boundary_patch
            .panel(ui, commands, (area, icons), selected);
        self.ruled_surface
            .panel(ui, commands, (area, icons, picked_axis), selected);
        self.replace_face
            .panel(ui, commands, (area, icons), selected);
    }
}
