// SPDX-License-Identifier: GPL-2.0-only
//! Prepare a populated layout from the selected Part, before publishing a new document.
use super::*;
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_automation::DrawingViewProjectionAccess;
use oblikovati_gui::{GuiDrawingLayoutAction, GuiLayoutComponent, GuiProjectTemplate};
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::{
    drawing::DrawingDocument,
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator},
};
use oblikovati_persistence::package::{DocumentPackage, SavedDrawingView, SavedDrawingViews};
use std::{collections::BTreeMap, path::Path, sync::Arc};

struct LayoutSourceRevision {
    generation: u64,
    recipe: Arc<DocumentSnapshot>,
}

struct PreparedLayout {
    source: u64,
    reference: String,
    focus: Option<u64>,
    package: DocumentPackage,
    deferred: bool,
    transient_source: bool,
}

#[derive(Default)]
pub(in super::super) struct LayoutCreation {
    captured: BTreeMap<u64, LayoutSourceRevision>,
    pending: Option<PreparedLayout>,
    suppress_warning: bool,
}

impl LayoutCreation {
    pub(in super::super) fn warning(&self) -> Option<String> {
        let pending = self.pending.as_ref()?;
        let name = Path::new(&pending.reference).file_name()?.to_str()?;
        Some(name.to_owned())
    }
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in super::super) fn drawing_layout_action(
        &mut self,
        request: &GuiDrawingLayoutAction,
    ) -> Result<bool, String> {
        match request {
            GuiDrawingLayoutAction::CaptureSources => self.capture_layout_sources(),
            GuiDrawingLayoutAction::Cancel => self.clear_layout_request(),
            GuiDrawingLayoutAction::Confirm {
                template,
                component,
            } => self.confirm_layout(template, component),
            GuiDrawingLayoutAction::Acknowledge { suppress } => {
                let Some(prepared) = self.layout_creation.pending.take() else {
                    return Ok(false);
                };
                self.publish_layout(prepared)?;
                self.layout_creation.suppress_warning |= suppress;
                Ok(true)
            }
        }
    }

    fn capture_layout_sources(&mut self) -> Result<bool, String> {
        self.clear_layout_request()?;
        self.layout_creation.captured = self
            .files
            .workspace()
            .documents()
            .iter()
            .filter_map(|&id| self.stale_layout_source(id).map(|revision| (id, revision)))
            .collect();
        Ok(false)
    }

    pub(super) fn clear_layout_request(&mut self) -> Result<bool, String> {
        self.layout_creation.captured.clear();
        if let Some(pending) = self.layout_creation.pending.take()
            && pending.transient_source
        {
            self.files
                .close_discarding_changes(pending.source)
                .map_err(|e| e.to_string())?;
        }
        Ok(false)
    }

    fn stale_layout_source(&self, id: u64) -> Option<LayoutSourceRevision> {
        let session = self.files.workspace().session(id).ok()?;
        let policy = self.drawing_projection_access.resolution_policy();
        let slot = session.owned_part_slot(Some(policy))?;
        if !oblikovati_automation::part_replay_is_stale(session, Some(slot)) {
            return None;
        }
        Some(LayoutSourceRevision {
            generation: session.generation(),
            recipe: slot.replay_snapshot(session).ok()??,
        })
    }

    fn confirm_layout(
        &mut self,
        template: &GuiProjectTemplate,
        component: &GuiLayoutComponent,
    ) -> Result<bool, String> {
        let package = self.project_template_package(template)?;
        let focus = self.files.workspace().active_document();
        let before = self.files.workspace().documents().to_vec();
        let source = self.resolve_layout_component(component)?;
        let transient_source = !before.contains(&source);
        let prepared = self.prepare_layout(package, source, focus, transient_source);
        self.layout_creation.captured.clear();
        let prepared = match prepared {
            Ok(layout) => layout,
            Err(error) => {
                if transient_source {
                    self.files
                        .close_discarding_changes(source)
                        .map_err(|e| format!("{error}. Closing rejected component: {e}"))?;
                }
                return Err(error);
            }
        };
        if prepared.deferred && !self.layout_creation.suppress_warning {
            self.layout_creation.pending = Some(prepared);
            return Ok(true);
        }
        self.publish_layout(prepared)
    }

    fn resolve_layout_component(&mut self, component: &GuiLayoutComponent) -> Result<u64, String> {
        match component {
            GuiLayoutComponent::Open(id) => Ok(*id),
            GuiLayoutComponent::Named(name) => {
                let path = Path::new(name);
                if path.is_absolute() {
                    if let Some(id) = self
                        .files
                        .resolve_parameter_source(name)
                        .map_err(|e| e.to_string())?
                    {
                        return Ok(id);
                    }
                    return self
                        .document_host()
                        .open_path(path, false)
                        .map_err(|e| e.to_string());
                }
                let workspace = self.files.workspace();
                let mut matching = workspace.documents().iter().copied().filter(|&id| {
                    workspace
                        .display_name(id)
                        .is_ok_and(|display| display == name)
                });
                let id = matching.next().ok_or_else(|| {
                    format!("Component {name:?}: expected an open Part or a full file path")
                })?;
                if matching.next().is_some() {
                    return Err(format!(
                        "Component {name:?}: multiple documents match that name"
                    ));
                }
                Ok(id)
            }
        }
    }

    fn prepare_layout(
        &mut self,
        package: DocumentPackage,
        source: u64,
        focus: Option<u64>,
        transient_source: bool,
    ) -> Result<PreparedLayout, String> {
        if self
            .files
            .document_type(source)
            .map_err(|e| e.to_string())?
            != DOCUMENT_PART
        {
            return Err(format!(
                "Drawing component {source}: expected a Part document"
            ));
        }
        let reference = self
            .files
            .parameter_source_name(source)
            .map_err(|e| e.to_string())?;
        let mut drawing = package
            .restore_drawing()
            .map_err(|e| e.to_string())?
            .ok_or("Expected a drawing recipe in the project template")?;
        if drawing.sheets().iter().all(|sheet| sheet.views.is_empty()) {
            return Err("The drawing template has no authored layout views".into());
        }
        drawing
            .set_model_reference(&reference)
            .map_err(|e| e.to_string())?;
        let mut package = package
            .with_drawing(Some(&drawing))
            .map_err(|e| e.to_string())?;
        let prior = self
            .layout_creation
            .captured
            .remove(&source)
            .filter(|snapshot| {
                self.files
                    .workspace()
                    .session(source)
                    .is_ok_and(|s| s.generation() == snapshot.generation)
            })
            .or_else(|| self.stale_layout_source(source));
        if let Some(prior) = &prior {
            let views = self.project_layout_snapshot(source, &drawing, &prior.recipe)?;
            package = package
                .with_saved_drawing_views(&views)
                .map_err(|e| e.to_string())?;
        } else {
            self.with_drawing_part(source, |part, history, guid| {
                sample_layout(&drawing, history, guid, part).map(|_| ())
            })?;
        }
        Ok(PreparedLayout {
            source,
            reference,
            focus,
            package,
            deferred: prior.is_some(),
            transient_source,
        })
    }

    fn project_layout_snapshot(
        &self,
        source: u64,
        drawing: &DrawingDocument,
        snapshot: &DocumentSnapshot,
    ) -> Result<SavedDrawingViews, String> {
        let workspace = self.files.workspace();
        let session = workspace.session(source).map_err(|e| e.to_string())?;
        let guid = workspace
            .identity(source)
            .map_err(|e| e.to_string())?
            .internal_name;
        let ids = RecipeIdAllocator::restore(
            Some(session.history().high_water()),
            snapshot.parameters.ids(),
        )
        .map_err(|e| e.to_string())?;
        let history = DocumentEditHistory::new(snapshot.clone(), ids).map_err(|e| e.to_string())?;
        let mut backend = CertifiedNamedSolidEvaluator::with_policy(
            &history,
            guid,
            self.drawing_projection_access.resolution_policy(),
        );
        let result = history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .map_err(|e| e.to_string())?;
        sample_layout(drawing, &history, guid, &result)
    }

    fn publish_layout(&mut self, layout: PreparedLayout) -> Result<bool, String> {
        let mut created = None;
        let result = (|| {
            let id = self.create_project_recipe(GuiTemplateKind::Drawing, layout.package)?;
            created = Some(id);
            self.files
                .attach_drawing_reference(
                    id,
                    layout.reference,
                    layout.source,
                    layout.transient_source,
                )
                .map_err(|e| e.to_string())?;
            if layout.deferred {
                self.files
                    .defer_drawing_updates(id)
                    .map_err(|e| e.to_string())?;
            }
            Ok(true)
        })();
        if result.is_err() {
            if let Some(id) = created {
                self.files
                    .close_discarding_changes(id)
                    .map_err(|e| e.to_string())?;
            }
            if layout.transient_source
                && self.files.workspace().documents().contains(&layout.source)
            {
                self.files
                    .close_discarding_changes(layout.source)
                    .map_err(|e| e.to_string())?;
            }
            if let Some(focus) = layout.focus {
                self.files.activate(focus).map_err(|e| e.to_string())?;
            }
        }
        result
    }
}

fn sample_layout(
    drawing: &DrawingDocument,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
    part: &oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
) -> Result<SavedDrawingViews, String> {
    let mut views = Vec::new();
    for sheet in drawing.sheets() {
        for view in &sheet.views {
            let lines = oblikovati_drawing::DrawingProjectionService
                .saved_lines_current(history, guid, view, part)?;
            views.push(SavedDrawingView {
                sheet: sheet.name.clone(),
                view: view.name.clone(),
                lines,
            });
        }
    }
    Ok(SavedDrawingViews {
        version: SavedDrawingViews::VERSION,
        drawing_recipe: drawing.recipe().clone(),
        views,
    })
}
