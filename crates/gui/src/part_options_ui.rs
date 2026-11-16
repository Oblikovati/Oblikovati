// SPDX-License-Identifier: GPL-2.0-only
//! Stage the Part settings until the user applies them.
use crate::GuiShellAction;
use imgui::Ui;
use oblikovati_model::application_options::PartApplicationOptions;

#[derive(Default)]
pub(crate) struct PartOptionsDraft {
    original: Option<PartApplicationOptions>,
    edited: Option<PartApplicationOptions>,
}
impl PartOptionsDraft {
    fn synchronize(&mut self, accepted: PartApplicationOptions) {
        if self.original == self.edited || self.edited == Some(accepted) {
            self.original = Some(accepted);
            self.edited = Some(accepted);
        }
    }
    pub(crate) fn clear(&mut self) {
        *self = Self::default();
    }
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        accepted: Option<PartApplicationOptions>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(accepted) = accepted else {
            self.clear();
            return;
        };
        self.synchronize(accepted);
        let Some(edited) = self.edited.as_mut() else {
            return;
        };
        ui.separator();
        ui.text("Part");
        ui.checkbox("Look at sketch plane", &mut edited.look_at_sketch_plane);
        let conflict = self.original != Some(accepted);
        if conflict {
            ui.text("Part settings changed. Revert before applying.");
        }
        let changed = *edited != accepted;
        let disabled = ui.begin_disabled(!changed || conflict);
        if ui.button_with_size("Apply Part", [0., 0.]) {
            actions.push(GuiShellAction::SetPartOptions(*edited));
        }
        disabled.end();
        ui.same_line();
        if ui.button_with_size("Revert Part", [0., 0.]) {
            self.clear();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn accepted_changes_refresh_clean_drafts_but_not_pending_edits() {
        let mut draft = PartOptionsDraft::default();
        let enabled = PartApplicationOptions {
            look_at_sketch_plane: true,
            ..Default::default()
        };
        let disabled = PartApplicationOptions {
            look_at_sketch_plane: false,
            ..Default::default()
        };
        draft.synchronize(enabled);
        draft.synchronize(disabled);
        assert_eq!(draft.edited, Some(disabled));
        draft.edited = Some(enabled);
        draft.synchronize(disabled);
        assert_eq!(draft.edited, Some(enabled));
        draft.synchronize(enabled);
        assert_eq!(draft.original, Some(enabled));
        draft.clear();
        assert!(draft.original.is_none() && draft.edited.is_none());
    }
    #[test]
    fn pending_view_edit_does_not_absorb_an_external_chamfer_change() {
        let mut draft = PartOptionsDraft::default();
        let initial = PartApplicationOptions::default();
        draft.synchronize(initial);
        draft.edited.as_mut().unwrap().look_at_sketch_plane = false;
        let external = PartApplicationOptions {
            chamfer_flat_corners: false,
            ..initial
        };
        draft.synchronize(external);
        assert_ne!(draft.original, Some(external));
        assert!(draft.edited.unwrap().chamfer_flat_corners);
    }
}
