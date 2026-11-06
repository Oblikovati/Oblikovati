// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn incompatible_dimension_refuses_both_owners_before_an_edit() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = fresh(owner);
        for point in [1, 5] {
            add(&mut files, &mut source, &request("fix", &[point])).unwrap();
        }
        let before = owned_geometry(&files, document).clone();
        let revision = owner_revision(&files, document);
        let parameters = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .parameters()
            .list()
            .unwrap();
        let mut input = dimension_request("distance", &[1, 5]);
        input.driven = false;
        let error = create_dimension(&mut files, &mut source, &input).unwrap_err();
        assert!(error.to_string().contains("expected Converged"), "{error}");
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(owner_revision(&files, document), revision);
        assert_eq!(
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .parameters()
                .list()
                .unwrap(),
            parameters
        );
        input.driven = true;
        let result = create_dimension(&mut files, &mut source, &input).unwrap();
        assert_eq!(result.parameter, "d1");
    }
}

#[test]
fn dimension_iteration_budget_refuses_without_adopting_a_source_or_relation() {
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
        let mut input = dimension_request("distance", &[1, 5]);
        input.driven = false;
        let error = create_dimension(&mut files, &mut source, &input).unwrap_err();
        assert!(error.to_string().contains("IterationLimit"), "{error}");
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(owner_revision(&files, document), revision);
    }
}
