// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn properties(
    owner: &mut TestOwner,
    index: i64,
    density: f64,
    accuracy: &str,
) -> Result<MassPropertiesResult, oblikovati_api::Error> {
    Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: index,
            density_g_cm3: density,
            accuracy: accuracy.into(),
        })
}

fn model_properties(
    owner: &mut TestOwner,
) -> Result<oblikovati_api::types::PhysicalProperties, oblikovati_api::Error> {
    Client::new(owner.document_host())
        .materials()
        .physical_properties()
}
fn close_value(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() <= expected.abs().max(1.) * 1e-10,
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn body_properties_use_analytic_units_and_preserve_pending_presentation() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let first = properties(&mut owner, 0, 2., "").unwrap();
    close_value(first.volume_mm3, 24000.);
    close_value(first.surface_area_mm2, 5200.);
    close_value(first.mass_g, 48.);
    assert_eq!(first.density_g_cm3, 2.);
    for (actual, expected) in [
        (first.centroid_x_mm, 20.),
        (first.centroid_y_mm, 15.),
        (first.centroid_z_mm, 10.),
        (first.inertia_xx_gmm2, 5200.),
        (first.inertia_yy_gmm2, 8000.),
        (first.inertia_zz_gmm2, 10000.),
    ] {
        close_value(actual, expected);
    }
    for accuracy in ["low", "medium", "high"] {
        assert_eq!(properties(&mut owner, 0, 2., accuracy).unwrap(), first);
    }
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        2
    );
}
#[test]
fn density_defaults_follow_the_part_contract_and_explicit_values_override_it() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, false);
    assert_eq!(properties(&mut owner, 0, 0., "").unwrap().density_g_cm3, 1.);
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: "steel".into(),
        })
        .unwrap();
    let assigned = properties(&mut owner, 0, 0., "").unwrap();
    let steel = Client::new(owner.document_host())
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    close_value(assigned.density_g_cm3, steel.density);
    close_value(assigned.mass_g, 24. * steel.density);
    assert_eq!(
        properties(&mut owner, 0, -3., "").unwrap().density_g_cm3,
        1.
    );
    assert_eq!(properties(&mut owner, 0, 3., "").unwrap().density_g_cm3, 3.);
}

#[test]
fn model_properties_use_current_named_bodies_and_part_density() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let default = model_properties(&mut owner).unwrap();
    close_value(default.volume, 24.);
    close_value(default.area, 52.);
    close_value(default.mass, 24.);
    close_value(default.density, 1.);
    for (actual, expected) in default.centroid.into_iter().zip([2., 1.5, 1.]) {
        close_value(actual, expected);
    }
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: "steel".into(),
        })
        .unwrap();
    let steel = Client::new(owner.document_host())
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    let assigned = model_properties(&mut owner).unwrap();
    close_value(assigned.density, steel.density);
    close_value(assigned.mass, 24. * steel.density);
    owner.files.commands(document).unwrap().undo().unwrap();
    close_value(model_properties(&mut owner).unwrap().mass, 24.);
}
#[test]
fn queries_reject_bad_indices_accuracy_and_overflow_then_track_replay() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    for index in [-1, 2, i64::MAX] {
        assert!(properties(&mut owner, index, 1., "").is_err());
    }
    assert!(properties(&mut owner, 0, 1., "finest").is_err());
    assert!(properties(&mut owner, 0, f64::MAX, "").is_err());
    set_parameter(&mut owner, "depth", "3 cm");
    close_value(
        properties(&mut owner, 0, 1., "").unwrap().volume_mm3,
        36000.,
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    close_value(
        properties(&mut owner, 0, 1., "").unwrap().volume_mm3,
        24000.,
    );
    set_parameter(&mut owner, "depth", "0 cm");
    close_value(
        properties(&mut owner, 0, 1., "").unwrap().volume_mm3,
        24000.,
    );
    assert!(properties(&mut owner, 1, 1., "").is_err());
}
#[test]
fn missing_selected_material_is_an_error_unless_density_is_explicit() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Missing material fixture", |snapshot, _| {
            snapshot.assets.assignments.set_part_material("missing")
        })
        .unwrap();
    assert!(
        properties(&mut owner, 0, 0., "")
            .unwrap_err()
            .to_string()
            .contains("missing")
    );
    close_value(properties(&mut owner, 0, 2., "").unwrap().mass_g, 48.);
    let mut guids = OwnerThreadGuids::default();
    let host = oblikovati_automation::FileHost::new(&mut owner.files, &mut guids, "test");
    assert!(
        Client::new(host)
            .body()
            .physical_properties(&BodyPhysicalPropertiesArgs::default())
            .unwrap_err()
            .to_string()
            .contains("injected exact topology owner")
    );
}
