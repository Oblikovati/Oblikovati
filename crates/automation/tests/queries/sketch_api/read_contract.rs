// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn typed_query_reads_structure_without_changing_geometry_history_or_saved_state() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    assert_circle_structure(&status_call(&files, 0).unwrap());
    assert_eq!(before, owner_revision(&files, document));
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_SOLVE));
    assert!(FILE_QUERY_METHODS.contains(&METHOD_SKETCH_CONSTRAINT_STATUS));
}

#[test]
fn typed_solve_distinguishes_convergence_freedom_and_redundancy_health() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let well = solve_call(&mut files, &mut source, 0).unwrap();
    assert_eq!(well.sketch_index, 0);
    assert_wire_solve(&well, "well", 0, true, true);
    for (redundant, status, dof, healthy) in [(false, "under", 2, true), (true, "over", 0, false)] {
        replace_with_point(&mut files, document, sketch, redundant);
        let solved = solve_call(&mut files, &mut source, 0).unwrap();
        assert_wire_solve(&solved, status, dof, true, healthy);
        let structure = status_call(&files, 0).unwrap();
        assert_eq!((structure.status.as_str(), structure.dof), (status, dof));
        assert_eq!(structure.redundant, if redundant { 2 } else { 0 });
    }
}

#[test]
fn invalid_indices_fail_before_any_owner_command() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(status_call(&files, index), &index.to_string());
        expect_api_decline(
            solve_call(&mut files, &mut source, index),
            &index.to_string(),
        );
        assert_eq!(before, owner_revision(&files, document));
    }
}
