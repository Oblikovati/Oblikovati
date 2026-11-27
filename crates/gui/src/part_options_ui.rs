// SPDX-License-Identifier: GPL-2.0-only
//! Stage the Part settings until the user applies them.
use crate::GuiShellAction;
use imgui::Ui;
use oblikovati_model::application_options::PartApplicationOptions;
use oblikovati_model::feature::work_plane::WorkPlaneOrigin;

const STARTUP_PLANES: [(Option<WorkPlaneOrigin>, &str); 4] = [
    (None, "No new sketch"),
    (Some(WorkPlaneOrigin::XY), "x-y"),
    (Some(WorkPlaneOrigin::YZ), "y-z"),
    (Some(WorkPlaneOrigin::XZ), "x-z"),
];

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
    pub(crate) fn has_changes(&self, accepted: Option<PartApplicationOptions>) -> bool {
        matches!((self.edited, accepted), (Some(edited), Some(accepted)) if edited != accepted)
    }
    pub(crate) fn has_conflict(&self, accepted: Option<PartApplicationOptions>) -> bool {
        matches!((self.original, accepted), (Some(original), Some(accepted)) if original != accepted)
    }
    pub(crate) fn apply(
        &mut self,
        accepted: Option<PartApplicationOptions>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let (Some(edited), Some(accepted)) = (self.edited, accepted)
            && edited != accepted
            && !self.has_conflict(Some(accepted))
        {
            actions.push(GuiShellAction::SetPartOptions(edited));
        }
    }
    pub(crate) fn draw(&mut self, ui: &Ui, accepted: Option<PartApplicationOptions>) {
        let Some(accepted) = accepted else {
            self.clear();
            return;
        };
        self.synchronize(accepted);
        let Some(edited) = self.edited.as_mut() else {
            return;
        };
        ui.separator();
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Part");
        }
        ui.checkbox("Look at sketch plane", &mut edited.look_at_sketch_plane);
        draw_startup_plane(ui, &mut edited.startup_sketch_plane);
        if self.original != Some(accepted) {
            ui.text_colored(
                crate::theme::error_accent(ui),
                "Part settings changed. Cancel and reopen before applying.",
            );
        }
    }
}

fn draw_startup_plane(ui: &Ui, plane: &mut Option<WorkPlaneOrigin>) {
    let label = STARTUP_PLANES[match plane {
        None => 0,
        Some(WorkPlaneOrigin::XY) => 1,
        Some(WorkPlaneOrigin::YZ) => 2,
        Some(WorkPlaneOrigin::XZ) => 3,
    }]
    .1;
    crate::theme::property_label(ui, "New part sketch");
    let Some(_combo) = ui.begin_combo("##new-part-sketch", label) else {
        return;
    };
    for (value, name) in STARTUP_PLANES {
        if ui.selectable_config(name).selected(value == *plane).build() {
            *plane = value;
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
    #[test]
    fn startup_choices_cover_each_origin_and_no_sketch_once() {
        let choices: Vec<_> = STARTUP_PLANES.iter().map(|(plane, _)| *plane).collect();
        assert_eq!(
            choices,
            [
                None,
                Some(WorkPlaneOrigin::XY),
                Some(WorkPlaneOrigin::YZ),
                Some(WorkPlaneOrigin::XZ)
            ]
        );
    }
}
