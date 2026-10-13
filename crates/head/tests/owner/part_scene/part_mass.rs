// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn total(owner: &mut TestOwner, density: f64) -> MassPropertiesResult {
    Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs {
            density_g_cm3: density,
            accuracy: String::new(),
        })
        .unwrap()
}
fn one(owner: &mut TestOwner) -> MassPropertiesResult {
    Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs::default())
        .unwrap()
}
#[test]
fn one_body_analysis_is_identical_and_empty_part_is_zero_with_the_requested_density() {
    let (mut owner, awake) = owner();
    let empty = create(&mut owner, &awake);
    let result = total(&mut owner, 2.);
    assert_eq!(
        result,
        MassPropertiesResult {
            density_g_cm3: 2.,
            principal_axes: [[1., 0., 0.], [0., 1., 0.], [0., 0., 1.]],
            ..Default::default()
        }
    );
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: empty,
            force: true,
        })
        .unwrap();
    open_prism(&mut owner, false);
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
#[test]
fn analysis_sums_current_bodies_without_union_and_tracks_edit_undo_and_failure() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let both = total(&mut owner, 2.);
    assert!((both.volume_mm3 - 48000.).abs() < 1e-8);
    assert!((both.mass_g - 96.).abs() < 1e-10);
    assert!((both.inertia_yy_gmm2 - 16000.).abs() < 1e-8);
    set_parameter(&mut owner, "depth", "3 cm");
    assert!((total(&mut owner, 2.).volume_mm3 - 60000.).abs() < 1e-8);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(total(&mut owner, 2.), both);
    set_parameter(&mut owner, "depth", "0 cm");
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
#[test]
fn part_analysis_uses_assigned_density_and_validates_requests() {
    let (mut owner, _) = owner();
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs::default())
            .is_err()
    );
    open_prism(&mut owner, false);
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: "steel".into(),
        })
        .unwrap();
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs {
                accuracy: "unknown".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs {
                density_g_cm3: f64::MAX,
                ..Default::default()
            })
            .is_err()
    );
}
