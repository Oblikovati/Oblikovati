// SPDX-License-Identifier: GPL-2.0-only
//! Independent full replay for projection fixture comparisons.

pub(crate) fn evaluate(
    history: &oblikovati_model::history::DocumentEditHistory,
    document: oblikovati_model::identity::DocumentGuid,
) -> Result<
    oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    String,
> {
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        history,
        document,
        oblikovati_drawing::modelling_policy(),
    );
    history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .map_err(|cause| cause.to_string())
}
