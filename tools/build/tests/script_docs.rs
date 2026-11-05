// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};
use oblikovati_script::help::MethodParameter;
const WIRE: &str = r#"package wire
const MethodCreate = "documents.create"
const MethodList = "documents.list"
const Unrelated = 3
type CreateArgs struct { Kind string `json:"type"`; Hidden bool `json:"-"`; Defaults int; }
"#;
const CLIENT: &str = r#"package client
// mcp:summary Create a document.
func (d Documents) Create(args wire.CreateArgs) error { return call[Reply](d.c, wire.MethodCreate, args) }
// mcp:summary List documents.
func (d Documents) List() error { return d.c.call[Reply](wire.MethodList, nil) }
"#;
pub(crate) struct HelpSourceFixture {
    pub(crate) api: FixtureTree,
}
impl HelpSourceFixture {
    pub(crate) fn new(root: &FixtureTree) -> Self {
        let fixture = Self {
            api: FixtureTree::new(),
        };
        fixture.api.put("go/wire/args.go", WIRE);
        fixture.api.put("go/client/client.go", CLIENT);
        root.put("release.toml", include_str!("../../../release.toml"));
        let pin = release_config::load(root.root()).unwrap().api;
        let snapshot = collect(&fixture.api.root().join("go"), &pin).unwrap();
        root.put(CATALOG, &(serde_json::to_string(&snapshot).unwrap() + "\n"));
        fixture
    }
    pub(crate) fn metadata(&self) -> String {
        let pin = release_config::load(
            Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../..")
                .as_path(),
        )
        .unwrap()
        .api;
        serde_json::json!({"packages":[{"name":"oblikovati-api","version":pin.version,
            "source":format!("git+https://github.com/{}?rev={}#{}",pin.repository,pin.revision,pin.revision),
            "manifest_path":self.api.root().join("rust/Cargo.toml")}]}).to_string()
    }
}
fn help_runner(fixture: &HelpSourceFixture) -> FakeRunner {
    let runner = FakeRunner::default();
    runner
        .replies
        .borrow_mut()
        .push_back(Ok(fixture.metadata()));
    runner
}
#[test]
fn help_generation_is_stable_and_stale_catalog_is_refused() {
    let root = FixtureTree::new();
    let fixture = HelpSourceFixture::new(&root);
    crate::dispatch(&["script-docs-check"], root.root(), &help_runner(&fixture)).unwrap();
    root.put(CATALOG, "stale");
    assert!(
        check(root.root(), &help_runner(&fixture))
            .unwrap_err()
            .contains("make script-docs")
    );
    crate::dispatch(&["script-docs"], root.root(), &help_runner(&fixture)).unwrap();
    crate::dispatch(&["script-docs-check"], root.root(), &help_runner(&fixture)).unwrap();
    verify_fixture_snapshot(&root);
}
#[test]
fn help_sources_require_released_owner_package_and_source_shape() {
    let root = FixtureTree::new();
    let fixture = HelpSourceFixture::new(&root);
    for metadata in [
        "invalid",
        r#"{"packages":[]}"#,
        &fixture.metadata().replace("0.158.0", "9.0.0"),
    ] {
        let runner = FakeRunner::default();
        runner.replies.borrow_mut().push_back(Ok(metadata.into()));
        assert!(generated(root.root(), &runner).is_err());
    }
    fixture.api.put("go/client/client.go", "package client");
    assert!(
        generated(root.root(), &help_runner(&fixture))
            .unwrap_err()
            .contains("expected scriptable")
    );
}
#[test]
fn wire_fields_retain_json_order_types_and_skip_tests_and_embedded_fields() {
    let root = FixtureTree::new();
    let fixture = HelpSourceFixture::new(&root);
    fixture.api.put("go/wire/bad_test.go", "not valid Go");
    fixture.api.put("go/wire/options.go", r#"package wire
const (
Label, Other = `literal`, "other"
)
type NotStruct []string
type Options struct { Embedded; A,B int; Value *types.Value `xml:"x" json:"value,omitempty"`; Omit string `json:"-"`; Items []string `json:",omitempty"`; }
"#);
    let help = read_fixture_wire(&fixture);
    verify_options_fields(&help);
}
#[test]
fn overloads_merge_canonical_fields_preserve_summary_and_reject_type_conflicts() {
    let next = overload_help();
    let mut previous = next.clone();
    previous.summary.clear();
    previous.parameters.clear();
    merge_help(&mut previous, next.clone()).unwrap();
    merge_help(&mut previous, next.clone()).unwrap();
    assert_eq!(previous.parameters, next.parameters);
    let mut conflict = next;
    conflict.parameters[0].type_name = "string".into();
    assert!(
        merge_help(&mut previous, conflict)
            .unwrap_err()
            .contains("expected uint64")
    );
}
#[test]
fn missing_wire_constants_and_request_definitions_decline_by_name() {
    let mut methods = BTreeMap::new();
    let mut wire = go::WireHelp::default();
    assert!(
        insert_method(&mut methods, &wire, missing_client())
            .unwrap_err()
            .contains("MethodMissing")
    );
    wire.constants
        .insert("MethodMissing".into(), "missing.method".into());
    assert!(
        insert_method(&mut methods, &wire, missing_client())
            .unwrap_err()
            .contains("MissingArgs")
    );
}

fn verify_fixture_snapshot(root: &FixtureTree) {
    let snapshot: MethodHelpSnapshot =
        serde_json::from_str(&read_text(&root.root().join(CATALOG)).unwrap()).unwrap();
    assert_eq!(snapshot.methods.len(), 2);
    assert_eq!(
        snapshot.methods[0].signature(),
        "documents.create{ type, Defaults }"
    );
    assert_eq!(snapshot.methods[1].summary, "List documents.");
}

fn read_fixture_wire(fixture: &HelpSourceFixture) -> go::WireHelp {
    let mut help = go::WireHelp::default();
    for file in go_files(&fixture.api.root().join("go/wire")).unwrap() {
        go::parse_wire(&read_text(&file).unwrap(), &mut help).unwrap();
    }
    help
}

fn overload_help() -> MethodHelp {
    MethodHelp {
        wire: "test.method".into(),
        summary: "Canonical summary.".into(),
        parameters: vec![MethodParameter {
            name: "id".into(),
            type_name: "uint64".into(),
        }],
    }
}

fn missing_client() -> go::ClientHelp {
    go::ClientHelp {
        constant: "MethodMissing".into(),
        request: Some("MissingArgs".into()),
        summary: String::new(),
    }
}

fn verify_options_fields(help: &go::WireHelp) {
    assert_eq!(help.constants["Label"], "literal");
    assert_eq!(
        help.requests["Options"]
            .iter()
            .map(|field| field.name.as_str())
            .collect::<Vec<_>>(),
        ["A", "B", "value", "Items"]
    );
    assert_eq!(help.requests["Options"][2].type_name, "*types.Value");
}
