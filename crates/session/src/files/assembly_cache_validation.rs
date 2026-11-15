// SPDX-License-Identifier: GPL-2.0-only
//! Qualify saved assembly presentation against its current source graph.
use super::*;
use oblikovati_model::{assembly::AssemblyOccurrenceRecipe, identity::RecipeDigest};
use oblikovati_persistence::package::{AssemblySourceRecord, SavedAssemblyView};
use std::collections::BTreeSet;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Borrow the validated presentation of an express assembly, if it is open in that mode.
    /// Example: `files.express_assembly_view(id)?` before drawing the saved view.
    pub fn express_assembly_view(
        &self,
        id: u64,
    ) -> Result<Option<&SavedAssemblyView>, FileWorkspaceError> {
        Ok(self.retained_file(id)?.express_view.as_ref())
    }

    /// Validate the cached presentation against the files and catalog used to create it.
    /// Example: `files.validated_assembly_view(path, &package)?` before Express open.
    pub fn validated_assembly_view(
        &mut self,
        path: &Path,
        package: &DocumentPackage,
    ) -> Result<Option<SavedAssemblyView>, String> {
        let Some(view) = package
            .restore_saved_assembly_view()
            .map_err(|e| e.to_string())?
        else {
            return Ok(None);
        };
        if !path.is_absolute() || self.assembly_view_catalog_digest()? != view.catalog_digest {
            return Err(format!(
                "assembly cache {path:?}: expected an absolute path and current catalog"
            ));
        }
        self.validate_cache_graph(path, &view)?;
        Ok(Some(view))
    }

    fn validate_cache_graph(
        &mut self,
        path: &Path,
        view: &SavedAssemblyView,
    ) -> Result<(), String> {
        let mut records = view.sources.iter();
        let mut ancestors = BTreeSet::from([path.to_owned()]);
        self.validate_cache_children(path, &view.occurrences, &[], &mut records, &mut ancestors)?;
        if let Some(extra) = records.next() {
            return Err(format!(
                "assembly cache source {:?}: expected no extra occurrence",
                extra.occurrence_path
            ));
        }
        Ok(())
    }

    fn validate_cache_children<'a>(
        &mut self,
        parent: &Path,
        occurrences: &[AssemblyOccurrenceRecipe],
        parent_index: &[usize],
        records: &mut std::slice::Iter<'a, AssemblySourceRecord>,
        ancestors: &mut BTreeSet<PathBuf>,
    ) -> Result<(), String> {
        for (index, occurrence) in occurrences
            .iter()
            .enumerate()
            .filter(|(_, item)| !item.virtual_)
        {
            let mut child_index = parent_index.to_vec();
            child_index.push(index);
            self.validate_cache_child(parent, occurrence, &child_index, records, ancestors)?;
        }
        Ok(())
    }

    fn validate_cache_child<'a>(
        &mut self,
        parent: &Path,
        occurrence: &AssemblyOccurrenceRecipe,
        index: &[usize],
        records: &mut std::slice::Iter<'a, AssemblySourceRecord>,
        ancestors: &mut BTreeSet<PathBuf>,
    ) -> Result<(), String> {
        let record = next_cache_record(records, occurrence, index)?;
        let (path, package) = self.validate_cache_source(parent, record)?;
        if !ancestors.insert(path.clone()) {
            return Err(format!(
                "assembly cache source {path:?}: expected an acyclic source graph"
            ));
        }
        let children = package
            .assembly_occurrences()
            .map_err(|e| e.to_string())?
            .unwrap_or_default();
        let result = self.validate_cache_children(&path, &children, index, records, ancestors);
        ancestors.remove(&path);
        result
    }

    fn validate_cache_source(
        &mut self,
        parent: &Path,
        record: &AssemblySourceRecord,
    ) -> Result<(PathBuf, DocumentPackage), String> {
        let path = self.resolve_cache_source(parent, record)?;
        self.reject_dirty_cache_source(&path)?;
        let bytes = self
            .files
            .read(&path)
            .map_err(|e| format!("assembly cache source {path:?}: expected readable bytes: {e}"))?;
        let package = checked_cache_package(&path, record, &bytes)?;
        Ok((path, package))
    }

    fn resolve_cache_source(
        &mut self,
        parent: &Path,
        record: &AssemblySourceRecord,
    ) -> Result<PathBuf, String> {
        let folder = parent.parent().ok_or_else(|| {
            format!("assembly cache parent {parent:?}: expected a parent directory")
        })?;
        let requested = folder.join(&record.reference);
        let path = self.files.canonical_path(&requested).map_err(|e| {
            format!("assembly cache source {requested:?}: expected an existing source: {e}")
        })?;
        if path != record.resolved_path {
            return Err(format!(
                "assembly cache source {path:?}: expected saved path {:?}",
                record.resolved_path
            ));
        }
        Ok(path)
    }

    fn reject_dirty_cache_source(&self, path: &Path) -> Result<(), String> {
        let Some(&id) = self.paths.get(path) else {
            return Ok(());
        };
        let session = self.workspace.session(id).map_err(|e| e.to_string())?;
        if session.in_transaction() || self.workspace.needs_save(id).map_err(|e| e.to_string())? {
            return Err(format!(
                "assembly cache source {path:?}: expected a saved committed document"
            ));
        }
        Ok(())
    }
}

fn next_cache_record<'a>(
    records: &mut std::slice::Iter<'a, AssemblySourceRecord>,
    occurrence: &AssemblyOccurrenceRecipe,
    index: &[usize],
) -> Result<&'a AssemblySourceRecord, String> {
    let record = records
        .next()
        .ok_or_else(|| format!("assembly cache {index:?}: expected a source record"))?;
    if record.occurrence_path != index
        || record.reference != occurrence.component
        || index.len() > 128
    {
        return Err(format!(
            "assembly cache source {:?}: expected occurrence {index:?} and reference {:?}",
            record.occurrence_path, occurrence.component
        ));
    }
    Ok(record)
}

fn checked_cache_package(
    path: &Path,
    record: &AssemblySourceRecord,
    bytes: &[u8],
) -> Result<DocumentPackage, String> {
    if RecipeDigest::of_recipe(bytes).to_string() != record.digest {
        return Err(format!(
            "assembly cache source {path:?}: expected saved content digest {}",
            record.digest
        ));
    }
    let package = DocumentPackage::decode(bytes).map_err(|e| e.to_string())?;
    let identity = package.identity_record().map_err(|e| e.to_string())?;
    if identity.as_ref().map(|item| item.internal_name.as_str()) != Some(record.identity.as_str()) {
        return Err(format!(
            "assembly cache source {path:?}: expected saved identity {:?}",
            record.identity
        ));
    }
    Ok(package)
}
