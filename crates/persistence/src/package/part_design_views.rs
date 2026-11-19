// SPDX-License-Identifier: GPL-2.0-only
//! Persist part view recipes without changing unrelated document sections (#4118).
use super::*;
use oblikovati_api::contract::PartDesignView as _;
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::identity::RecipeIdAllocator;
use oblikovati_model::part_design_view::PartDesignViews;

const SECTION: &str = "partDesignViews";

impl DocumentPackage {
    /// Restore part views, including names from a legacy part without view recipes.
    /// Example: `package.restore_part_design_views(&mut ids)?` before adopting the document.
    pub fn restore_part_design_views(
        &self,
        ids: &mut RecipeIdAllocator,
    ) -> Result<Option<PartDesignViews>, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_PART.0 {
            return self.reject_nonpart_view_section();
        }
        let Some(section) = self.root.get(SECTION) else {
            return self.migrate_legacy_part_view_names(ids).map(Some);
        };
        let legacy = self.restore_design_views()?;
        if !legacy.is_empty() {
            return Err(PackageCodecError(format!(
                "legacy designViews {legacy:?} with {SECTION}: expected one part view source"
            )));
        }
        let views: PartDesignViews =
            serde_yaml_ng::from_value(section.clone()).map_err(|cause| {
                PackageCodecError(format!(
                    "{SECTION} {section:?}: expected a part view recipe: {cause}"
                ))
            })?;
        views.validate().map_err(PackageCodecError)?;
        Ok(Some(views))
    }

    fn reject_nonpart_view_section(&self) -> Result<Option<PartDesignViews>, PackageCodecError> {
        if self.root.contains_key(SECTION) {
            return Err(PackageCodecError(format!(
                "document type {} with {SECTION}: expected a part document",
                self.manifest.document_type
            )));
        }
        Ok(None)
    }

    fn migrate_legacy_part_view_names(
        &self,
        ids: &mut RecipeIdAllocator,
    ) -> Result<PartDesignViews, PackageCodecError> {
        let mut views = PartDesignViews::default();
        let mut seen = std::collections::BTreeSet::new();
        for name in self.restore_design_views()? {
            if !seen.insert(name.clone()) {
                return Err(PackageCodecError(format!(
                    "legacy part view name {name:?}: expected unique names"
                )));
            }
            if views.list().iter().any(|view| view.name() == name) {
                continue;
            }
            let mut id = ids
                .allocate()
                .map_err(|cause| PackageCodecError(cause.to_string()))?;
            while id <= 5 {
                id = ids
                    .allocate()
                    .map_err(|cause| PackageCodecError(cause.to_string()))?;
            }
            views.copy(1, id, &name).map_err(PackageCodecError)?;
        }
        Ok(views)
    }

    /// Store validated part views and remove their legacy name-only section.
    /// Example: `package.with_part_design_views(Some(&views))?` when saving a part.
    pub fn with_part_design_views(
        &self,
        views: Option<&PartDesignViews>,
    ) -> Result<Self, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_PART.0 {
            if views.is_some() {
                return Err(PackageCodecError(format!(
                    "document type {} with part views: expected a part document",
                    self.manifest.document_type
                )));
            }
            self.reject_nonpart_view_section()?;
            return Ok(self.clone());
        }
        let Some(views) = views else {
            if self.root.contains_key(SECTION) {
                return Err(PackageCodecError(
                    "part views None: expected retained part view recipes".into(),
                ));
            }
            return Ok(self.clone());
        };
        let mut validation_ids = RecipeIdAllocator::default();
        self.restore_part_design_views(&mut validation_ids)?;
        views.validate().map_err(PackageCodecError)?;
        let encoded = serde_yaml_ng::to_value(views).map_err(|cause| {
            PackageCodecError(format!(
                "part views {views:?}: expected serializable recipes: {cause}"
            ))
        })?;
        let mut candidate = self.clone();
        candidate.root.insert(SECTION.into(), encoded);
        candidate.root.remove("designViews");
        Ok(candidate)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::DOCUMENT_DRAWING;

    #[test]
    fn migrate_and_round_trip_part_views() {
        let package = DocumentPackage::empty(DOCUMENT_PART, "Part");
        assert!(package.with_part_design_views(None).is_ok());
        let package = package
            .with_design_views(&["Front".into(), "Review".into()])
            .unwrap();
        let views = package
            .restore_part_design_views(&mut RecipeIdAllocator::default())
            .unwrap()
            .unwrap();
        assert_eq!(views.list().last().unwrap().name(), "Review");
        let stored = package.with_part_design_views(Some(&views)).unwrap();
        assert!(stored.with_part_design_views(None).is_err());
        assert!(!stored.root.contains_key("designViews"));
        assert_eq!(
            stored
                .restore_part_design_views(&mut RecipeIdAllocator::default())
                .unwrap(),
            Some(views)
        );
        super::super::validation::validate_root(&stored.root).unwrap();
    }

    #[test]
    fn reject_bad_recipes_and_nonpart_recipes() {
        let mut part = DocumentPackage::empty(DOCUMENT_PART, "Part");
        part.root.insert(SECTION.into(), Value::from(2));
        assert!(
            part.restore_part_design_views(&mut RecipeIdAllocator::default())
                .is_err()
        );
        let mut drawing = DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing");
        assert!(
            drawing
                .restore_part_design_views(&mut RecipeIdAllocator::default())
                .unwrap()
                .is_none()
        );
        assert!(
            drawing
                .with_part_design_views(Some(&PartDesignViews::default()))
                .is_err()
        );
        drawing.root.insert(SECTION.into(), Value::Null);
        assert!(
            drawing
                .restore_part_design_views(&mut RecipeIdAllocator::default())
                .is_err()
        );
        assert!(drawing.with_part_design_views(None).is_err());
    }

    #[test]
    fn rejects_competing_or_duplicate_legacy_names() {
        let part = DocumentPackage::empty(DOCUMENT_PART, "Part");
        let duplicate = part
            .with_design_views(&["Review".into(), "Review".into()])
            .unwrap();
        assert!(
            duplicate
                .restore_part_design_views(&mut RecipeIdAllocator::default())
                .is_err()
        );
        let stored = part
            .with_part_design_views(Some(&PartDesignViews::default()))
            .unwrap();
        let competing = stored.with_design_views(&["Review".into()]).unwrap();
        assert!(
            competing
                .restore_part_design_views(&mut RecipeIdAllocator::default())
                .is_err()
        );
    }

    #[test]
    fn migrated_names_reserve_ids_above_the_existing_high_water() {
        let part = DocumentPackage::empty(DOCUMENT_PART, "Part")
            .with_design_views(&["Review".into()])
            .unwrap();
        let mut ids = RecipeIdAllocator::restore(Some(100), &[]).unwrap();
        let views = part.restore_part_design_views(&mut ids).unwrap().unwrap();
        assert_eq!(views.list().last().unwrap().id(), 101);
        assert_eq!(ids.high_water(), 101);
    }
}
