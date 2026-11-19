// SPDX-License-Identifier: GPL-2.0-only
//! Apply model browser selections and representation commands.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn browser_choice(
        &mut self,
        document: u64,
        choice: &oblikovati_gui::GuiBrowserChoice,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiBrowserChoice::{
            ActivateModelState, ActivatePartView, Body, BodyVisible, CopyModelState, CopyPartView,
            CreateModelState, LockPartView, Nothing, Occurrence, PartViewBodyVisibility,
            RenameModelState, RenamePartView, Root, SetOccurrenceSuppressed, ShowFeatureDimensions,
            SketchDimensionsShown, ToggleModelStateEditScope,
        };
        self.require_browser_document(document)?;
        if matches!(choice, Body { toggle: true, .. })
            && self.chosen_model_face(document).is_some()
            && self.current_face_appearance_picks(document).is_empty()
        {
            return Err(format!(
                "document {document}: expected a named face before extending the selection"
            ));
        }
        if let SetOccurrenceSuppressed { path, suppressed } = choice {
            self.document_host()
                .set_occurrence_suppressed(document, path, *suppressed)
                .map_err(|cause| cause.to_string())?;
            self.selected_occurrence = None;
            self.model_face = None;
            self.model_edges.clear();
            self.face_appearance_picks.clear();
            return Ok(true);
        }
        if let Occurrence(path) = choice {
            let browser = self.gui_browser_snapshot(document)?;
            if !super::model_browser::occurrences::contains_occurrence(&browser.occurrences, path) {
                return Err(format!(
                    "occurrence path {path:?}: expected a current assembly occurrence"
                ));
            }
            self.files
                .commands(document)
                .map_err(|cause| cause.to_string())?
                .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                .map_err(|cause| cause.to_string())?;
            self.selected_occurrence = Some((document, path.clone()));
            self.model_face = None;
            self.model_edges.clear();
            self.face_appearance_picks.clear();
            return Ok(true);
        }
        if let RenameModelState { id, name } = choice {
            return self.rename_model_state(*id, name);
        }
        if let RenamePartView { id, name } = choice {
            oblikovati_api::client::Client::new(self.document_host())
                .part_design_views()
                .rename(&oblikovati_api::wire::RenamePartDesignViewArgs {
                    id: *id,
                    name: name.clone(),
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let LockPartView { id, locked } = choice {
            oblikovati_api::client::Client::new(self.document_host())
                .part_design_views()
                .set_options(&oblikovati_api::wire::SetPartDesignViewOptionsArgs {
                    id: *id,
                    locked: Some(std::boxed::Box::new(*locked)),
                    ..Default::default()
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let PartViewBodyVisibility { id, visible } = choice {
            oblikovati_api::client::Client::new(self.document_host())
                .part_design_views()
                .set_body_visibility(&oblikovati_api::wire::SetPartDesignViewBodyVisibilityArgs {
                    id: *id,
                    body_keys: Some(Vec::new()),
                    visible: *visible,
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if matches!(choice, CreateModelState) {
            return self.create_active_model_state(document);
        }
        if matches!(choice, ToggleModelStateEditScope) {
            return self.toggle_model_state_scope(document);
        }
        if let CopyModelState(id) = choice {
            return self.copy_model_state(*id);
        }
        if let ActivateModelState(id) = choice {
            oblikovati_api::client::Client::new(self.document_host())
                .model_states()
                .activate(&oblikovati_api::wire::RepRef { id: *id })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let ShowFeatureDimensions(feature) = choice {
            self.shown_feature_dimensions = feature.map(|feature| (document, feature));
            return Ok(true);
        }
        if let SketchDimensionsShown { sketch, shown } = choice {
            return self.set_sketch_dimensions_shown(document, *sketch, *shown);
        }
        if let BodyVisible { key, visible } = choice {
            use oblikovati_api::contract::PartDesignView as _;
            let active = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot()
                .part_design_views
                .as_ref()
                .and_then(|views| views.active())
                .map(|view| view.id());
            if let Some(id) = active {
                oblikovati_api::client::Client::new(self.document_host())
                    .part_design_views()
                    .set_body_visibility(
                        &oblikovati_api::wire::SetPartDesignViewBodyVisibilityArgs {
                            id,
                            body_keys: Some(vec![key.clone()]),
                            visible: *visible,
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                return Ok(true);
            }
        }
        let mut commands = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?;
        match choice {
            CreateModelState
            | CopyPartView(_)
            | ActivatePartView(_)
            | LockPartView { .. }
            | PartViewBodyVisibility { .. }
            | CopyModelState(_)
            | RenameModelState { .. }
            | RenamePartView { .. }
            | ActivateModelState(_)
            | ToggleModelStateEditScope
            | ShowFeatureDimensions(_)
            | SketchDimensionsShown { .. }
            | SetOccurrenceSuppressed { .. }
            | Occurrence(_) => {}
            Nothing => {
                commands
                    .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                    .map_err(|cause| cause.to_string())?;
            }
            Root => commands.set_browser_root_selection(),
            Body {
                key,
                feature,
                toggle,
            } => commands.apply_browser_body_click(
                key,
                *feature,
                if *toggle {
                    SelectionClickMode::Toggle
                } else {
                    SelectionClickMode::Replace
                },
            ),
            BodyVisible { key, visible } => {
                commands.set_body_visible(key, *visible)?;
            }
        }
        if matches!(choice, Nothing | Root | Body { toggle: false, .. }) {
            self.model_face = None;
            self.model_edges.clear();
            self.face_appearance_picks.clear();
            self.selected_occurrence = None;
        }
        Ok(true)
    }
}
