// SPDX-License-Identifier: GPL-2.0-only
//! Compose referenced component scenes in occurrence order.
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
pub(super) use oblikovati_viewport::assembly_presentation::MAX_ASSEMBLY_DEPTH;

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
        let generation = match files.workspace().session(document) {
            Ok(session) => session.generation(),
            Err(error) => {
                let _ = self.empty();
                return Err(error.to_string());
            }
        };
        let selection = self.occurrence_selection.clone();
        let mut source = AssemblySceneSource {
            scenes: self,
            files,
        };
        let mut result = oblikovati_viewport::assembly_presentation::present_assembly(
            &mut source,
            document,
            selection.as_deref(),
        )?;
        result.status.source = Some((document, generation));
        Ok(result)
    }
}

struct AssemblySceneSource<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> {
    scenes: &'a mut DesktopPartScenes,
    files: &'a mut FileWorkspace<O, W, F>,
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>
    oblikovati_viewport::assembly_presentation::AssemblyPresentationSource
    for AssemblySceneSource<'_, O, W, F>
{
    type Output = HeadSceneUpdate;
    fn empty(&mut self) -> Self::Output {
        self.scenes.empty()
    }
    fn occurrences(&self, owner: u64) -> Result<(u64, Vec<AssemblyOccurrenceRecipe>), String> {
        let session = self
            .files
            .workspace()
            .session(owner)
            .map_err(|error| error.to_string())?;
        let occurrences = session
            .snapshot()
            .assembly
            .clone()
            .ok_or_else(|| format!("document {owner}: expected an assembly"))?;
        Ok((session.generation(), occurrences))
    }
    fn component(&self, owner: u64, occurrence: &AssemblyOccurrenceRecipe) -> Result<u64, String> {
        component_document(self.files, owner, occurrence)
    }
    fn is_assembly(&self, component: u64) -> Result<bool, String> {
        Ok(self
            .files
            .workspace()
            .session(component)
            .map_err(|error| error.to_string())?
            .snapshot()
            .assembly
            .is_some())
    }
    fn part(&mut self, component: u64) -> Result<Self::Output, String> {
        self.scenes.evaluate(self.files, component)
    }
    fn place(
        &self,
        scene: &mut Self::Output,
        occurrence: &AssemblyOccurrenceRecipe,
    ) -> Result<(), String> {
        scene.scene = scene.scene.placed(&occurrence.placement()?)?;
        Ok(())
    }
    fn select(&self, scene: &mut Self::Output) {
        for triangle in &mut scene.scene.triangles {
            triangle.color = CHOSEN_BODY_COLOR;
        }
    }
    fn append(&self, result: &mut Self::Output, scene: Self::Output) {
        result.scene.triangles.extend(scene.scene.triangles);
        result.status.visible_bodies += scene.status.visible_bodies;
        result.status.health.extend(scene.status.health);
        result.status.scene_errors.extend(scene.status.scene_errors);
        result
            .status
            .presentation_declines
            .extend(scene.status.presentation_declines);
    }
    fn fail(&self, result: &mut Self::Output, message: String) {
        result.status.scene_errors.push(message);
    }
}
