// SPDX-License-Identifier: GPL-2.0-only
//! Invariants for copied and active part design views (#4118).
use super::*;
use oblikovati_api::contract::PartDesignView as _;
use oblikovati_api::types::{PART_ANNOTATION_SCALE_CUSTOM, PART_ANNOTATION_SCALE_FIXED};

#[test]
fn built_in_views_have_stable_order_and_one_primary_active() {
    let views = PartDesignViews::default();
    assert_eq!(
        views
            .list()
            .iter()
            .map(PartDesignView::name)
            .collect::<Vec<_>>(),
        ["Primary", "Isometric", "Front", "Top", "Right"]
    );
    assert_eq!(views.active().unwrap().id(), 1);
    assert_eq!(
        views.get(3).unwrap().standard_orientation(),
        oblikovati_api::types::FRONT_VIEW_ORIENTATION
    );
    assert_eq!(
        views.get(4).unwrap().standard_orientation(),
        oblikovati_api::types::TOP_VIEW_ORIENTATION
    );
    assert!(views.validate().is_ok());
}

#[test]
fn copy_rename_activate_and_delete_preserve_identity() {
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "View1").unwrap();
    assert_eq!(views.active().unwrap().id(), 1);
    views.rename(6, "Capture View").unwrap();
    views.activate(6).unwrap();
    assert_eq!(views.active().unwrap().name(), "Capture View");
    assert_eq!(views.get(6).unwrap().id(), 6);
    views.delete(6).unwrap();
    assert_eq!(views.active().unwrap().id(), 1);
    assert!(views.validate().is_ok());
}

#[test]
fn invalid_identity_or_name_never_changes_the_collection() {
    let mut views = PartDesignViews::default();
    let before = views.clone();
    assert!(views.copy(1, 1, "Copy").is_err());
    assert!(views.copy(1, 6, "Front").is_err());
    assert!(views.copy(99, 6, "Copy").is_err());
    assert!(views.rename(1, "Other").is_err());
    assert!(views.activate(99).is_err());
    assert!(views.delete(1).is_err());
    assert!(views.delete(4).is_err());
    assert_eq!(views, before);
    views.copy(1, 6, "front").unwrap();
    assert!(views.validate().is_ok());
}

#[test]
fn view_lock_and_scale_reject_invalid_changes() {
    let mut views = PartDesignViews::default();
    views.get_mut(1).unwrap().set_locked(true);
    assert!(views.get(1).unwrap().locked());
    views.get_mut(1).unwrap().set_locked(false);
    views.copy(1, 6, "Review").unwrap();
    let scale = PartAnnotationScale {
        mode: PART_ANNOTATION_SCALE_FIXED,
        numerator: 1,
        denominator: 1,
    };
    assert!(
        views
            .get_mut(6)
            .unwrap()
            .set_annotation_scale(scale.clone())
            .is_err()
    );
    views.activate(6).unwrap();
    views
        .get_mut(6)
        .unwrap()
        .set_annotation_scale(scale.clone())
        .unwrap();
    let invalid = PartAnnotationScale {
        mode: PART_ANNOTATION_SCALE_CUSTOM,
        numerator: 1,
        denominator: 0,
    };
    assert!(
        views
            .get_mut(6)
            .unwrap()
            .set_annotation_scale(invalid)
            .is_err()
    );
    assert_eq!(views.active().unwrap().annotation_scale(), scale);
}

#[test]
fn copied_view_retains_options_but_not_active_state() {
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "First").unwrap();
    views.get_mut(6).unwrap().set_published(false);
    views.get_mut(6).unwrap().set_autosave_camera(false);
    views.get_mut(6).unwrap().set_locked(true);
    views.copy(6, 7, "Second").unwrap();
    let copy = views.get(7).unwrap();
    assert!(!copy.active());
    assert!(copy.locked());
    assert!(!copy.published());
    assert!(!copy.autosave_camera());
}

