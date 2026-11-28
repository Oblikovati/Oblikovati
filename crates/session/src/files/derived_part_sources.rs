// SPDX-License-Identifier: GPL-2.0-only
//! Load and synchronize external source Part snapshots for Derived Part evaluation.
use super::*;
use oblikovati_model::{
    feature::part::PartFeatureDefinition,
    history::DocumentSnapshot,
    identity::{DocumentGuid, GuidSource, IdentityTransitionError},
};
use oblikovati_persistence::load::read_parameter_package;
use std::collections::BTreeSet;
use std::sync::Arc;

struct StaticDerivedGuidSource;

impl GuidSource for StaticDerivedGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        "00000000-0000-4000-8000-00000000d3a2"
            .parse::<DocumentGuid>()
            .map_err(|cause| IdentityTransitionError::Randomness(cause.to_string()))
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Load a Derived Part source snapshot from an open document or disk `.opd` package.
    pub fn load_derived_source_snapshot(
        &mut self,
        source_path: &str,
        owner_document: Option<u64>,
    ) -> Result<DocumentSnapshot, String> {
        let resolved = self.resolve_derived_source_path(source_path, owner_document);
        if let Some(&id) = self.paths.get(&resolved) {
            return self
                .workspace
                .session(id)
                .map(|session| session.snapshot().clone())
                .map_err(|error| error.to_string());
        }
        if let Ok(canonical) = self.files.canonical_path(&resolved)
            && let Some(&id) = self.paths.get(&canonical)
        {
            return self
                .workspace
                .session(id)
                .map(|session| session.snapshot().clone())
                .map_err(|error| error.to_string());
        }
        let mut guids = StaticDerivedGuidSource;
        let prepared =
            read_parameter_package(&resolved, "derived-source", &mut guids, &mut self.files)
                .map_err(|error| {
                    format!(
                        "derived source {:?}: failed to read source package: {error}",
                        resolved.display()
                    )
                })?;
        let mut snapshot = DocumentSnapshot {
            work_geometry: prepared.work_geometry.program,
            sketches: prepared.sketches.sketches,
            attributes: prepared.attributes,
            attachments: prepared.attachments,
            interests: prepared.interests,
            resources: prepared.resources,
            assets: prepared.assets,
            model_states: prepared.model_states,
            drawing: prepared.drawing,
            assembly: prepared.assembly,
            body_names: prepared.body_names,
            design_views: prepared.design_views,
            part_design_views: prepared.part_design_views,
            sketch_settings: prepared.sketch_settings,
            parameters: prepared.restored.parameters,
            properties: prepared.properties,
        };
        let solved = snapshot.solve_document_sketches(Default::default(), || false);
        snapshot.sketches = solved.candidate;
        Ok(snapshot)
    }

    /// Synchronize loaded Derived Part source snapshots onto `document`'s session.
    pub fn sync_derived_sources(
        &mut self,
        document: u64,
        extra_paths: &[&str],
    ) -> Result<(), String> {
        let mut paths = BTreeSet::new();
        {
            let session = self
                .workspace
                .session(document)
                .map_err(|error| error.to_string())?;
            for feature in session.snapshot().work_geometry.features() {
                if let PartFeatureDefinition::DerivedPart(definition) = feature.definition() {
                    paths.insert(definition.source_path().to_string());
                }
            }
        }
        for path in extra_paths {
            let trimmed = path.trim();
            if !trimmed.is_empty() {
                paths.insert(trimmed.to_string());
            }
        }
        if paths.is_empty() {
            let had_sources = self
                .workspace
                .session(document)
                .map(|session| !session.derived_sources().is_empty())
                .unwrap_or(false);
            if had_sources {
                self.commands(document)
                    .map_err(|error| error.to_string())?
                    .set_derived_sources(Arc::new(BTreeMap::new()));
            }
            return Ok(());
        }
        let mut loaded = BTreeMap::new();
        for path in paths {
            let snapshot = self.load_derived_source_snapshot(&path, Some(document))?;
            loaded.insert(path, snapshot);
        }
        self.commands(document)
            .map_err(|error| error.to_string())?
            .set_derived_sources(Arc::new(loaded));
        Ok(())
    }

    fn resolve_derived_source_path(
        &self,
        source_path: &str,
        owner_document: Option<u64>,
    ) -> PathBuf {
        let raw = PathBuf::from(source_path);
        if raw.is_absolute() {
            return raw;
        }
        if let Some(doc) = owner_document
            && let Some(owner_path) = self
                .retained
                .get(&doc)
                .and_then(|retained| retained.path.as_ref())
            && let Some(parent) = owner_path.parent()
        {
            return parent.join(&raw);
        }
        raw
    }
}
