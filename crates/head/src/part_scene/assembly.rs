// SPDX-License-Identifier: GPL-2.0-only
//! Compose referenced component scenes in occurrence order.
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
pub(super) const MAX_ASSEMBLY_DEPTH: usize = 128;

#[derive(PartialEq, Eq)]
pub(super) struct SceneSource {
    pub document: u64,
    pub is_assembly: bool,
    revisions: Vec<(u64, u64, u64)>,
    failures: Vec<String>,
}

pub(super) fn active_scene_source<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<SceneSource>, String> {
    let Some(document) = files.workspace().active_document() else {
        return Ok(None);
    };
    let session = files
        .workspace()
        .session(document)
        .map_err(|error| error.to_string())?;
    let mut source = SceneSource {
        document,
        is_assembly: session.snapshot().assembly.is_some(),
        revisions: Vec::new(),
        failures: Vec::new(),
    };
    collect_revisions(files, document, &mut source, 0)?;
    Ok(Some(source))
}

fn collect_revisions<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    source: &mut SceneSource,
    depth: usize,
) -> Result<(), String> {
    if depth > MAX_ASSEMBLY_DEPTH {
        return Err(format!(
            "assembly document {document}: expected at most 128 nested levels"
        ));
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|error| error.to_string())?;
    source.revisions.push((
        document,
        session.generation(),
        session.body_visibility_revision(),
    ));
    if let Some(occurrences) = &session.snapshot().assembly {
        for occurrence in occurrences
            .iter()
            .filter(|item| !item.suppressed && !item.virtual_)
        {
            match component_document(files, document, occurrence) {
                Ok(component) => collect_revisions(files, component, source, depth + 1)?,
                Err(error) => source.failures.push(error),
            }
        }
    }
    Ok(())
}

fn component_document<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    occurrence: &AssemblyOccurrenceRecipe,
) -> Result<u64, String> {
    let name = &occurrence.component;
    files.document_reference(owner, name).ok_or_else(|| {
        let cause = files
            .document_reference_error(owner, name)
            .unwrap_or("reference is not resolved");
        format!(
            "assembly {owner} occurrence {:?} component {name:?}: {cause}",
            occurrence.name
        )
    })
}

impl DesktopPartScenes {
    /// Build a transient component presentation without changing its assembly recipe.
    /// Example: `scenes.component_preview(files, source, &placement)?` before accepting placement.
    pub(crate) fn component_preview<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        component: u64,
        placement: &oblikovati_api::types::Matrix,
    ) -> Result<ViewportScene, String> {
        let nested = files
            .workspace()
            .session(component)
            .map_err(|error| error.to_string())?
            .snapshot()
            .assembly
            .is_some();
        let result = if nested {
            self.evaluate_assembly(files, component)?
        } else {
            self.evaluate(files, component)?
        };
        if !result.status.scene_errors.is_empty()
            || !result.status.presentation_declines.is_empty()
            || result.status.unavailable_count() > 0
        {
            return Err(format!(
                "component {component}: expected a complete preview: {:?}",
                result.status
            ));
        }
        let mut scene = result.scene.placed(placement)?;
        for triangle in &mut scene.triangles {
            triangle.color = [1., 0.65, 0.1, 1.];
        }
        Ok(scene)
    }

    pub(super) fn failed_assembly_scene<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        error: String,
    ) -> Result<Option<HeadSceneUpdate>, String> {
        let document = files
            .workspace()
            .active_document()
            .ok_or_else(|| error.clone())?;
        let generation = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let failure = (document, generation, error);
        if self.scene_failure.as_ref() == Some(&failure) && !self.presentation_invalidated {
            return Ok(None);
        }
        let mut scene = self.empty();
        scene.status.source = Some((document, generation));
        scene.status.scene_errors.push(failure.2.clone());
        self.source = None;
        self.scene_failure = Some(failure);
        self.presentation_invalidated = false;
        Ok(Some(scene))
    }

    pub(super) fn evaluate_assembly<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<HeadSceneUpdate, String> {
        let mut result = self.empty();
        let generation = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?
            .generation();
        result.status.source = Some((document, generation));
        self.append_assembly(files, document, &mut result, &[])?;
        Ok(result)
    }

    fn append_assembly<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        result: &mut HeadSceneUpdate,
        parent: &[(u64, u64, usize)],
    ) -> Result<(), String> {
        if parent.len() > MAX_ASSEMBLY_DEPTH {
            return Err(format!(
                "assembly document {document}: expected at most 128 nested levels"
            ));
        }
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let generation = session.generation();
        let occurrences = session
            .snapshot()
            .assembly
            .clone()
            .ok_or_else(|| format!("document {document}: expected an assembly"))?;
        for (ordinal, occurrence) in occurrences
            .iter()
            .enumerate()
            .filter(|(_, item)| !item.suppressed && !item.virtual_)
        {
            let mut path = parent.to_vec();
            path.push((document, generation, ordinal));
            if let Err(error) = self.append_occurrence(files, document, occurrence, result, &path) {
                result
                    .status
                    .scene_errors
                    .push(format!("occurrence {:?}: {error}", occurrence.name));
            }
        }
        Ok(())
    }

    fn append_occurrence<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        owner: u64,
        occurrence: &AssemblyOccurrenceRecipe,
        result: &mut HeadSceneUpdate,
        path: &[(u64, u64, usize)],
    ) -> Result<(), String> {
        let component = component_document(files, owner, occurrence)?;
        let nested = files
            .workspace()
            .session(component)
            .map_err(|error| error.to_string())?
            .snapshot()
            .assembly
            .is_some();
        let mut scene = if nested {
            let mut child = self.empty();
            self.append_assembly(files, component, &mut child, path)?;
            child
        } else {
            self.evaluate(files, component)?
        };
        scene.scene = scene.scene.placed(&occurrence.placement()?)?;
        if self.occurrence_selection.as_deref() == Some(path) {
            for triangle in &mut scene.scene.triangles {
                triangle.color = CHOSEN_BODY_COLOR;
            }
        }
        result.scene.triangles.extend(scene.scene.triangles);
        result.status.visible_bodies += scene.status.visible_bodies;
        result.status.health.extend(scene.status.health);
        result.status.scene_errors.extend(scene.status.scene_errors);
        result
            .status
            .presentation_declines
            .extend(scene.status.presentation_declines);
        Ok(())
    }
}
