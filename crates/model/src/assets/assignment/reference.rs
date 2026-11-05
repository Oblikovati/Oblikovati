// SPDX-License-Identifier: GPL-2.0-only
/// One persistent entity in an occurrence path; the owning assignment store supplies document context.
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct AssignmentReference {
    reference_key: String,
    occurrence_path: Vec<String>,
}
impl AssignmentReference {
    /// Retain exact identity without geometric matching; example: `AssignmentReference::new("body-key", vec![])?`.
    pub fn new(reference_key: &str, occurrence_path: Vec<String>) -> Result<Self, String> {
        for component in
            std::iter::once(reference_key).chain(occurrence_path.iter().map(String::as_str))
        {
            if component.is_empty() || component.contains('\0') {
                return Err(format!(
                    "assignment reference component {component:?}: expected nonempty text without NUL"
                ));
            }
        }
        Ok(Self {
            reference_key: reference_key.into(),
            occurrence_path,
        })
    }
    /// Borrow the opaque persistent key; example: `reference.reference_key()` for storage.
    pub fn reference_key(&self) -> &str {
        &self.reference_key
    }
    /// Borrow the occurrence identity path; example: `reference.occurrence_path()` for storage.
    pub fn occurrence_path(&self) -> &[String] {
        &self.occurrence_path
    }
}
