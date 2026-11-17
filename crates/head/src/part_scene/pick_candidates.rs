// SPDX-License-Identifier: GPL-2.0-only
//! Partial exact hit batches. These do not yet establish competing-category coverage.
use oblikovati_session::selection::SelectionKind;

/// Match the Go picker's eight-pixel depth window at the view target (#3890).
pub(crate) fn depth_window(scale: f64, height: u32) -> Result<f64, String> {
    if !scale.is_finite() || scale <= 0. || height == 0 {
        return Err(format!(
            "pick scale {scale}, height {height}: expected positive finite scale and nonzero height"
        ));
    }
    Ok(8. * scale / f64::from(height))
}

#[derive(Clone, Debug, PartialEq)]
pub(crate) enum PickTarget {
    Edge(oblikovati_model::feature::FeatureId, Vec<u8>),
    Face(oblikovati_model::feature::FeatureId, Vec<u8>),
    Body(oblikovati_model::feature::FeatureId, String),
    Occurrence(oblikovati_gui::GuiOccurrencePath),
    WorkPlane(oblikovati_model::feature::FeatureId),
}
impl PickTarget {
    fn kind(&self) -> SelectionKind {
        match self {
            Self::Edge(..) => SelectionKind::Edge,
            Self::Face(..) => SelectionKind::Face,
            Self::Body(..) => SelectionKind::Body,
            Self::Occurrence(..) => SelectionKind::Occurrence,
            Self::WorkPlane(..) => SelectionKind::WorkPlane,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum PickCoverage {
    #[cfg(test)]
    NamedEdges,
    NamedEdgesAndFaces,
    NearestFacePerBody,
    NearestFacePerOccurrenceBody,
    VisiblePlanesAndNearestFaces,
}

#[derive(Debug)]
pub(crate) struct PickCandidate {
    pub(crate) target: PickTarget,
    pub(crate) line_parameter: f64,
}

pub(crate) struct PickCandidates {
    coverage: PickCoverage,
    candidates: Vec<PickCandidate>,
}

impl PickCandidates {
    pub(crate) fn new(coverage: PickCoverage) -> Self {
        Self {
            coverage,
            candidates: Vec::new(),
        }
    }

    pub(crate) fn include_visible_planes(&mut self) {
        self.coverage = PickCoverage::VisiblePlanesAndNearestFaces;
    }
    pub(crate) fn include_edges(&mut self) {
        self.coverage = PickCoverage::NamedEdgesAndFaces;
    }

    pub(crate) fn push(&mut self, target: PickTarget, line_parameter: f64) -> Result<(), String> {
        if !line_parameter.is_finite() {
            return Err(format!(
                "{:?} hit {line_parameter}: expected a finite line parameter",
                target.kind()
            ));
        }
        self.candidates.push(PickCandidate {
            target,
            line_parameter,
        });
        Ok(())
    }

    pub(crate) fn nearest(self) -> Result<Option<PickTarget>, String> {
        if let Some(first) = self.candidates.first()
            && self
                .candidates
                .iter()
                .any(|hit| hit.target.kind() != first.target.kind())
        {
            return Err("mixed candidate categories: expected priority arbitration with a camera-derived depth window".into());
        }
        self.resolve(
            &oblikovati_session::selection::SelectionFilter::All,
            &oblikovati_session::selection::AmbientSelectionFilter::default(),
            0.,
        )
    }

    pub(crate) fn resolve(
        self,
        allowed: &oblikovati_session::selection::SelectionFilter,
        ambient: &oblikovati_session::selection::AmbientSelectionFilter,
        window: f64,
    ) -> Result<Option<PickTarget>, String> {
        Ok(self
            .resolve_hit(allowed, ambient, window)?
            .map(|hit| hit.target))
    }

    pub(crate) fn resolve_hit(
        self,
        allowed: &oblikovati_session::selection::SelectionFilter,
        ambient: &oblikovati_session::selection::AmbientSelectionFilter,
        window: f64,
    ) -> Result<Option<PickCandidate>, String> {
        if !window.is_finite() || window < 0. {
            return Err(format!(
                "pick depth window {window}: expected a finite nonnegative distance"
            ));
        }
        let accepted: Vec<_> = self
            .candidates
            .into_iter()
            .filter(|hit| allowed.accepts(hit.target.kind()))
            .collect();
        let Some(depth) = accepted
            .iter()
            .map(|hit| hit.line_parameter)
            .min_by(f64::total_cmp)
        else {
            return Ok(None);
        };
        select_ranked(accepted, ambient, depth, window, self.coverage)
    }
}

fn select_ranked(
    candidates: Vec<PickCandidate>,
    ambient: &oblikovati_session::selection::AmbientSelectionFilter,
    depth: f64,
    window: f64,
    coverage: PickCoverage,
) -> Result<Option<PickCandidate>, String> {
    let rank = candidates
        .iter()
        .filter(|hit| hit.line_parameter - depth <= window)
        .map(|hit| ambient.rank(hit.target.kind()))
        .min()
        .ok_or_else(|| {
            format!("pick depth {depth}, window {window}: expected a candidate in range")
        })?;
    let top: Vec<_> = candidates
        .into_iter()
        .filter(|hit| {
            hit.line_parameter - depth <= window && ambient.rank(hit.target.kind()) == rank
        })
        .collect();
    nearest_ranked(top, rank, coverage)
}

fn nearest_ranked(
    candidates: Vec<PickCandidate>,
    rank: usize,
    coverage: PickCoverage,
) -> Result<Option<PickCandidate>, String> {
    let Some(nearest) = candidates
        .iter()
        .map(|hit| hit.line_parameter)
        .min_by(f64::total_cmp)
    else {
        return Ok(None);
    };
    let mut top = candidates
        .into_iter()
        .filter(|hit| hit.line_parameter == nearest);
    let Some(first) = top.next() else {
        return Ok(None);
    };
    if top.any(|candidate| candidate.target != first.target) {
        return Err(format!(
            "pick rank {rank} ({coverage:?}): expected an unambiguous identity"
        ));
    }
    Ok(Some(first))
}

#[cfg(test)]
mod tests {
    use super::*;
    fn face(id: u64) -> PickTarget {
        PickTarget::Face(oblikovati_model::feature::FeatureId(id), vec![id as u8])
    }
    #[test]
    fn arbitration_retains_the_selected_face_and_its_ray_distance() {
        use oblikovati_session::selection::{AmbientSelectionFilter, SelectionFilter};
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 3.).unwrap();
        batch.push(face(2), 2.).unwrap();
        let hit = batch
            .resolve_hit(
                &SelectionFilter::All,
                &AmbientSelectionFilter::default(),
                0.,
            )
            .unwrap()
            .unwrap();
        assert_eq!(hit.target, face(2));
        assert_eq!(hit.line_parameter, 2.);
    }
    #[test]
    fn batch_refuses_distinct_equal_depths_but_accepts_repeat_identity() {
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(7), 3.).unwrap();
        batch.push(face(7), 3.).unwrap();
        assert_eq!(batch.nearest().unwrap(), Some(face(7)));
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(7), 3.).unwrap();
        batch.push(face(8), 3.).unwrap();
        assert!(batch.nearest().unwrap_err().contains("unambiguous"));
    }
    #[test]
    fn batch_picks_nearest_and_refuses_non_finite_parameters() {
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        assert!(
            batch
                .push(face(4), f64::NAN)
                .unwrap_err()
                .contains("finite")
        );
        batch.push(face(4), 7.).unwrap();
        batch.push(face(5), 2.).unwrap();
        assert_eq!(batch.nearest().unwrap(), Some(face(5)));
    }
    #[test]
    fn a_nearer_candidate_clears_ambiguity_among_farther_hits() {
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 3.).unwrap();
        batch.push(face(2), 3.).unwrap();
        batch.push(face(3), 2.).unwrap();
        assert_eq!(batch.nearest().unwrap(), Some(face(3)));
    }
    #[test]
    fn equal_rank_candidates_inside_the_window_choose_the_nearer_hit() {
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 2.05).unwrap();
        batch.push(face(2), 2.).unwrap();
        let ambient = oblikovati_session::selection::AmbientSelectionFilter::default();
        assert_eq!(
            batch
                .resolve(
                    &oblikovati_session::selection::SelectionFilter::All,
                    &ambient,
                    0.1
                )
                .unwrap(),
            Some(face(2))
        );
    }
    #[test]
    fn mixed_categories_decline_until_camera_priority_is_available() {
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 3.).unwrap();
        batch
            .push(PickTarget::Occurrence(vec![(1, 1, 0)]), 3.)
            .unwrap();
        assert!(batch.nearest().unwrap_err().contains("camera-derived"));
    }
    #[test]
    fn plane_hits_respect_category_priority_and_reject_equal_identity_ambiguity() {
        use oblikovati_session::selection::{AmbientSelectionFilter, SelectionFilter};
        let mut batch = PickCandidates::new(PickCoverage::VisiblePlanesAndNearestFaces);
        batch
            .push(
                PickTarget::WorkPlane(oblikovati_model::feature::FeatureId(1)),
                2.,
            )
            .unwrap();
        batch.push(face(2), 2.01).unwrap();
        assert!(matches!(
            batch
                .resolve(
                    &SelectionFilter::All,
                    &AmbientSelectionFilter::default(),
                    0.1
                )
                .unwrap(),
            Some(PickTarget::Face(..))
        ));
        let mut batch = PickCandidates::new(PickCoverage::VisiblePlanesAndNearestFaces);
        batch
            .push(
                PickTarget::WorkPlane(oblikovati_model::feature::FeatureId(1)),
                2.,
            )
            .unwrap();
        batch
            .push(
                PickTarget::WorkPlane(oblikovati_model::feature::FeatureId(2)),
                2.,
            )
            .unwrap();
        assert!(
            batch
                .resolve(
                    &SelectionFilter::All,
                    &AmbientSelectionFilter::default(),
                    0.1
                )
                .is_err()
        );
    }
    #[test]
    fn camera_depth_window_limits_when_rank_can_override_nearness() {
        use oblikovati_session::selection::{AmbientSelectionFilter, SelectionFilter};
        let ambient = AmbientSelectionFilter::default();
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 2.).unwrap();
        batch
            .push(
                PickTarget::Body(oblikovati_model::feature::FeatureId(2), "body".into()),
                2.05,
            )
            .unwrap();
        assert!(matches!(
            batch
                .resolve(
                    &SelectionFilter::All,
                    &ambient,
                    depth_window(10., 800).unwrap()
                )
                .unwrap(),
            Some(PickTarget::Body(..))
        ));
        let mut batch = PickCandidates::new(PickCoverage::NearestFacePerBody);
        batch.push(face(1), 2.).unwrap();
        batch
            .push(
                PickTarget::Body(oblikovati_model::feature::FeatureId(2), "body".into()),
                2.05,
            )
            .unwrap();
        assert_eq!(
            batch
                .resolve(
                    &SelectionFilter::All,
                    &ambient,
                    depth_window(1., 800).unwrap()
                )
                .unwrap(),
            Some(face(1))
        );
        assert_eq!(
            depth_window(10., 1600).unwrap(),
            depth_window(5., 800).unwrap()
        );
    }
}
