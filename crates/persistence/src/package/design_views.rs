// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const SECTION: &str = "designViews";
impl DocumentPackage {
    /// Restore a document's named design views; example: `package.restore_design_views()?`.
    pub fn restore_design_views(&self) -> Result<Vec<String>, PackageCodecError> {
        let Some(section) = self.root.get(SECTION) else {
            return Ok(Vec::new());
        };
        serde_yaml_ng::from_value(section.clone()).map_err(|cause| {
            PackageCodecError(format!(
                "{SECTION} {section:?}: expected a list of view names: {cause}"
            ))
        })
    }
    /// Capture the names, leaving a file with none exactly as it was.
    /// Example: `let saved = package.with_design_views(&names)?;`.
    pub fn with_design_views(&self, names: &[String]) -> Result<Self, PackageCodecError> {
        if self.restore_design_views()? == names {
            return Ok(self.clone());
        }
        let mut candidate = self.clone();
        let list = names
            .iter()
            .map(|name| Value::from(name.as_str()))
            .collect();
        candidate.root.insert(SECTION.into(), Value::Sequence(list));
        Ok(candidate)
    }
}
/// Read a closed file's view names for its open options; a file that cannot be read has none.
/// Example: `design_view_names_of_file(Path::new("/work/part.opd"))`.
pub fn design_view_names_of_file(path: &std::path::Path) -> Vec<String> {
    let Ok(encoded) = std::fs::read(path) else {
        return Vec::new();
    };
    DocumentPackage::decode(&encoded)
        .and_then(|package| package.open_option_design_view_names())
        .unwrap_or_default()
}

impl DocumentPackage {
    fn open_option_design_view_names(&self) -> Result<Vec<String>, PackageCodecError> {
        use oblikovati_api::contract::PartDesignView as _;
        if self.manifest.document_type != oblikovati_api::types::DOCUMENT_PART.0 {
            return self.restore_design_views();
        }
        let mut ids = self.restore_document_parameters()?.ids;
        let Some(views) = self.restore_part_design_views(&mut ids)? else {
            return self.restore_design_views();
        };
        Ok(views
            .list()
            .iter()
            .filter(|view| !view.primary())
            .map(|view| view.name())
            .collect())
    }
}

/// A closed file's properties by name and text, whichever set holds them; a file that cannot
/// be read has none. Example: `property_rows_of_file(path)` for the Open dialog's Details.
pub fn property_rows_of_file(path: &std::path::Path) -> Vec<(String, String)> {
    let Ok(encoded) = std::fs::read(path) else {
        return Vec::new();
    };
    let Ok(properties) = DocumentPackage::decode(&encoded).and_then(|p| p.restore_properties())
    else {
        return Vec::new();
    };
    properties
        .sets()
        .iter()
        .flat_map(|set| set.properties())
        .map(|property| (property.name.clone(), value_text(&property.value)))
        .collect()
}
fn value_text(value: &oblikovati_api::types::VariantValue) -> String {
    use oblikovati_api::types::VariantValue::String as Text;
    match value {
        Text(text) => text.clone(),
        other => format!("{other:?}"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn view_names_round_trip_and_an_empty_list_leaves_the_package_unchanged() {
        let package = DocumentPackage::empty_container();
        assert!(package.restore_design_views().unwrap().is_empty());
        assert_eq!(package.with_design_views(&[]).unwrap().root, package.root);
        let names = vec!["Front".to_owned()];
        let saved = package.with_design_views(&names).unwrap();
        assert_eq!(saved.restore_design_views().unwrap(), names);
    }
    #[test]
    fn a_saved_section_passes_the_file_validator() {
        let mut root = Mapping::new();
        root.insert(SECTION.into(), Value::Sequence(vec![Value::from("Front")]));
        super::super::validation::validate_root(&root).unwrap();
    }
    #[test]
    fn a_malformed_section_names_what_was_expected() {
        let mut package = DocumentPackage::empty_container();
        package.root.insert(SECTION.into(), Value::from(3));
        let cause = package.restore_design_views().unwrap_err().to_string();
        assert!(cause.contains("expected a list of view names"), "{cause}");
    }
    #[test]
    fn part_open_options_include_saved_views_without_repeating_primary() {
        let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Part");
        let mut views = oblikovati_model::part_design_view::PartDesignViews::default();
        views.copy(1, 6, "Review").unwrap();
        let stored = package.with_part_design_views(Some(&views)).unwrap();
        let names = stored.open_option_design_view_names().unwrap();
        assert_eq!(names, ["Isometric", "Front", "Top", "Right", "Review"]);
    }
}
