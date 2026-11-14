// SPDX-License-Identifier: GPL-2.0-only
use super::*;

struct ExactCapVerifier<'a> {
    named: &'a oblikovati_martensite::NamedPrismSnapshot,
    history: &'a DocumentEditHistory,
}
impl oblikovati_persistence::package::LegacyCapBindingVerifier for ExactCapVerifier<'_> {
    fn verify(
        &self,
        key: &[u8],
        role: oblikovati_persistence::package::LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        let actual = self
            .named
            .face_report(self.history, document_guid(), key)?
            .role;
        let expected = match role {
            oblikovati_persistence::package::LegacyExtrusionFaceRole::StartCap => {
                PrismFaceRole::Start
            }
            oblikovati_persistence::package::LegacyExtrusionFaceRole::EndCap => PrismFaceRole::End,
            other => return Err(format!("role {other:?}: expected cap")),
        };
        if actual != expected {
            return Err(format!("role {actual:?}: expected {expected:?}"));
        }
        Ok(())
    }
}

fn verify_migrated_caps(history: &DocumentEditHistory) {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let named = retain_certified_named_sketch_prism(
        history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let verifier = ExactCapVerifier {
        named: &named,
        history,
    };
    for (source, expected) in [
        (
            b"\x03Extrusion1:start-cap#0".as_slice(),
            PrismFaceRole::Start,
        ),
        (b"\x03Extrusion1:end-cap#0".as_slice(), PrismFaceRole::End),
    ] {
        let mapped = candidate
            .migrate_cap_face(source, document_guid(), feature, &verifier)
            .unwrap();
        assert_eq!(
            named
                .face_report(history, document_guid(), mapped.key())
                .unwrap()
                .role,
            expected
        );
    }
    assert!(
        candidate
            .migrate_cap_face(
                b"\x03Extrusion1:side#1",
                document_guid(),
                feature,
                &verifier
            )
            .is_err()
    );
}

#[test]
fn go_cap_names_migrate_through_edit_undo_redo_and_reopen() {
    let mut history = imported_history(GO_CUBE);
    verify_migrated_caps(&history);
    let token = history.begin("size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let id = snapshot.parameters.find("size").unwrap();
            snapshot.parameters.set_expression(id, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    verify_migrated_caps(&history);
    history.undo().unwrap();
    verify_migrated_caps(&history);
    history.redo().unwrap();
    verify_migrated_caps(&history);
    verify_migrated_caps(&reopened_history(&history));
}

struct RejectCapVerifier;
impl oblikovati_persistence::package::LegacyCapBindingVerifier for RejectCapVerifier {
    fn verify(
        &self,
        _: &[u8],
        _: oblikovati_persistence::package::LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("named face was not found".into())
    }
}

#[test]
fn cap_migration_refuses_unverified_wrong_target_and_wrong_document() {
    let history = imported_history(GO_CUBE);
    let candidate = DocumentPackage::decode(GO_CUBE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let key = b"\x03Extrusion1:start-cap#0";
    let error = candidate
        .migrate_cap_face(key, document_guid(), feature, &RejectCapVerifier)
        .unwrap_err();
    assert!(error.to_string().contains("named face was not found"));
    let named = retain_certified_named_sketch_prism(
        &history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let verifier = ExactCapVerifier {
        named: &named,
        history: &history,
    };
    let foreign = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    assert!(
        candidate
            .migrate_cap_face(key, foreign, feature, &verifier)
            .is_err()
    );
    let wrong = AuthoredFeature::sketch_prism(
        FeatureId(9),
        "Other",
        candidate.extrusions[0].definition.clone(),
        history.snapshot(),
    )
    .unwrap();
    assert!(
        candidate
            .migrate_cap_face(key, document_guid(), &wrong, &verifier)
            .is_err()
    );
}

#[test]
fn cap_migration_refuses_ambiguous_source_sequences() {
    use oblikovati_persistence::package::LegacyExtrusionCandidate;
    let history = imported_history(GO_CUBE);
    let mut candidate = DocumentPackage::decode(GO_CUBE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let source = &candidate.extrusions[0];
    candidate.extrusions.push(LegacyExtrusionCandidate {
        name: "Another extrusion".into(),
        creation_sequence: source.creation_sequence,
        suppressed: false,
        definition: source.definition.clone(),
    });
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let error = candidate
        .migrate_cap_face(
            b"\x03Extrusion1:start-cap#0",
            document_guid(),
            feature,
            &RejectCapVerifier,
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("one nonzero source sequence"),
        "{error}"
    );
}
