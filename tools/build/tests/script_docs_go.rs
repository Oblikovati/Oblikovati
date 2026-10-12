// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn client_request_forms_and_summaries_are_parsed_from_real_syntax() {
    let text = r#"package client
// mcp:summary First method.
func (c Client) One() error { unrelated(); return c.call(wire.MethodOne, &wire.OneArgs{}) }
func (c Client) Two() error { req := wire.TwoArgs{}; return call[R](c, wire.MethodTwo, req) }
func (c Client) Three() error { var req wire.ThreeArgs; return call[R](c, wire.MethodThree, req) }
func (c Client) Four() error { return call[R](c, wire.MethodFour, struct{}{}) }
func (c Client) NoTransport() int { return 1 }
func (c Client) call[T interface{ ~[]int | map[string]int }](method string, req interface{}) T { return zero[T]() }
"#;
    let methods = parse_client(text).unwrap();
    assert_eq!(methods.len(), 4);
    assert_eq!(methods[0].summary, "First method.");
    verify_request_shapes(&methods);
}
#[test]
fn invalid_go_requests_and_missing_fields_decline_explicitly() {
    assert!(GoSource::parse("package client\n func (").is_err());
    assert!(
        parse_client("package client\nfunc (c Client) Broken() { call[R](c, wire.MethodX) }")
            .is_err()
    );
    assert!(
        parse_client("package client\nfunc (c Client) Unknown() { call[R](c, wire.MethodX, raw) }")
            .is_err()
    );
    let source = GoSource::parse("package wire").unwrap();
    assert!(
        source
            .field(source.tree.root_node(), "missing")
            .unwrap_err()
            .contains("expected field")
    );
}
#[test]
fn generic_method_adapter_preserves_bytes_and_does_not_change_literal_examples() {
    let source = "package client\nconst Example = `\nfunc (c Client) Sample[T any]() {}\n`\nfunc (c Client) Real[T ~[]int /* ] */](args T) T { return args }\n";
    let normalized = generic_methods::normalized(source).unwrap();
    assert_eq!(source.len(), normalized.len());
    assert!(normalized.contains("Sample[T any]"));
    assert!(!normalized.contains("Real[T"));
    GoSource::parse(source).unwrap();
}
#[test]
fn generic_method_adapter_checks_quotes_comments_and_unbalanced_parameters() {
    for shape in [
        "[T interface{ Value() struct{ X string `json:\"[x]\"` } }]",
        "[T ~[]int // ]\n]",
        "[T interface{ X() string /* [ ] */ }]",
        "[T interface{ X() struct{ Tag string `x` } }]",
    ] {
        let source = format!("package client\nfunc (c Client) M{shape}(x T) {{}}");
        assert!(generic_methods::normalized(&source).is_ok());
    }
    for source in [
        "func (c Client) M[T any(",
        "func (c Client) M[T \"unclosed",
        "func (c Client) M[T /* unclosed",
    ] {
        assert!(generic_methods::normalized(source).is_err());
    }
}
#[test]
fn invalid_wire_tags_decline_and_ordinary_tags_keep_default_names() {
    for tag in ["bad", "json:unquoted", "json:\"unterminated"] {
        let source = format!("package wire\ntype Args struct{{ X string `{tag}` }}");
        assert!(parse_wire(&source, &mut WireHelp::default()).is_err());
    }
    let mut wire = WireHelp::default();
    parse_wire(
        "package wire\ntype Args struct{ X string `xml:\"x\"`; Y string \"json:\\\"why\\\"\" }",
        &mut wire,
    )
    .unwrap();
    assert_eq!(wire.requests["Args"][0].name, "X");
    assert_eq!(wire.requests["Args"][1].name, "why");
}

fn verify_request_shapes(methods: &[ClientHelp]) {
    assert_eq!(
        methods
            .iter()
            .map(|help| help.request.as_deref())
            .collect::<Vec<_>>(),
        [Some("OneArgs"), Some("TwoArgs"), Some("ThreeArgs"), None]
    );
}

#[test]
fn nearest_summary_and_fixed_array_dimensions_preserve_contract_intent() {
    let methods = parse_client("package client\n// mcp:summary Older section.\n// New section.\n// mcp:summary Current method.\nfunc (c Client) M() { return call[R](c, wire.MethodM, nil) }\n").unwrap();
    assert_eq!(methods[0].summary, "Current method.");
    let mut wire = WireHelp::default();
    parse_wire("package wire\ntype Args struct { Point [3]float64; Points [][2]int; Nested struct { Hidden string }; }\n", &mut wire).unwrap();
    assert_eq!(wire.requests["Args"].len(), 3);
    assert_eq!(wire.requests["Args"][0].type_name, "[3]float64");
    assert_eq!(wire.requests["Args"][1].type_name, "[][2]int");
}
