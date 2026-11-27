// SPDX-License-Identifier: GPL-2.0-only
//! Map a picked sketch curve to its authored geometric relation identities.
use oblikovati_api::{types, wire::ConstraintInfo};
use oblikovati_gui::GuiSketchRelation;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};

pub(super) fn related_relations(
    geometry: &Sketch2D,
    curve: SketchEntityId,
    rows: &[ConstraintInfo],
) -> Result<Vec<GuiSketchRelation>, String> {
    let mut entities = geometry
        .read_index()
        .entity(curve)
        .map_err(|e| e.to_string())?
        .points();
    entities.push(curve);
    rows.iter()
        .filter(|row| {
            row.deletable
                && row.entities.as_ref().is_some_and(|refs| {
                    refs.iter()
                        .any(|id| entities.contains(&SketchEntityId(*id)))
                })
        })
        .map(|row| relation_row(geometry, row))
        .collect()
}
fn relation_row(geometry: &Sketch2D, row: &ConstraintInfo) -> Result<GuiSketchRelation, String> {
    let index = usize::try_from(row.index)
        .map_err(|_| format!("constraint index {}: expected a nonnegative row", row.index))?;
    let relation = geometry.geometric_constraints().nth(index).ok_or_else(|| {
        format!(
            "constraint index {}: expected an owned geometric relation",
            row.index
        )
    })?;
    Ok(GuiSketchRelation {
        id: relation.id.0,
        label: relation_label(&row.kind),
    })
}
fn relation_label(kind: &str) -> String {
    let names = [
        (types::GEO_CONSTRAINT_COINCIDENT, "Coincident"),
        (types::GEO_CONSTRAINT_COLLINEAR, "Collinear"),
        (types::GEO_CONSTRAINT_CONCENTRIC, "Concentric"),
        (types::GEO_CONSTRAINT_EQUAL_LENGTH, "Equal Length"),
        (types::GEO_CONSTRAINT_EQUAL_RADIUS, "Equal Radius"),
        (types::GEO_CONSTRAINT_FIX, "Fix"),
        (types::GEO_CONSTRAINT_GROUND, "Ground"),
        (types::GEO_CONSTRAINT_HORIZONTAL, "Horizontal"),
        (types::GEO_CONSTRAINT_HORIZONTAL_ALIGN, "Horizontal"),
        (types::GEO_CONSTRAINT_MIDPOINT, "Midpoint"),
        (types::GEO_CONSTRAINT_PARALLEL, "Parallel"),
        (types::GEO_CONSTRAINT_PERPENDICULAR, "Perpendicular"),
        (types::GEO_CONSTRAINT_POINT_ON_CIRCLE, "Point on Circle"),
        (types::GEO_CONSTRAINT_POINT_ON_LINE, "Point on Line"),
        (types::GEO_CONSTRAINT_TANGENT, "Tangent"),
        (types::GEO_CONSTRAINT_VERTICAL, "Vertical"),
        (types::GEO_CONSTRAINT_VERTICAL_ALIGN, "Vertical"),
    ];
    names
        .into_iter()
        .find(|(id, _)| id.0 == kind)
        .map_or_else(|| kind.to_owned(), |(_, label)| label.to_owned())
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
    #[test]
    fn curve_relations_match_direct_operands_and_owned_endpoints_in_stable_order() {
        let mut geometry = Sketch2D::default();
        for (id, point) in [(1, [0., 0.]), (2, [2., 0.]), (3, [3., 0.]), (4, [5., 0.])] {
            geometry.add_point(SketchEntityId(id), point).unwrap();
        }
        geometry
            .add_line(SketchEntityId(5), [SketchEntityId(1), SketchEntityId(2)])
            .unwrap();
        geometry
            .add_line(SketchEntityId(6), [SketchEntityId(3), SketchEntityId(4)])
            .unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(11),
                PlanarConstraint::Coincident {
                    points: [SketchEntityId(2), SketchEntityId(3)],
                },
            )
            .unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(12),
                PlanarConstraint::LineHorizontal {
                    line: SketchEntityId(5),
                },
            )
            .unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(13),
                PlanarConstraint::LineVertical {
                    line: SketchEntityId(6),
                },
            )
            .unwrap();
        let rows = [
            row(0, types::GEO_CONSTRAINT_COINCIDENT.0.as_ref(), &[2, 3]),
            row(1, types::GEO_CONSTRAINT_HORIZONTAL.0.as_ref(), &[5]),
            row(2, types::GEO_CONSTRAINT_VERTICAL.0.as_ref(), &[6]),
        ];
        let found = related_relations(&geometry, SketchEntityId(5), &rows).unwrap();
        assert_eq!(
            found.iter().map(|item| item.id).collect::<Vec<_>>(),
            [11, 12]
        );
        assert_eq!(
            found
                .iter()
                .map(|item| item.label.as_str())
                .collect::<Vec<_>>(),
            ["Coincident", "Horizontal"]
        );
        assert!(related_relations(&geometry, SketchEntityId(999), &rows).is_err());
    }
    fn row(index: i64, kind: &str, entities: &[u64]) -> ConstraintInfo {
        ConstraintInfo {
            index,
            kind: kind.into(),
            entities: Some(entities.to_vec()),
            deletable: true,
            ..Default::default()
        }
    }
}
