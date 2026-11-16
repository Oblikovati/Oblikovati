// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::feature::FeatureId;
use oblikovati_model::feature::work_geometry::DatumLength;
use oblikovati_session::{SessionConditionalSketch, TransactionObserver};

pub(super) fn check<O: TransactionObserver>(
    evidence: &SessionConditionalSketch<'_, O>,
    feature: FeatureId,
) {
    let displacement = evidence
        .conditional_prism_displacement(feature, [Some([0., 0.]), Some([0., 0.]), None])
        .unwrap();
    assert_eq!(displacement.feature(), feature);
    let axes = displacement.axes();
    assert_eq!(axes[0].bounds(), [0., 0.]);
    assert_eq!(axes[1].bounds(), [0., 0.]);
    assert_eq!(axes[0].source(), DatumLength::Constant(0.));
    assert_eq!(axes[2].source(), DatumLength::Parameter(1));
    assert!(axes[0].parameter().is_none());
    let size = evidence
        .projection()
        .unwrap()
        .snapshot()
        .parameters
        .model_value(1)
        .unwrap()
        .value;
    assert!(axes[2].bounds()[0] <= size && size <= axes[2].bounds()[1]);
    assert_eq!(axes[2].parameter().unwrap().id(), 1);
    let source = evidence
        .projection()
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap()
        .source
        .clone();
    assert_eq!(axes[2].parameter().unwrap().sources(), &[(1, source)]);
    assert!(
        evidence
            .conditional_prism_displacement(feature, [None, Some([0., 0.]), None])
            .is_err()
    );
    assert!(
        evidence
            .conditional_prism_displacement(feature, [Some([1., 2.]), Some([0., 0.]), None])
            .is_err()
    );
    assert!(
        evidence
            .conditional_prism_displacement(
                feature,
                [Some([0., 0.]), Some([0., 0.]), Some([0., 1.])]
            )
            .is_err()
    );
    assert!(
        evidence
            .conditional_prism_displacement(feature, [Some([f64::NAN, 0.]), Some([0., 0.]), None])
            .is_err()
    );
    assert!(
        evidence
            .conditional_prism_displacement(
                oblikovati_model::feature::FeatureId(u64::MAX),
                [Some([0., 0.]), Some([0., 0.]), None]
            )
            .is_err()
    );
}
