// SPDX-License-Identifier: GPL-2.0-only
//! Ordered assembly presentation, independent of a window or persisted cache.
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

/// The same finite nesting limit used by assembly selection.
pub const MAX_ASSEMBLY_DEPTH: usize = 128;

/// Supplies component scenes and combines them at occurrence boundaries.
/// Example: a desktop adapter evaluates a part and presents its result without changing its recipe.
pub trait AssemblyPresentationSource {
    /// An accumulated result; example: a scene with diagnostics and visible-body counts.
    type Output;
    /// Start an empty result; example: `source.empty()` for a new assembly node.
    fn empty(&mut self) -> Self::Output;
    /// Read the current owner and authored occurrences; example: `source.occurrences(id)?`.
    fn occurrences(&self, owner: u64) -> Result<(u64, Vec<AssemblyOccurrenceRecipe>), String>;
    /// Resolve one component without guessing; example: `source.component(owner, occurrence)?`.
    fn component(&self, owner: u64, occurrence: &AssemblyOccurrenceRecipe) -> Result<u64, String>;
    /// Tell whether a resolved component has children; example: `source.is_assembly(id)?`.
    fn is_assembly(&self, component: u64) -> Result<bool, String>;
    /// Evaluate one analytic part for display; example: `source.part(id)?`.
    fn part(&mut self, component: u64) -> Result<Self::Output, String>;
    /// Apply one occurrence placement; example: `source.place(scene, occurrence)?`.
    fn place(
        &self,
        scene: &mut Self::Output,
        occurrence: &AssemblyOccurrenceRecipe,
    ) -> Result<(), String>;
    /// Apply transient selection styling; example: `source.select(scene)`.
    fn select(&self, scene: &mut Self::Output);
    /// Append a completed child in occurrence order; example: `source.append(parent, child)`.
    fn append(&self, parent: &mut Self::Output, child: Self::Output);
    /// Retain a local failure; example: `source.fail(parent, message)`.
    fn fail(&self, parent: &mut Self::Output, message: String);
}

/// Present a referenced assembly without taking ownership of its model.
/// Selection paths are session-scoped; `None` produces an unselected presentation for caching.
/// Example: `present_assembly(&mut source, document, None)?` for a native scene.
pub fn present_assembly<S: AssemblyPresentationSource>(
    source: &mut S,
    document: u64,
    selection: Option<&[(u64, u64, usize)]>,
) -> Result<S::Output, String> {
    present_nested(source, document, &[], selection)
}

fn present_nested<S: AssemblyPresentationSource>(
    source: &mut S,
    document: u64,
    parent: &[(u64, u64, usize)],
    selection: Option<&[(u64, u64, usize)]>,
) -> Result<S::Output, String> {
    let mut result = source.empty();
    if parent.len() > MAX_ASSEMBLY_DEPTH {
        return Err(format!(
            "assembly document {document}: expected at most 128 nested levels"
        ));
    }
    let (generation, occurrences) = source.occurrences(document)?;
    for (ordinal, occurrence) in occurrences
        .iter()
        .enumerate()
        .filter(|(_, item)| !item.suppressed && !item.virtual_)
    {
        append_occurrence(
            source,
            &mut result,
            (document, generation),
            ordinal,
            occurrence,
            parent,
            selection,
        );
    }
    Ok(result)
}

fn append_occurrence<S: AssemblyPresentationSource>(
    source: &mut S,
    result: &mut S::Output,
    owner: (u64, u64),
    ordinal: usize,
    occurrence: &AssemblyOccurrenceRecipe,
    parent: &[(u64, u64, usize)],
    selection: Option<&[(u64, u64, usize)]>,
) {
    let mut path = parent.to_vec();
    path.push((owner.0, owner.1, ordinal));
    match present_occurrence(source, owner.0, occurrence, &path, selection) {
        Ok(scene) => source.append(result, scene),
        Err(error) => source.fail(result, format!("occurrence {:?}: {error}", occurrence.name)),
    }
}

fn present_occurrence<S: AssemblyPresentationSource>(
    source: &mut S,
    owner: u64,
    occurrence: &AssemblyOccurrenceRecipe,
    path: &[(u64, u64, usize)],
    selection: Option<&[(u64, u64, usize)]>,
) -> Result<S::Output, String> {
    let component = source.component(owner, occurrence)?;
    let mut result = if source.is_assembly(component)? {
        present_nested(source, component, path, selection)?
    } else {
        source.part(component)?
    };
    source.place(&mut result, occurrence)?;
    if selection == Some(path) {
        source.select(&mut result);
    }
    Ok(result)
}

#[cfg(test)]
#[path = "../tests/assembly_presentation.rs"]
mod tests;
