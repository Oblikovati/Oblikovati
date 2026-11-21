// SPDX-License-Identifier: GPL-2.0-only
use std::{
    collections::BTreeMap,
    ops::{Deref, Index},
    path::{Path, PathBuf},
};

/// Resolve the virtual filesystem root; example: `fixture_native_path(Path::new("/fixtures/a.opd"))`.
pub(crate) fn fixture_native_path(path: &Path) -> PathBuf {
    // #3795: rooted Windows paths need a volume; this fake never reads the real drive.
    #[cfg(windows)]
    let root = Path::new("C:/");
    #[cfg(not(windows))]
    let root = Path::new("/");
    root.join(path)
}
/// Qualify fixture filenames without rewriting logical document names; example: `fixture_file_name("/source.opd")`.
pub(crate) fn fixture_file_name(name: &str) -> String {
    let path = Path::new(name);
    if path.has_root() {
        return fixture_native_path(path).to_string_lossy().into_owned();
    }
    name.into()
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct FixturePathMap<V>(BTreeMap<PathBuf, V>);
impl<V> Default for FixturePathMap<V> {
    fn default() -> Self {
        Self(BTreeMap::new())
    }
}
impl<V> FixturePathMap<V> {
    /// Admit one normalized key; example: `packages.insert(path, contents)`.
    pub(crate) fn insert(&mut self, path: PathBuf, value: V) -> Option<V> {
        self.0.insert(fixture_native_path(&path), value)
    }
    /// Resolve the same key for reads; example: `packages.get(Path::new("/fixtures/a.opd"))`.
    pub(crate) fn get(&self, path: &Path) -> Option<&V> {
        self.0.get(&fixture_native_path(path))
    }
    /// Inspect one virtual file; example: `packages.contains_key(path)`.
    pub(crate) fn contains_key(&self, path: &Path) -> bool {
        self.get(path).is_some()
    }
    /// Remove the same key used by reads; example: `packages.remove(path)`.
    pub(crate) fn remove(&mut self, path: &Path) -> Option<V> {
        self.0.remove(&fixture_native_path(path))
    }
    /// Reset stored files; example: `packages.clear()` to emulate absent storage.
    pub(crate) fn clear(&mut self) {
        self.0.clear();
    }
}
impl<V> Deref for FixturePathMap<V> {
    type Target = BTreeMap<PathBuf, V>;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}
impl<V, P: AsRef<Path>> Index<P> for FixturePathMap<V> {
    type Output = V;
    fn index(&self, index: P) -> &Self::Output {
        let path = index.as_ref();
        self.get(path)
            .unwrap_or_else(|| panic!("fixture path {path:?}: expected a stored virtual file"))
    }
}
#[test]
fn fixture_paths_are_absolute_and_idempotent_without_host_io() {
    for raw in ["/fixtures/a.opd", "fixtures/a.opd"] {
        let native = fixture_native_path(Path::new(raw));
        assert!(
            native.is_absolute(),
            "fixture path {native:?}: expected a fully qualified native path"
        );
        assert_eq!(fixture_native_path(&native), native);
        assert_eq!(native.file_name().unwrap(), "a.opd");
    }
}
#[test]
fn file_storage_uses_one_key_for_seeded_and_canonical_paths() {
    let mut packages = FixturePathMap::default();
    let raw = Path::new("/fixtures/a.opd");
    let native = fixture_native_path(raw);
    packages.insert(raw.into(), vec![1, 2]);
    assert_eq!(packages.insert(native.clone(), vec![3]), Some(vec![1, 2]));
    assert_eq!(packages.len(), 1);
    assert!(packages.contains_key(raw));
    assert_eq!(packages[raw], packages[&native]);
    assert!(packages.keys().all(|path| path.is_absolute()));
    assert_eq!(packages.remove(raw), Some(vec![3]));
    packages.insert(native, vec![4]);
    packages.clear();
    assert!(packages.is_empty());
}
#[test]
fn explicit_invalid_provider_responses_are_not_rewritten_as_valid_aliases() {
    let mut aliases = FixturePathMap::default();
    let requested = Path::new("/fixtures/a.opd");
    aliases.insert(requested.into(), PathBuf::from("relative"));
    assert_eq!(aliases.get(requested), Some(&PathBuf::from("relative")));
}

#[test]
fn source_names_preserve_unbound_names_and_qualify_file_references() {
    assert_eq!(fixture_file_name("Unbound source"), "Unbound source");
    let native = fixture_file_name("/source.opd");
    assert!(Path::new(&native).is_absolute());
    assert_eq!(fixture_file_name(&native), native);
}
