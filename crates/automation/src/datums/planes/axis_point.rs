// SPDX-License-Identifier: GPL-2.0-only
use super::reference_only::repick;
use super::*;
use oblikovati_model::feature::work_geometry::WorkPlaneLinePoint;

fn source(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<WorkPlaneLinePoint, Error> {
    let [line, point] = reference_only::references(request)?;
    Ok(WorkPlaneLinePoint {
        line: input::axis_reference(snapshot, line)?,
        point: input::point_reference(snapshot, point)?,
    })
}
pub(in crate::datums) fn revise(
    snapshot: &DocumentSnapshot,
    mut source: WorkPlaneLinePoint,
    args: &RedefineWorkPlaneArgs,
    constructor: AxisPointPlaneConstructor,
) -> Result<WorkGeometryDefinition, Error> {
    let kind = constructor.api_kind();
    reference_only::validate(args, kind.0.as_ref(), 2)?;
    source.line = repick(snapshot, args, (0, source.line), input::axis_reference)?;
    source.point = repick(snapshot, args, (1, source.point), input::point_reference)?;
    Ok(constructor.definition(source))
}

#[derive(Clone, Copy, Debug)]
pub(in crate::datums) enum AxisPointPlaneConstructor {
    ContainsLine,
    NormalToAxis,
}
impl AxisPointPlaneConstructor {
    fn api_kind(self) -> WorkPlaneKind {
        match self {
            Self::ContainsLine => WORK_PLANE_LINE_AND_POINT,
            Self::NormalToAxis => WORK_PLANE_NORMAL_TO_CURVE,
        }
    }
    fn definition(self, source: WorkPlaneLinePoint) -> WorkGeometryDefinition {
        match self {
            Self::ContainsLine => WorkGeometryDefinition::LineThroughPoint(source),
            Self::NormalToAxis => WorkGeometryDefinition::NormalToAxis(source),
        }
    }
    pub(super) fn slots(self) -> Vec<WorkPlaneRefSlot> {
        let labels = match self {
            Self::ContainsLine => [("Line", "axis"), ("Through point", "point")],
            Self::NormalToAxis => [("Curve", "axis"), ("Point", "point")],
        };
        reference_only::slots(&labels)
    }
}
pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
    constructor: AxisPointPlaneConstructor,
) -> Result<PreparedGeometry, Error> {
    Ok(PreparedGeometry::Definition(
        constructor.definition(source(request, snapshot)?),
    ))
}
