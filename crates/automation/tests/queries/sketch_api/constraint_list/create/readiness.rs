// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn conflicting_fixed_line_constructor_refuses_both_document_owners_without_an_edit() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = fresh(owner);
        for point in [1, 2] {
            add(&mut files, &mut source, &request("fix", &[point])).unwrap();
        }
        let before = owned_geometry(&files, document).clone();
        let revision = owner_revision(&files, document);
        let error = add(&mut files, &mut source, &request("vertical", &[7])).unwrap_err();
        assert!(error.to_string().contains("expected Converged"), "{error}");
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(owner_revision(&files, document), revision);
        add(&mut files, &mut source, &request("horizontal", &[7])).unwrap();
        assert_eq!(
            owned_geometry(&files, document).sketches()[0]
                .geometry()
                .constraints()
                .last()
                .unwrap()
                .id,
            SketchConstraintId(4)
        );
    }
}
#[test]
fn iteration_limit_refuses_creation_instead_of_returning_only_a_dof_count() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = configured_files(
            owner,
            SolveOptions {
                max_iterations: 0,
                ..Default::default()
            },
        );
        files
            .commands(document)
            .unwrap()
            .run("Operands", install_operands)
            .unwrap();
        let before = owned_geometry(&files, document).clone();
        let revision = owner_revision(&files, document);
        let error = add(&mut files, &mut source, &request("coincident", &[2, 4])).unwrap_err();
        assert!(error.to_string().contains("IterationLimit"), "{error}");
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(owner_revision(&files, document), revision);
    }
}

#[test]
fn earlier_parallel_seed_is_not_reported_as_success_when_its_solver_stalls() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = fresh(owner);
        let before = owned_geometry(&files, document).clone();
        let revision = owner_revision(&files, document);
        let error = add(&mut files, &mut source, &request("perpendicular", &[7, 8])).unwrap_err();
        assert!(error.to_string().contains("expected Converged"), "{error}");
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(owner_revision(&files, document), revision);
    }
}
