// SPDX-License-Identifier: GPL-2.0-only
//! Explicit blend capability decline checks.
use super::*;

#[test]
fn unsupported_blend_declines_name_each_operation() {
    for (kind, expected) in [
        (PartBlendKind::Fillet, "fillet"),
        (PartBlendKind::Chamfer, "chamfer"),
    ] {
        let PartFeatureFailure::<String>::Unsupported(message) = kind.unsupported() else {
            panic!("blend {kind:?}: expected a named decline");
        };
        assert!(message.starts_with(expected), "{message}");
    }
}

#[test]
fn unsupported_chamfer_keeps_its_source_body() {
    let mut snapshot = document();
    let definition = oblikovati_model::feature::chamfer::ChamferDefinition::new(
        key(2, 4, "body"),
        vec![key(2, 2, "edge")],
        DatumLength::Constant(1.),
    )
    .unwrap();
    let reads = definition.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Chamfer",
            PartFeatureDefinition::Chamfer(Box::new(definition)),
            reads,
        ))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut NamedFilletBackend::default())
        .unwrap();
    assert!(result.body(FeatureId(2)).is_some());
    assert!(result.body(FeatureId(3)).is_none());
    assert!(matches!(&result.health()[2].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Unsupported(message)) if message.starts_with("chamfer")));
}
