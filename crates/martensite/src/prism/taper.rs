// SPDX-License-Identifier: GPL-2.0-only
//! Apply a signed extrusion taper with exact history from the native draft builder.
use super::*;
use martensite_tkbrep::{topods, topods::Shape};
use martensite_tkernel::standard::Failure;
use martensite_tkmath::gp::{Direction3, Plane3, Point3, Vector3};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIDraftAngle;

#[derive(Clone, Copy)]
pub(crate) struct PrismTaper {
    plane: SketchPlane,
    vector: Vector3,
    radians: f64,
}

pub(super) struct TaperedPrism {
    pub(super) shape: Shape,
    pub(super) faces: Vec<(PrismFaceRole, Shape)>,
    pub(super) subentities: Vec<Shape>,
}

impl PrismTaper {
    pub(crate) fn new(plane: SketchPlane, vector: Vector3, radians: f64) -> Option<Self> {
        (radians != 0.).then_some(Self {
            plane,
            vector,
            radians,
        })
    }

    pub(super) fn apply(
        self,
        shape: &Shape,
        faces: &[(PrismFaceRole, Shape)],
        subentities: &[Shape],
    ) -> Result<TaperedPrism, EvaluationFailure> {
        let [x, y, z] = self.plane.recipe().origin;
        let direction = Direction3::new(self.vector.x(), self.vector.y(), self.vector.z())
            .map_err(|cause| native(Phase::Placement, Entity::Body, "taper direction", cause))?;
        let neutral = Plane3::new(Point3::new(x, y, z), direction).map_err(|cause| {
            native(Phase::Placement, Entity::Body, "taper neutral plane", cause)
        })?;
        let mut draft = BRepOffsetAPIDraftAngle::with_shape(shape)
            .map_err(|cause| native(Phase::Prism, Entity::Body, "initialize taper", cause))?;
        for (role, side) in faces {
            if matches!(role, PrismFaceRole::ProfileEdge(_)) {
                let entity = Entity::Face(*role);
                let face = topods::face(side).map_err(|cause| {
                    native(Phase::Prism, entity.clone(), "taper side face", cause)
                })?;
                // The native draft sign is opposite the widening sign of Taper A.
                draft
                    .add(&face, direction, -self.radians, &neutral, true)
                    .map_err(|cause| {
                        native(Phase::Prism, entity.clone(), "add taper face", cause)
                    })?;
                if !draft.add_done().map_err(|cause| {
                    native(Phase::Prism, entity.clone(), "check taper face", cause)
                })? {
                    return Err(draft_decline(&draft, entity));
                }
            }
        }
        draft
            .build(&Default::default())
            .map_err(|cause| native(Phase::Prism, Entity::Body, "build taper", cause))?;
        if !draft.is_done() {
            return Err(draft_decline(&draft, Entity::Body));
        }
        let shape = draft
            .shape()
            .cloned()
            .map_err(|cause| native(Phase::Prism, Entity::Body, "tapered prism", cause))?;
        crate::checked_shape::require_geometry(&shape, "tapered extrusion")?;
        let image = |source, entity| {
            draft
                .modified_shape(source)
                .map_err(|cause| native(Phase::History, entity, "taper topology history", cause))
        };
        let faces = faces
            .iter()
            .map(|(role, source)| Ok((*role, image(source, Entity::Face(*role))?)))
            .collect::<Result<_, EvaluationFailure>>()?;
        let subentities = subentities
            .iter()
            .map(|source| image(source, Entity::Body))
            .collect::<Result<_, _>>()?;
        Ok(TaperedPrism {
            shape,
            faces,
            subentities,
        })
    }
}

fn draft_decline(draft: &BRepOffsetAPIDraftAngle, entity: Entity) -> EvaluationFailure {
    match draft.status() {
        Ok(status) => Phase::Prism.refused(
            Kind::InvalidGeometry,
            entity,
            format!("Extrude Taper A failed with native draft status {status:?}."),
        ),
        Err(cause) => native(Phase::Prism, entity, "taper status", cause),
    }
}

fn native(phase: Phase, entity: Entity, message: &str, cause: Failure) -> EvaluationFailure {
    phase.native(entity, message.to_owned(), cause)
}
