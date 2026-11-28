// SPDX-License-Identifier: GPL-2.0-only
//! Create, redefine, and toggle visibility of User Coordinate System features (#3918).
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::PartFeatureDefinition,
    ucs::{DEFAULT_UCS_PREFIX, UcsDefinition},
};

impl DocumentSnapshot {
    /// Allocate the next default UCS name (`UCS1`, `UCS2`, ...); example: `snapshot.next_ucs_name()`.
    pub fn next_ucs_name(&self) -> String {
        let mut ordinal = 1usize;
        loop {
            let candidate = format!("{DEFAULT_UCS_PREFIX}{ordinal}");
            let taken = self
                .work_geometry
                .features()
                .iter()
                .any(|feature| feature.name() == candidate);
            if !taken {
                return candidate;
            }
            ordinal += 1;
        }
    }

    /// Append a User Coordinate System datum feature; example: `snapshot.create_ucs("UCS1", def, ids)?`.
    pub fn create_ucs(
        &mut self,
        name: &str,
        definition: UcsDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature()?;
        let visible = definition.is_visible();
        let reads = definition.reads();
        let feature = AuthoredFeature::new(
            id,
            name,
            PartFeatureDefinition::Ucs(Box::new(definition)),
            reads,
        );
        self.work_geometry.append(feature).map_err(|cause| {
            HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                cause,
            ))
        })?;
        self.work_geometry
            .set_datum_visible(id, visible)
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })?;
        Ok(id)
    }

    /// Replace an existing UCS definition in place; example: `snapshot.edit_ucs(id, def)?`.
    pub fn edit_ucs(
        &mut self,
        feature: FeatureId,
        definition: UcsDefinition,
    ) -> Result<(), HistoryError> {
        let visible = definition.is_visible();
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Ucs(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })?;
        self.work_geometry
            .set_datum_visible(feature, visible)
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
    }

    /// Toggle persisted visibility of an existing UCS feature; example: `snapshot.set_ucs_visible(id, false)?`.
    pub fn set_ucs_visible(
        &mut self,
        feature: FeatureId,
        visible: bool,
    ) -> Result<(), HistoryError> {
        let current = self
            .work_geometry
            .get(feature)
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })?
            .definition();
        let PartFeatureDefinition::Ucs(definition) = current else {
            return Err(HistoryError::PartProgram(
                crate::feature::part::PartFeatureProgramError::Sequence(
                    crate::feature::FeatureSequenceError::Dependency(
                        crate::dependency::DependencyError::Missing(feature),
                    ),
                ),
            ));
        };
        let updated = definition.with_visible(visible);
        self.edit_ucs(feature, updated)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        identity::RecipeIdAllocator, parameter::ParameterCollection, units::UnitsOfMeasure,
    };

    fn empty_history() -> DocumentEditHistory {
        let snapshot = DocumentSnapshot {
            assembly: None,
            drawing: None,
            model_states: None,
            body_names: Default::default(),
            design_views: Vec::new(),
            part_design_views: None,
            work_geometry: Default::default(),
            assets: Default::default(),
            sketch_settings: None,
            sketches: Default::default(),
            attributes: Default::default(),
            attachments: Default::default(),
            interests: Default::default(),
            resources: Default::default(),
            parameters: ParameterCollection::new(UnitsOfMeasure::default()),
            properties: Default::default(),
        };
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
    }

    #[test]
    fn ucs_create_redefine_visibility_and_recipe_round_trip() {
        let mut history = empty_history();
        let token = history.begin("Create UCS").unwrap();
        let def = UcsDefinition::new(
            [1.0, 2.0, 3.0],
            [1.0, 0.0, 0.0],
            [0.0, 1.0, 0.0],
            Some(vec![4, 5]),
            true,
        )
        .unwrap();
        let id = history
            .edit(&token, |snapshot, ids| {
                let name = snapshot.next_ucs_name();
                assert_eq!(name, "UCS1");
                snapshot.create_ucs(&name, def.clone(), ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();

        let recipe = history.snapshot().work_geometry.part_recipe().unwrap();
        assert_eq!(recipe.version, 69);
        assert_eq!(recipe.features[0].definition.version, 59);
        let restored = crate::feature::FeatureSequence::from_part_recipe(&recipe).unwrap();
        assert!(
            history
                .snapshot()
                .work_geometry
                .same_work_plane_recipe(&restored)
        );

        let eval = history
            .snapshot()
            .work_geometry
            .evaluate_work_geometry(&history.snapshot().parameters)
            .unwrap();
        assert_eq!(eval.ucs(id).unwrap(), def.resolved());

        let token = history.begin("Hide UCS").unwrap();
        history
            .edit(&token, |snapshot, _| snapshot.set_ucs_visible(id, false))
            .unwrap();
        history.commit(&token, false).unwrap();
        assert!(
            !history
                .snapshot()
                .work_geometry
                .get(id)
                .unwrap()
                .is_visible()
        );

        let token = history.begin("Redefine UCS").unwrap();
        let moved = UcsDefinition::new(
            [5.0, 6.0, 7.0],
            [0.0, 1.0, 0.0],
            [0.0, 0.0, 1.0],
            None,
            true,
        )
        .unwrap();
        history
            .edit(&token, |snapshot, _| snapshot.edit_ucs(id, moved.clone()))
            .unwrap();
        history.commit(&token, false).unwrap();
        let eval2 = history
            .snapshot()
            .work_geometry
            .evaluate_work_geometry(&history.snapshot().parameters)
            .unwrap();
        assert_eq!(eval2.ucs(id).unwrap(), moved.resolved());
    }
}
