// SPDX-License-Identifier: GPL-2.0-only
//! Straight-hole authored inputs, replay dependencies, and recipe validation (#4093).
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        FeatureId, FeatureReadKey,
        hole::{HoleDefinition, HoleExtent, HolePosition, HoleRecipe},
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn document() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}
fn key(document: DocumentGuid, feature: u64, kind: u32) -> Vec<u8> {
    SemanticReferenceName::new(document, feature, kind, "fixture", "reference", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn recipe() -> HoleRecipe {
    HoleRecipe {
        version: 1,
        body: key(document(), 1, 4),
        face: key(document(), 2, 1),
        position: HolePosition::ModelPoint([
            DatumLength::Parameter(4),
            DatumLength::Constant(-2.),
            DatumLength::Constant(0.),
        ]),
        diameter: DatumLength::Parameter(3),
        extent: HoleExtent::Distance(DatumLength::Parameter(5)),
        options: Default::default(),
    }
}
fn parameters() -> ParameterCollection<u64> {
    let mut values = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name, value) in [(3, "diameter", 2.), (4, "x", 1.), (5, "depth", 4.)] {
        values
            .insert_value(
                id,
                name,
                USER_PARAM,
                Quantity::new(value, Dimension::Length),
            )
            .unwrap();
    }
    values
}

#[test]
fn hole_recipe_round_trips_and_tracks_all_inputs() {
    let hole = HoleDefinition::from_recipe(recipe()).unwrap();
    let encoded = serde_json::to_string(&hole.recipe()).unwrap();
    let decoded: HoleRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(HoleDefinition::from_recipe(decoded).unwrap(), hole);
    for key in [
        FeatureReadKey::Feature(FeatureId(1)),
        FeatureReadKey::Feature(FeatureId(2)),
        FeatureReadKey::Parameter(3),
        FeatureReadKey::Parameter(4),
        FeatureReadKey::Parameter(5),
    ] {
        assert!(
            hole.reads().keys().contains(&key),
            "missing dependency {key:?}"
        );
    }
    assert!(!hole.reads().has_unattributed_reads());
    assert_eq!(hole.body(), recipe().body);
    assert_eq!(hole.face(), recipe().face);
    assert_eq!(hole.body_source(), FeatureId(1));
}

#[test]
fn hole_resolves_current_parameter_values_and_through_extent() {
    let hole = HoleDefinition::from_recipe(recipe()).unwrap();
    let mut values = parameters();
    let initial = hole.resolve(&values).unwrap();
    assert_eq!(initial.position, Some([1., -2., 0.]));
    assert_eq!((initial.diameter, initial.depth), (2., Some(4.)));
    values
        .set_value(3, Quantity::new(3., Dimension::Length))
        .unwrap();
    assert_eq!(hole.resolve(&values).unwrap().diameter, 3.);
    let mut through = recipe();
    through.extent = HoleExtent::ThroughAll;
    through.position = HolePosition::FaceCentroid;
    let resolved = HoleDefinition::from_recipe(through)
        .unwrap()
        .resolve(&values)
        .unwrap();
    assert_eq!((resolved.position, resolved.depth), (None, None));
}

#[test]
fn hole_rejects_invalid_references_lengths_and_recipe_versions() {
    let mut cases = Vec::new();
    let mut bad = recipe();
    bad.version = 2;
    cases.push(bad);
    let mut bad = recipe();
    bad.body = key(document(), 1, 1);
    cases.push(bad);
    let mut bad = recipe();
    bad.face = key(document(), 2, 4);
    cases.push(bad);
    let mut bad = recipe();
    bad.face = key(
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
        2,
        1,
    );
    cases.push(bad);
    let mut bad = recipe();
    bad.diameter = DatumLength::Constant(0.);
    cases.push(bad);
    let mut bad = recipe();
    bad.extent = HoleExtent::Distance(DatumLength::Constant(-1.));
    cases.push(bad);
    let mut bad = recipe();
    bad.position = HolePosition::ModelPoint([DatumLength::Constant(f64::NAN); 3]);
    cases.push(bad);
    let mut bad = recipe();
    bad.diameter = DatumLength::Parameter(0);
    cases.push(bad);
    for candidate in cases {
        assert!(HoleDefinition::from_recipe(candidate).is_err());
    }
}

#[test]
fn hole_refuses_invalid_parameter_values_and_unknown_recipe_fields() {
    let hole = HoleDefinition::from_recipe(recipe()).unwrap();
    let mut values = parameters();
    values
        .set_value(3, Quantity::new(-1., Dimension::Length))
        .unwrap();
    assert!(hole.resolve(&values).is_err());
    let mut encoded = serde_json::to_value(recipe()).unwrap();
    encoded
        .as_object_mut()
        .unwrap()
        .insert("unexpected".into(), true.into());
    assert!(serde_json::from_value::<HoleRecipe>(encoded).is_err());
}

#[test]
fn hole_recipe_comparison_preserves_coordinate_signed_zero() {
    let before = HoleDefinition::from_recipe(recipe()).unwrap();
    let mut changed = recipe();
    changed.position = HolePosition::ModelPoint([
        DatumLength::Parameter(4),
        DatumLength::Constant(-2.),
        DatumLength::Constant(-0.),
    ]);
    let after = HoleDefinition::from_recipe(changed).unwrap();
    assert!(!before.same_recipe(&after));
    assert!(before.same_recipe(&HoleDefinition::from_recipe(before.recipe()).unwrap()));
}

#[test]
fn hole_v2_options_validate_seats_taper_tapped_and_to_surface_round_trip() {
    use oblikovati_model::feature::hole::{
        HoleDrillPoint, HoleFastenerFit, HoleFastenerSpec, HoleKind, HoleOptions, HoleOriginPlane,
        HoleSeat, HoleTerminationSurface, HoleThreadSpec, ResolvedHoleSeat,
    };
    let mut c_sink = recipe();
    c_sink.version = 2;
    c_sink.extent = HoleExtent::ToSurface(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy));
    c_sink.options = HoleOptions {
        kind: HoleKind::Tapped,
        seat: HoleSeat::Countersink {
            diameter: DatumLength::Constant(6.),
            angle_deg: 90.,
        },
        drill_point: HoleDrillPoint::Angle(118.),
        direction_flipped: true,
        extend_start: true,
        fastener: None,
        thread: Some(HoleThreadSpec {
            profile: "ISO Metric profile".into(),
            designation: "M4x0.7".into(),
            full_depth: true,
            left_hand: true,
        }),
    };
    let def = HoleDefinition::from_recipe(c_sink.clone()).unwrap();
    let encoded = serde_json::to_string(&def.recipe()).unwrap();
    let decoded: HoleRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(HoleDefinition::from_recipe(decoded).unwrap(), def);
    let resolved = def.resolve(&parameters()).unwrap();
    assert_eq!(
        resolved.termination_surface,
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy))
    );
    assert_eq!(
        resolved.seat,
        ResolvedHoleSeat::Countersink {
            diameter: 6.,
            angle_deg: 90.
        }
    );
    assert!(resolved.direction_flipped && resolved.extend_start);

    // Invalid stepped seat: seat diameter equal to bore diameter must be refused.
    let mut bad_seat = recipe();
    bad_seat.version = 2;
    bad_seat.diameter = DatumLength::Constant(4.);
    bad_seat.options.seat = HoleSeat::Counterbore {
        diameter: DatumLength::Constant(4.),
        depth: DatumLength::Constant(2.),
    };
    assert!(HoleDefinition::from_recipe(bad_seat).is_err());

    // Invalid taper-tapped hole with conical seat must be refused until seat is removed.
    let mut bad_taper = c_sink;
    bad_taper.options.kind = HoleKind::TaperTapped;
    assert!(HoleDefinition::from_recipe(bad_taper.clone()).is_err());
    bad_taper.options.seat = HoleSeat::None;
    assert!(HoleDefinition::from_recipe(bad_taper).is_ok());

    // Fastener clearance hole round-trips cleanly.
    let mut clearance = recipe();
    clearance.version = 2;
    clearance.options.kind = HoleKind::Clearance;
    clearance.options.fastener = Some(HoleFastenerSpec {
        standard: "ANSI Unified Screw Threads".into(),
        fastener_type: "Hex Head Bolt".into(),
        size: "#4".into(),
        fit: HoleFastenerFit::Loose,
    });
    assert!(HoleDefinition::from_recipe(clearance).is_ok());
}
