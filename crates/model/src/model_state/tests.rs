// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn primary_create_rename_activate_copy_preserves_identity_and_selection() {
    let mut members = ModelStateMembers::default();
    assert_eq!(members.list().len(), 1);
    assert_eq!(members.active().unwrap().id, 1);
    members.insert(2, "Variant").unwrap();
    assert_eq!(members.active().unwrap().id, 1);
    members.rename(2, "Large").unwrap();
    members.activate(2).unwrap();
    members.copy(2, 3, "Large Copy").unwrap();
    assert_eq!(members.active().unwrap().name, "Large");
    assert_eq!(members.list()[2].id, 3);
    members.validate().unwrap();
}
#[test]
fn invalid_member_edits_do_not_change_content() {
    let mut members = ModelStateMembers::default();
    let before = members.clone();
    assert!(members.rename(1, "Other").is_err());
    assert!(members.insert(0, "Other").is_err());
    assert!(members.insert(2, "Primary").is_err());
    assert!(members.insert(2, " ").is_err());
    assert!(members.activate(99).is_err());
    assert!(members.copy(99, 2, "Copy").is_err());
    assert!(members.rename(99, "Other").is_err());
    assert_eq!(members, before);
}
#[test]
fn names_with_nul_are_rejected_without_changing_members() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Variant").unwrap();
    let before = members.clone();
    for result in [
        members.insert(3, "bad\0name"),
        members.rename(2, "bad\0name"),
        members.copy(2, 3, "bad\0name"),
    ] {
        assert!(result.unwrap_err().contains("bad\\0name"));
    }
    assert_eq!(members, before);
    let malformed = ModelStateMembers {
        members: vec![
            before.members[0].clone(),
            ModelStateMember {
                id: 2,
                name: "bad\0name".into(),
            },
        ],
        active: 1,
        ..ModelStateMembers::default()
    };
    assert!(malformed.validate().is_err());
}
#[test]
fn restored_member_invariants_are_checked() {
    let valid = ModelStateMembers::default();
    assert!(valid.validate().is_ok());
    let invalid = ModelStateMembers {
        members: vec![],
        active: 1,
        ..ModelStateMembers::default()
    };
    assert!(invalid.validate().is_err());
    let invalid = ModelStateMembers {
        active: 99,
        ..valid
    };
    assert!(invalid.validate().is_err());
}
#[test]
fn member_delete_protects_primary_and_restores_active_selection() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Model State1").unwrap();
    assert_eq!(members.generated_name("Model State"), "Model State2");
    members.activate(2).unwrap();
    assert!(members.delete(1).is_err());
    assert!(members.delete(99).is_err());
    members.delete(2).unwrap();
    assert_eq!(members, ModelStateMembers::default());
}

#[test]
fn suppression_follows_scope_and_copies_the_chosen_member() {
    let mut states = ModelStateMembers::default();
    states.insert(2, "Variant").unwrap();
    states.set_suppressed(42, false, true);
    states.copy(1, 3, "Copy").unwrap();
    states.activate(2).unwrap();
    assert!(!states.suppressed(42, false));
    states.register_feature(43);
    states.activate(1).unwrap();
    assert!(states.suppressed(42, false));
    assert!(states.suppressed(43, false));
    states.activate(3).unwrap();
    assert!(states.suppressed(42, false));
    states.set_edit_scope(ModelStateEditScope::AllMembers);
    states.set_suppressed(42, false, false);
    for member in [1, 2, 3] {
        states.activate(member).unwrap();
        assert!(!states.suppressed(42, false));
    }
    states.forget_feature(43);
    assert!(!states.referenced_features().any(|id| id == 43));
    states.validate().unwrap();
}