#[test]
fn options_commit_together_or_not_at_all() {
    use oblikovati_api::wire::SetPartDesignViewOptionsArgs;
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "Review").unwrap();
    let before = views.clone();
    let options = SetPartDesignViewOptionsArgs {
        id: 6,
        published: Some(Box::new(false)),
        annotation_scale: Some(Box::new(PartAnnotationScale {
            mode: PART_ANNOTATION_SCALE_CUSTOM,
            numerator: 3,
            denominator: 2,
        })),
        ..Default::default()
    };
    assert!(views.set_options(&options).is_err());
    assert_eq!(views, before);
    views.activate(6).unwrap();
    views.set_options(&options).unwrap();
    assert!(!views.active().unwrap().published());
    assert_eq!(views.active().unwrap().annotation_scale().numerator, 3);
}

#[test]
fn generated_names_skip_existing_views() {
    let mut views = PartDesignViews::default();
    assert_eq!(views.generated_name(), "View1");
    views.copy(1, 6, "View1").unwrap();
    assert_eq!(views.generated_name(), "View2");
}

#[test]
fn camera_is_validated_and_cannot_change_in_locked_view() {
    use oblikovati_api::types::{ORTHOGRAPHIC_PROJECTION, Point, Vector};
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "Review").unwrap();
    assert_eq!(views.get(6).unwrap().camera(), None);
    assert!(views.get(6).unwrap().hidden_body_keys().is_empty());
    let camera = PartDesignViewCamera {
        eye: Point {
            x: 0.,
            y: 0.,
            z: 3.,
        },
        target: Point::default(),
        up: Vector {
            x: 0.,
            y: 1.,
            z: 0.,
        },
        scale: 12.,
        near: 0.5,
        far: 250.,
        projection: ORTHOGRAPHIC_PROJECTION,
    };
    views.capture_camera(6, camera.clone()).unwrap();
    let encoded = serde_json::to_vec(&views).unwrap();
    let restored: PartDesignViews = serde_json::from_slice(&encoded).unwrap();
    assert_eq!(restored.get(6).unwrap().camera(), Some(&camera));
    views.get_mut(6).unwrap().set_locked(true);
    let invalid = PartDesignViewCamera {
        eye: Point::default(),
        ..camera.clone()
    };
    assert!(views.capture_camera(6, invalid).is_err());
    assert_eq!(views.get(6).unwrap().camera(), Some(&camera));
}

#[test]
fn serialized_views_keep_identity_and_reject_corrupt_active_state() {
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "Review").unwrap();
    views.activate(6).unwrap();
    let encoded = serde_json::to_vec(&views).unwrap();
    let decoded: PartDesignViews = serde_json::from_slice(&encoded).unwrap();
    decoded.validate().unwrap();
    assert_eq!(decoded, views);
    let mut corrupt = decoded;
    corrupt.views[0].active = true;
    assert!(corrupt.validate().is_err());
    let mut missing_preset = views;
    missing_preset.views.retain(|view| view.id != 4);
    assert!(missing_preset.validate().is_err());
}

#[test]
fn body_visibility_checks_scope_and_preserves_stored_state_when_locked() {
    let mut views = PartDesignViews::default();
    views.copy(1, 6, "Review").unwrap();
    views.activate(6).unwrap();
    let known = BTreeSet::from([String::from("aa"), String::from("bb")]);
    views.set_body_visibility(6, &[], &known, false).unwrap();
    assert_eq!(views.get(6).unwrap().hidden_body_keys(), &known);
    let before = views.clone();
    assert!(
        views
            .set_body_visibility(6, &["cc".into()], &known, true)
            .is_err()
    );
    assert!(
        views
            .set_body_visibility(6, &["aa".into(), "aa".into()], &known, true)
            .is_err()
    );
    assert_eq!(views, before);
    views.get_mut(6).unwrap().set_locked(true);
    assert!(
        views
            .set_body_visibility(6, &["aa".into()], &known, true)
            .is_err()
    );
    assert_eq!(views.get(6).unwrap().hidden_body_keys(), &known);
}
