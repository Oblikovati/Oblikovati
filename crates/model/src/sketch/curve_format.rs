// SPDX-License-Identifier: GPL-2.0-only
use super::{OwnedSketchGeometry, Sketch2D, SketchEntityId, SketchError};
use std::collections::BTreeSet;
/// Native Format intent; construction applies to curves and centerline applies only to lines.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchCurveFormat {
    Construction,
    Centerline,
}
/// Applicable selection and actual metadata changes are distinct: a centerline can stay construction.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct SketchCurveFormatChange {
    pub applicable: Vec<SketchEntityId>,
    pub changed: Vec<SketchEntityId>,
}
impl SketchCurveFormat {
    /// Read the Format caption; example: `SketchCurveFormat::Construction.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Construction => "Construction",
            Self::Centerline => "Centerline",
        }
    }
    fn accepts(self, geometry: OwnedSketchGeometry<'_>) -> bool {
        match self {
            Self::Construction => !matches!(geometry, OwnedSketchGeometry::Point(_)),
            Self::Centerline => matches!(geometry, OwnedSketchGeometry::Line(_)),
        }
    }
}
impl Sketch2D {
    /// Invert each applicable curve role atomically in authored order without solving coordinates.
    /// Example: `sketch.toggle_curve_format(&[line, circle], SketchCurveFormat::Construction)?;`.
    pub fn toggle_curve_format(
        &mut self,
        entities: &[SketchEntityId],
        format: SketchCurveFormat,
    ) -> Result<SketchCurveFormatChange, SketchError> {
        let applicable = self.checked_format_selection(entities, format)?;
        let changed = applicable
            .iter()
            .copied()
            .filter(|id| self.toggle_owned_curve_role(*id, format))
            .collect();
        Ok(SketchCurveFormatChange {
            applicable,
            changed,
        })
    }
    fn checked_format_selection(
        &self,
        entities: &[SketchEntityId],
        format: SketchCurveFormat,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        require_unique_format_ids(entities, format)?;
        let index = self.read_index();
        let accepted: BTreeSet<_> = entities
            .iter()
            .copied()
            .map(|id| index.entity(id).map(|shape| (id, format.accepts(shape))))
            .collect::<Result<Vec<_>, _>>()?
            .into_iter()
            .filter_map(|(id, accepted)| accepted.then_some(id))
            .collect();
        Ok(self
            .entity_ids()
            .iter()
            .copied()
            .filter(|id| accepted.contains(id))
            .collect())
    }
    fn toggle_owned_curve_role(&mut self, id: SketchEntityId, format: SketchCurveFormat) -> bool {
        // Go #2015 reads effective construction, including the independent axis role.
        let enabled = match format {
            SketchCurveFormat::Construction => {
                !(self.construction_curves.contains(&id) || self.centerlines.contains(&id))
            }
            SketchCurveFormat::Centerline => !self.centerlines.contains(&id),
        };
        let flags = match format {
            SketchCurveFormat::Construction => &mut self.construction_curves,
            SketchCurveFormat::Centerline => &mut self.centerlines,
        };
        if enabled {
            flags.insert(id)
        } else {
            flags.remove(&id)
        }
    }
}

fn require_unique_format_ids(
    entities: &[SketchEntityId],
    format: SketchCurveFormat,
) -> Result<(), SketchError> {
    let unique: BTreeSet<_> = entities.iter().copied().collect();
    if unique.len() != entities.len() {
        return Err(SketchError::Definition(format!(
            "{format:?} selection {entities:?}: expected unique owned geometry IDs"
        )));
    }
    Ok(())
}
