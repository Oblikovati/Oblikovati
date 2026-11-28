// SPDX-License-Identifier: GPL-2.0-only
//! Dockable blend draft; the owner holds the temporary preview (#4045, #4046).
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiFilletRequest, GuiShellAction};
use imgui::Ui;
#[path = "blend_panel.rs"]
mod panel;

#[derive(Default)]
pub(crate) struct BlendDialog {
    chamfer: bool,
    fillet_dock: crate::dockable_lifecycle::DockableState,
    chamfer_dock: crate::dockable_lifecycle::DockableState,
    edited_edges: usize,
    document: Option<u64>,
    radius: String,
    attempted: Option<GuiFilletRequest>,
    submitted: Option<GuiFilletRequest>,
    edit: Option<(oblikovati_model::feature::FeatureId, u64)>,
    attempted_edit: Option<crate::GuiFilletEditRequest>,
    submitted_edit: Option<crate::GuiFilletEditRequest>,
}
impl DockableWindow for BlendDialog {}
impl BlendDialog {
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Fillet", &self.fillet_dock)?;
        registry.register::<Self>("Chamfer", &self.chamfer_dock)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.fillet_dock
            .set_open(document.is_some() && !self.chamfer);
        self.chamfer_dock
            .set_open(document.is_some() && self.chamfer);
    }
    pub(crate) fn active(&self) -> bool {
        self.document.is_some()
    }
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.open_for(state, false);
    }
    fn open_for(&mut self, state: &GuiCommandState, chamfer: bool) {
        self.chamfer = chamfer;
        self.edited_edges = 0;
        self.set_document(state.document);
        self.radius = "2 mm".into();
        self.attempted = None;
        self.submitted = None;
        self.edit = None;
        self.attempted_edit = None;
        self.submitted_edit = None;
    }
    pub(crate) fn open_chamfer(&mut self, state: &GuiCommandState) {
        self.open_for(state, true);
        self.radius = "1 mm".into();
    }
    pub(crate) fn open_edit(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        self.open_blend_edit(state, feature, false);
    }
    pub(crate) fn open_chamfer_edit(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        self.open_blend_edit(state, feature, true);
    }
    fn open_blend_edit(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
        chamfer: bool,
    ) {
        let Some(seed) = state.browser.as_ref().and_then(|browser| {
            if chamfer {
                browser.chamfer_edits.get(&feature)
            } else {
                browser.fillet_edits.get(&feature)
            }
        }) else {
            return;
        };
        self.chamfer = chamfer;
        self.edited_edges = seed.edge_count;
        self.set_document(state.document);
        self.radius = seed.radius.clone();
        self.edit = Some((feature, seed.generation));
        self.attempted = None;
        self.submitted = None;
        self.attempted_edit = None;
        self.submitted_edit = None;
    }
    fn edit_request(&self) -> Option<crate::GuiFilletEditRequest> {
        let (feature, generation) = self.edit?;
        Some(crate::GuiFilletEditRequest {
            document: self.document?,
            generation,
            feature,
            radius: self.radius.clone(),
        })
    }
    fn request(&self, state: &GuiCommandState) -> Option<GuiFilletRequest> {
        let document = self.document?;
        let (generation, edges) = state.fillet_selection.as_ref()?;
        Some(GuiFilletRequest {
            document,
            generation: *generation,
            edges: edges.clone(),
            radius: self.radius.clone(),
        })
    }
    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.is_some() {
            self.set_document(None);
            actions.push(self.preview_action(None));
        }
        self.attempted = None;
        self.submitted = None;
        self.edit = None;
        self.attempted_edit = None;
        self.submitted_edit = None;
    }
    pub(crate) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document
                || !state.fillet_available
                || state.authoring != crate::GuiAuthoringContext::Part)
        {
            self.close(actions);
        }
        if self.submitted.as_ref().is_some_and(|request| {
            state
                .fillet_selection
                .as_ref()
                .is_none_or(|(generation, _)| *generation != request.generation)
        }) {
            self.close(actions);
        }
        if self.submitted_edit.as_ref().is_some_and(|request| {
            state
                .browser
                .as_ref()
                .and_then(|browser| {
                    if self.chamfer {
                        browser.chamfer_edits.get(&request.feature)
                    } else {
                        browser.fillet_edits.get(&request.feature)
                    }
                })
                .is_none_or(|seed| seed.generation != request.generation)
        }) {
            self.close(actions);
        }
    }
    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if self.edit.is_some() {
            let request = self.edit_request().filter(|request| {
                self.preview_edit_action(request.clone())
                    .dispatchable(state)
            });
            if request == self.attempted_edit {
                return;
            }
            if self.attempted_edit.is_some() {
                actions.push(self.preview_action(None));
            }
            self.attempted_edit = request.clone();
            if let Some(request) = request {
                actions.push(self.preview_edit_action(request));
            }
            return;
        }
        let request = self.request(state).filter(|request| {
            self.preview_action(Some(request.clone()))
                .dispatchable(state)
        });
        if request == self.attempted {
            return;
        }
        if self.attempted.is_some() {
            actions.push(self.preview_action(None));
        }
        self.attempted = request.clone();
        if let Some(request) = request {
            actions.push(self.preview_action(Some(request)));
        }
    }
    fn controls(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if self.edit.is_some() {
            self.edit_controls(ui, state, actions);
            return;
        }
        let request = self
            .request(state)
            .filter(|request| self.create_action(request.clone()).dispatchable(state));
        if (crate::theme::property_accept(ui, request.is_some())
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Enter)))
            && let Some(request) = request
        {
            self.submitted = Some(request.clone());
            actions.push(self.create_action(request));
        }
        ui.same_line_with_spacing(0., 6.);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }

    fn preview_action(&self, request: Option<GuiFilletRequest>) -> GuiShellAction {
        if self.chamfer {
            GuiShellAction::PreviewChamfer(request)
        } else {
            GuiShellAction::PreviewFillet(request)
        }
    }

    fn create_action(&self, request: GuiFilletRequest) -> GuiShellAction {
        if self.chamfer {
            GuiShellAction::CreateChamfer(request)
        } else {
            GuiShellAction::CreateFillet(request)
        }
    }
    fn preview_edit_action(&self, request: crate::GuiFilletEditRequest) -> GuiShellAction {
        if self.chamfer {
            GuiShellAction::PreviewChamferEdit(request)
        } else {
            GuiShellAction::PreviewFilletEdit(request)
        }
    }
    fn commit_edit_action(&self, request: crate::GuiFilletEditRequest) -> GuiShellAction {
        if self.chamfer {
            GuiShellAction::EditChamferDistance(request)
        } else {
            GuiShellAction::EditFilletRadius(request)
        }
    }
    fn edit_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let request = self
            .edit_request()
            .filter(|request| self.commit_edit_action(request.clone()).dispatchable(state));
        if (crate::theme::property_accept(ui, request.is_some())
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Enter)))
            && let Some(request) = request
        {
            self.submitted_edit = Some(request.clone());
            actions.push(self.commit_edit_action(request));
        }
        ui.same_line_with_spacing(0., 6.);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn selected_state() -> GuiCommandState {
        GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_edge_count: 1,
            fillet_available: true,
            fillet_selection: Some((3, vec![vec![1, 2, 3]])),
            ..Default::default()
        }
    }
    #[test]
    fn visible_body_allows_start_without_allowing_an_empty_fillet() {
        let mut state = selected_state();
        state.selected_edge_count = 0;
        state.fillet_selection = Some((3, vec![]));
        let mut browser = crate::GuiModelBrowser::default();
        browser.bodies.push(crate::GuiBrowserBody {
            name: "Solid1".into(),
            feature: oblikovati_model::feature::FeatureId(1),
            key: "body".into(),
            index: 0,
            visible: true,
            faces: 6,
            appearance: String::new(),
            physical: None,
        });
        state.browser = Some(browser);
        assert!(GuiShellAction::OpenFillet.enabled(&state));
        let mut dialog = BlendDialog::default();
        dialog.open(&state);
        let empty = dialog.request(&state).unwrap();
        assert!(!GuiShellAction::CreateFillet(empty).enabled(&state));
        let mut actions = Vec::new();
        dialog.refresh_preview(&state, &mut actions);
        assert!(actions.is_empty());
        state.browser.as_mut().unwrap().bodies[0].visible = false;
        assert!(!GuiShellAction::OpenFillet.enabled(&state));
    }
    #[test]
    fn document_change_closes_blend_before_docking_once() {
        let mut dialog = BlendDialog::default();
        let mut state = selected_state();
        dialog.open(&state);
        state.document = None;
        let mut actions = Vec::new();
        dialog.sync_before_docking(&state, &mut actions);
        dialog.sync_before_docking(&state, &mut actions);
        assert!(dialog.document.is_none());
        assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
    }

    #[test]
    fn draft_previews_each_selection_once_and_cancel_clears_it() {
        let mut dialog = BlendDialog::default();
        let mut state = selected_state();
        let mut actions = Vec::new();
        dialog.open(&state);
        dialog.refresh_preview(&state, &mut actions);
        let GuiShellAction::PreviewFillet(Some(request)) = &actions[0] else {
            panic!("expected fillet preview")
        };
        assert_eq!(request.radius, "2 mm");
        assert_eq!(request.edges, vec![vec![1, 2, 3]]);
        actions.clear();
        dialog.refresh_preview(&state, &mut actions);
        assert!(actions.is_empty());
        state.fillet_selection = Some((3, vec![vec![4, 5, 6]]));
        dialog.refresh_preview(&state, &mut actions);
        assert!(matches!(
            actions.as_slice(),
            [
                GuiShellAction::PreviewFillet(None),
                GuiShellAction::PreviewFillet(Some(_))
            ]
        ));
        actions.clear();
        dialog.close(&mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
        assert!(dialog.document.is_none());
    }

    #[test]
    fn chamfer_draft_previews_once_and_cancel_clears_it() {
        let state = selected_state();
        let mut dialog = BlendDialog::default();
        let mut actions = Vec::new();
        dialog.open_chamfer(&state);
        dialog.refresh_preview(&state, &mut actions);
        let GuiShellAction::PreviewChamfer(Some(request)) = &actions[0] else {
            panic!("expected chamfer preview")
        };
        assert_eq!(request.radius, "1 mm");
        assert!(dialog.create_action(request.clone()).enabled(&state));
        actions.clear();
        dialog.refresh_preview(&state, &mut actions);
        assert!(actions.is_empty());
        dialog.close(&mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewChamfer(None)]);
    }

    #[test]
    fn chamfer_edit_uses_distance_and_rejects_stale_generation() {
        let feature = oblikovati_model::feature::FeatureId(9);
        let mut state = selected_state();
        let mut browser = crate::GuiModelBrowser::default();
        browser.chamfer_edits.insert(
            feature,
            crate::GuiFilletEditSeed {
                generation: 3,
                edge_count: 1,
                radius: "bevel_size".into(),
            },
        );
        state.browser = Some(browser);
        let mut dialog = BlendDialog::default();
        let mut actions = Vec::new();
        dialog.open_chamfer_edit(&state, feature);
        dialog.refresh_preview(&state, &mut actions);
        let [GuiShellAction::PreviewChamferEdit(request)] = actions.as_slice() else {
            panic!("expected Chamfer edit preview")
        };
        assert_eq!(request.radius, "bevel_size");
        assert!(dialog.commit_edit_action(request.clone()).enabled(&state));
        actions.clear();
        state
            .browser
            .as_mut()
            .unwrap()
            .chamfer_edits
            .get_mut(&feature)
            .unwrap()
            .generation = 4;
        dialog.refresh_preview(&state, &mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewChamfer(None)]);
        assert!(
            !dialog
                .commit_edit_action(dialog.edit_request().unwrap())
                .enabled(&state)
        );
    }

    #[test]
    fn edit_uses_authored_radius_and_rejects_a_stale_generation() {
        let feature = oblikovati_model::feature::FeatureId(9);
        let mut state = selected_state();
        let mut browser = crate::GuiModelBrowser::default();
        browser.fillet_edits.insert(
            feature,
            crate::GuiFilletEditSeed {
                generation: 3,
                edge_count: 1,
                radius: "round_radius".into(),
            },
        );
        state.browser = Some(browser);
        let mut dialog = BlendDialog::default();
        let mut actions = Vec::new();
        dialog.open_edit(&state, feature);
        dialog.refresh_preview(&state, &mut actions);
        let [GuiShellAction::PreviewFilletEdit(request)] = actions.as_slice() else {
            panic!("expected Fillet edit preview")
        };
        assert_eq!(request.radius, "round_radius");
        assert_eq!(request.generation, 3);
        actions.clear();
        state
            .browser
            .as_mut()
            .unwrap()
            .fillet_edits
            .get_mut(&feature)
            .unwrap()
            .generation = 4;
        dialog.refresh_preview(&state, &mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
        assert!(!GuiShellAction::EditFilletRadius(dialog.edit_request().unwrap()).enabled(&state));
        actions.clear();
        dialog.close(&mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
    }
}
