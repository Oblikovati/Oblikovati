// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    identity::RecipeIdError,
    sketch::{SketchEntityId, SketchError, SketchSplineHandle2D},
};

/// One active-state request with an already validated endpoint position.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchSplineHandleEdit {
    pub parent: SketchEntityId,
    pub fit_index: usize,
    pub endpoint: Option<[f64; 2]>,
}

/// The active handle after an edit; `None` records a successful deactivation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchSplineHandleReport {
    pub handle: Option<SketchSplineHandle2D>,
}

impl DocumentSnapshot {
    /// Activate, edit, or deactivate one fit-spline handle atomically; example: `snapshot.edit_spline_handle(sketch, edit, ids)?`.
    pub fn edit_spline_handle(
        &mut self,
        sketch: DocumentSketchId,
        edit: SketchSplineHandleEdit,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchSplineHandleReport, SketchSplineHandleEditError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        let report = apply_edit(&mut geometry, edit, ids)?;
        self.sketches.replace_geometry(sketch, geometry)?;
        Ok(report)
    }
}

fn apply_edit(
    geometry: &mut crate::sketch::Sketch2D,
    edit: SketchSplineHandleEdit,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchSplineHandleReport, SketchSplineHandleEditError> {
    let Some(endpoint) = edit.endpoint else {
        return deactivate(geometry, edit);
    };
    let handle = active_handle(geometry, edit, endpoint, ids)?;
    Ok(SketchSplineHandleReport {
        handle: Some(handle),
    })
}

fn deactivate(
    geometry: &mut crate::sketch::Sketch2D,
    edit: SketchSplineHandleEdit,
) -> Result<SketchSplineHandleReport, SketchSplineHandleEditError> {
    let Ok(handle) = geometry.spline_handle(edit.parent, edit.fit_index) else {
        return Ok(SketchSplineHandleReport { handle: None });
    };
    geometry.delete_entities(&[handle.end()])?;
    Ok(SketchSplineHandleReport { handle: None })
}

fn active_handle(
    geometry: &mut crate::sketch::Sketch2D,
    edit: SketchSplineHandleEdit,
    endpoint: [f64; 2],
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchSplineHandle2D, SketchSplineHandleEditError> {
    if let Ok(handle) = geometry.spline_handle(edit.parent, edit.fit_index) {
        geometry.set_position(handle.end(), endpoint)?;
        return Ok(handle);
    }
    let end = ids.allocate_sketch_entity()?;
    geometry.add_defining_point(end, endpoint)?;
    let handle = SketchSplineHandle2D::new(edit.parent, edit.fit_index, end)?;
    geometry.add_spline_handle(handle)?;
    Ok(handle)
}

/// Keeps sketch, document, and permanent-allocation failures distinct.
#[derive(Debug)]
pub enum SketchSplineHandleEditError {
    Sketch(SketchError),
    Identity(RecipeIdError),
}

impl From<SketchError> for SketchSplineHandleEditError {
    fn from(cause: SketchError) -> Self {
        Self::Sketch(cause)
    }
}
impl From<RecipeIdError> for SketchSplineHandleEditError {
    fn from(cause: RecipeIdError) -> Self {
        Self::Identity(cause)
    }
}
impl std::fmt::Display for SketchSplineHandleEditError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Sketch(cause) => write!(formatter, "spline handle edit: {cause}"),
            Self::Identity(cause) => write!(formatter, "spline handle identity: {cause}"),
        }
    }
}
impl std::error::Error for SketchSplineHandleEditError {}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sketch::{SketchSpline2D, SketchSplineKind};
    use oblikovati_api::types::SPLINE_FIT_SMOOTH;

    #[test]
    fn activation_reserves_a_hidden_endpoint_and_deactivation_retires_it() {
        let mut geometry = fit_spline();
        let mut ids = crate::identity::RecipeIdAllocator::default();
        let mut sketch_ids = crate::identity::RecipeIdAllocator::default();
        let mut feature_ids = crate::identity::RecipeIdAllocator::default();
        let mut sketch_entity_ids = crate::identity::RecipeIdAllocator::default();
        let mut sketch_constraint_ids = super::super::SketchConstraintReservations::default();
        let mut dimension_names = super::super::SketchDimensionNameReservations::default();
        let mut derived_high_water = 0;
        let mut reservations = RecipeReservations {
            ids: &mut ids,
            sketch_ids: &mut sketch_ids,
            feature_ids: &mut feature_ids,
            sketch_entity_ids: &mut sketch_entity_ids,
            sketch_constraint_ids: &mut sketch_constraint_ids,
            dimension_names: &mut dimension_names,
            derived_high_water: &mut derived_high_water,
        };
        let edit = SketchSplineHandleEdit {
            parent: SketchEntityId(10),
            fit_index: 1,
            endpoint: Some([3., 0.]),
        };
        let active = apply_edit(&mut geometry, edit, &mut reservations)
            .unwrap()
            .handle
            .unwrap();
        assert!(!geometry.entity_ids().contains(&active.end()));
        assert_eq!(geometry.point(active.end()).unwrap().position, [3., 0.]);
        let inactive = apply_edit(
            &mut geometry,
            SketchSplineHandleEdit {
                endpoint: None,
                ..edit
            },
            &mut reservations,
        )
        .unwrap();
        assert_eq!(inactive.handle, None);
        assert!(geometry.point(active.end()).is_err());
    }

    fn fit_spline() -> crate::sketch::Sketch2D {
        let mut geometry = crate::sketch::Sketch2D::default();
        for (id, position) in [(20, [0., 0.]), (21, [1., 1.]), (22, [2., 0.])] {
            geometry.add_point(SketchEntityId(id), position).unwrap();
        }
        geometry
            .add_spline(
                SketchSpline2D::new(
                    SketchEntityId(10),
                    vec![SketchEntityId(20), SketchEntityId(21), SketchEntityId(22)],
                    false,
                    SketchSplineKind::Fit {
                        method: SPLINE_FIT_SMOOTH,
                    },
                )
                .unwrap(),
            )
            .unwrap();
        geometry
    }
}
