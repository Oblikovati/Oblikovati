// SPDX-License-Identifier: GPL-2.0-only
//! Certified Emboss evaluation on top of the named sketch-prism Boolean pipeline.
use crate::{
    CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure, EvaluationFailureKind,
    EvaluationPhase, NamedSolid,
};
use oblikovati_model::feature::{
    FeatureId,
    part::{PartFeatureEvaluator, PartFeatureFailure},
    sketch_emboss::{EmbossType, ResolvedSketchEmboss},
    sketch_prism::{PrismBoolean, ResolvedSketchPrism, ThroughAllDirection},
};

const PLANE_INTERIOR_EPS: f64 = 1e-6;

impl CertifiedNamedSolidEvaluator<'_> {
    /// Evaluate an Emboss (raise, engrave, or level-to-plane) against a live named target solid.
    pub(crate) fn evaluate_named_emboss(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchEmboss<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(
                        EvaluationFailureKind::InvalidResult,
                        EvaluationEntity::Body,
                        message,
                    )
                    .for_feature(feature),
            )
        };
        if input.wrap_to_face {
            if input.emboss_type == EmbossType::EmbossEngraveFromPlane {
                return Err(failure(
                    "emboss: Wrap to Face is not supported for Emboss/Engrave from Plane".into(),
                ));
            }
            let Some(face_key) = input.wrap_face else {
                return Err(failure(
                    "emboss: Wrap to Face requires a selected face".into(),
                ));
            };
            self.resolve_planar_face(body, face_key)
                .map_err(|cause| failure(format!("emboss wrap face: {cause}")))?;
        }
        let make_prism = |signed_depth: f64| ResolvedSketchPrism {
            sketch: oblikovati_model::history::ResolvedDocumentSketch {
                sketch: input.sketch.sketch,
                plane: input.sketch.plane,
                verification: input.sketch.verification,
            },
            sweep_start: input.sketch.plane,
            selection: input.selection,
            additional_outputs: &[],
            displacement: [0., 0., signed_depth],
            taper_radians: input.taper,
        };
        match input.emboss_type {
            EmbossType::EmbossFromFace => {
                let signed = if input.direction_reversed {
                    -input.depth
                } else {
                    input.depth
                };
                self.evaluate_named_prism_boolean(
                    feature,
                    body,
                    make_prism(signed),
                    PrismBoolean::Join,
                )
            }
            EmbossType::EngraveFromFace => {
                let signed = if input.direction_reversed {
                    input.depth
                } else {
                    -input.depth
                };
                self.evaluate_named_prism_boolean(
                    feature,
                    body,
                    make_prism(signed),
                    PrismBoolean::Cut,
                )
            }
            EmbossType::EmbossEngraveFromPlane => {
                let (min_t, max_t) = self
                    .prism_through_all_span(body, input.sketch.plane, ThroughAllDirection::Positive)
                    .map_err(|cause| failure(format!("emboss from plane: {cause}")))?;
                if min_t >= -PLANE_INTERIOR_EPS || max_t <= PLANE_INTERIOR_EPS {
                    return Err(failure(
                        "Edit Emboss Feature Failed: Emboss/Engrave from Plane requires the sketch plane to intersect the solid interior".into(),
                    ));
                }
                let raised = self.evaluate_named_prism_boolean(
                    feature,
                    body,
                    make_prism(input.depth),
                    PrismBoolean::Join,
                )?;
                self.evaluate_named_prism_boolean(
                    feature,
                    &raised,
                    make_prism(-input.depth),
                    PrismBoolean::Cut,
                )
            }
        }
    }
}
