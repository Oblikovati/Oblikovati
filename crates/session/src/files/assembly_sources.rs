// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the saved sources that can justify an assembly presentation.
use super::*;
use oblikovati_model::{assembly::AssemblyOccurrenceRecipe, identity::RecipeDigest};
use oblikovati_persistence::package::AssemblySourceRecord;
use std::collections::BTreeSet;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Fingerprint the actual catalog used for display. Projects need separate provenance.
    /// Example: `files.assembly_view_catalog_digest()?` before capturing an assembly.
    pub fn assembly_view_catalog_digest(&self) -> Result<String, String> {
        if self.active_project().is_some() {
            return Err("assembly view: expected no active project until project assets have saved provenance".into());
        }
        let catalog = self.builtin_catalog()?;
        let materials: Vec<_> = catalog
            .materials()
            .ordered()
            .map(|asset| asset.info())
            .collect();
        let appearances: Vec<_> = catalog
            .appearances()
            .ordered()
            .map(|asset| asset.info())
            .collect();
        let bytes = serde_json::to_vec(&(materials, appearances)).map_err(|error| {
            format!("assembly catalog: expected serializable asset records: {error}")
        })?;
        Ok(RecipeDigest::of_recipe(&bytes).to_string())
    }

    /// Resolve the destination whose directory controls saved assembly references.
    /// Example: `files.assembly_save_destination(path)?` before capturing sources.
    pub fn assembly_save_destination(&mut self, path: &Path) -> Result<PathBuf, String> {
        checked_path(path, self.files.save_path(path)).map_err(|error| error.to_string())
    }

    /// Stage a view for one resolved destination; example: `files.stage_assembly_view(id, path, view)?`.
    pub fn stage_assembly_view(
        &mut self,
        id: u64,
        path: &Path,
        view: oblikovati_persistence::package::SavedAssemblyView,
    ) -> Result<(), String> {
        if !path.is_absolute() {
            return Err(format!(
                "assembly destination {path:?}: expected a resolved absolute path"
            ));
        }
        let file = self
            .retained
            .get_mut(&id)
            .ok_or_else(|| format!("document {id}: expected a retained file"))?;
        file.assembly_view = Some((path.to_owned(), view));
        Ok(())
    }

    /// Discard a staged view after an attempted save; example: `files.clear_assembly_view(id)`.
    pub fn clear_assembly_view(&mut self, id: u64) {
        if let Some(file) = self.retained.get_mut(&id) {
            file.assembly_view = None;
        }
    }

    /// Read and validate the saved graph for an assembly candidate at its final destination.
    /// Example: `files.assembly_saved_sources(id, destination)?` before staging a cache.
    pub fn assembly_saved_sources(
        &mut self,
        root: u64,
        destination: &Path,
    ) -> Result<Vec<AssemblySourceRecord>, String> {
        let mut records = Vec::new();
        let mut ancestors = BTreeSet::new();
        self.collect_assembly_sources(root, destination, &[], &mut ancestors, &mut records)?;
        Ok(records)
    }

    fn collect_assembly_sources(
        &mut self,
        owner: u64,
        owner_path: &Path,
        path: &[usize],
        ancestors: &mut BTreeSet<u64>,
        records: &mut Vec<AssemblySourceRecord>,
    ) -> Result<(), String> {
        if path.len() > 128 || !ancestors.insert(owner) {
            return Err(format!(
                "assembly {owner} path {path:?}: expected an acyclic graph with at most 128 nested levels"
            ));
        }
        let result = self.collect_assembly_edges(owner, owner_path, path, ancestors, records);
        ancestors.remove(&owner);
        result
    }

    fn collect_assembly_edges(
        &mut self,
        owner: u64,
        owner_path: &Path,
        path: &[usize],
        ancestors: &mut BTreeSet<u64>,
        records: &mut Vec<AssemblySourceRecord>,
    ) -> Result<(), String> {
        let occurrences = self.assembly_source_occurrences(owner)?;
        for (ordinal, occurrence) in occurrences.iter().enumerate() {
            if occurrence.virtual_ {
                continue;
            }
            if occurrence.component.is_empty() {
                return Err(format!(
                    "assembly {owner} occurrence {:?}: expected a saved component reference",
                    occurrence.name
                ));
            }
            let mut child_path = path.to_vec();
            child_path.push(ordinal);
            self.collect_assembly_edge(
                owner,
                owner_path,
                occurrence,
                &child_path,
                ancestors,
                records,
            )?;
        }
        Ok(())
    }

    fn assembly_source_occurrences(
        &self,
        owner: u64,
    ) -> Result<Vec<AssemblyOccurrenceRecipe>, String> {
        let session = self
            .workspace
            .session(owner)
            .map_err(|error| error.to_string())?;
        if session.in_transaction() || !session.snapshot().parameters.derived_tables().is_empty() {
            return Err(format!(
                "assembly source {owner}: expected a committed document without derived links"
            ));
        }
        session
            .snapshot()
            .assembly
            .clone()
            .ok_or_else(|| format!("assembly source {owner}: expected an assembly"))
    }

    fn collect_assembly_edge(
        &mut self,
        owner: u64,
        owner_path: &Path,
        occurrence: &AssemblyOccurrenceRecipe,
        path: &[usize],
        ancestors: &mut BTreeSet<u64>,
        records: &mut Vec<AssemblySourceRecord>,
    ) -> Result<(), String> {
        let name = &occurrence.component;
        let child = self.document_reference(owner, name).ok_or_else(|| {
            format!("assembly {owner} reference {name:?}: expected an owned saved component")
        })?;
        let parent = owner_path.parent().ok_or_else(|| {
            format!("assembly destination {owner_path:?}: expected a parent directory")
        })?;
        let requested = parent.join(name);
        let canonical = self.files.canonical_path(&requested).map_err(|error| {
            format!("assembly reference {requested:?}: expected an existing source: {error}")
        })?;
        if self.path(child).map_err(|error| error.to_string())? != canonical {
            return Err(format!(
                "assembly reference {name:?} resolves to {canonical:?}: expected the current source {:?}",
                self.path(child)
            ));
        }
        let record = self.saved_source_record(child, name, &canonical, path)?;
        records.push(record);
        if self
            .workspace
            .session(child)
            .map_err(|error| error.to_string())?
            .snapshot()
            .assembly
            .is_some()
        {
            self.collect_assembly_sources(child, &canonical, path, ancestors, records)?;
        }
        Ok(())
    }

    fn saved_source_record(
        &mut self,
        child: u64,
        name: &str,
        path: &Path,
        ordinal_path: &[usize],
    ) -> Result<AssemblySourceRecord, String> {
        let session = self
            .workspace
            .session(child)
            .map_err(|error| error.to_string())?;
        if session.in_transaction()
            || self
                .workspace
                .needs_save(child)
                .map_err(|error| error.to_string())?
            || !session.snapshot().parameters.derived_tables().is_empty()
        {
            return Err(format!(
                "assembly component {child} at {path:?}: expected a saved committed source without derived links"
            ));
        }
        let identity = self
            .workspace
            .identity(child)
            .map_err(|error| error.to_string())?;
        let bytes = self.files.read(path).map_err(|error| {
            format!("assembly source {path:?}: expected readable saved bytes: {error}")
        })?;
        let mut decoded = DocumentPackage::decode(&bytes).map_err(|error| error.to_string())?;
        decoded = normalize_title(decoded, path);
        let retained = normalize_title(
            self.retained_file(child)
                .map_err(|error| error.to_string())?
                .package
                .clone(),
            path,
        );
        if retained != decoded {
            return Err(format!(
                "assembly source {path:?}: expected the package loaded by document {child}"
            ));
        }
        let record = decoded
            .identity_record()
            .map_err(|error| error.to_string())?
            .ok_or_else(|| format!("assembly source {path:?}: expected a saved identity"))?;
        if record.internal_name.is_empty()
            || record
                .restore(identity)
                .map_err(|error| error.to_string())?
                != *identity
        {
            return Err(format!(
                "assembly source {path:?}: expected the session's saved identity"
            ));
        }
        Ok(AssemblySourceRecord {
            occurrence_path: ordinal_path.to_vec(),
            reference: name.to_owned(),
            resolved_path: path.into(),
            identity: identity.internal_name.to_string(),
            digest: RecipeDigest::of_recipe(&bytes).to_string(),
        })
    }
}

fn normalize_title(package: DocumentPackage, path: &Path) -> DocumentPackage {
    if name::display_override(path, &package.manifest().display_name).is_none() {
        return package.with_display_name("");
    }
    package
}
