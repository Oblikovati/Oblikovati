#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""End-to-end headless Chromium test loading real Go js-wasm add-ins into Oblikovati Web."""
import argparse
import functools
import http.server
import io
import json
import pathlib
import sys
import threading
import zipfile

WEB_ROOT = pathlib.Path(__file__).resolve().parent


def make_bundle_zip(manifest_path: pathlib.Path, icon_path: pathlib.Path, wasm_path: pathlib.Path) -> bytes:
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        zf.write(manifest_path, "manifest.json")
        zf.write(icon_path, "icon.svg")
        zf.write(wasm_path, wasm_path.name)
    return buf.getvalue()


class AddInTestHandler(http.server.SimpleHTTPRequestHandler):
    extra_files: dict[str, tuple[str, bytes]] = {}
    extensions_map = {
        **http.server.SimpleHTTPRequestHandler.extensions_map,
        ".wasm": "application/wasm",
        ".js": "application/javascript",
        ".zip": "application/zip",
    }

    def end_headers(self):
        self.send_header("Cross-Origin-Opener-Policy", "same-origin")
        self.send_header("Cross-Origin-Embedder-Policy", "require-corp")
        self.send_header("Cross-Origin-Resource-Policy", "same-origin")
        super().end_headers()

    def do_GET(self):
        path = self.path.split("?", 1)[0]
        if path in self.extra_files:
            ctype, body = self.extra_files[path]
            self.send_response(200)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        super().do_GET()

    def log_message(self, format, *args):
        return


def main():
    from playwright.sync_api import sync_playwright

    parser = argparse.ArgumentParser()
    parser.add_argument("--mcp-wasm", type=pathlib.Path, required=True)
    parser.add_argument("--motor-wasm", type=pathlib.Path, required=True)
    parser.add_argument("--motor-manifest", type=pathlib.Path, required=True)
    parser.add_argument("--motor-icon", type=pathlib.Path, required=True)
    args = parser.parse_args()

    bundle_zip = make_bundle_zip(args.motor_manifest, args.motor_icon, args.motor_wasm)
    AddInTestHandler.extra_files = {
        "/test-addins/oblikovati-mcp-bridge.wasm": ("application/wasm", args.mcp_wasm.read_bytes()),
        "/test-addins/motor-designer-js-wasm.zip": ("application/zip", bundle_zip),
    }

    handler = functools.partial(AddInTestHandler, directory=str(WEB_ROOT))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    port = server.server_address[1]
    threading.Thread(target=server.serve_forever, daemon=True).start()

    console_logs = []
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(
                headless=True,
                args=[
                    "--enable-unsafe-webgpu",
                    "--use-angle=vulkan",
                    "--enable-features=Vulkan",
                    "--ignore-gpu-blocklist",
                    "--no-sandbox",
                ],
            )
            page = browser.new_page(viewport={"width": 1280, "height": 720})
            page.on("console", lambda msg: console_logs.append(f"{msg.type}: {msg.text}"))
            page.on("pageerror", lambda err: console_logs.append(f"pageerror: {err}"))

            page.goto(f"http://127.0.0.1:{port}/", wait_until="domcontentloaded")
            page.wait_for_function("() => Boolean(window.__OBLIKOVATI_WEB__)", timeout=30000)

            result = page.evaluate(
                """async () => {
                const web = window.__OBLIKOVATI_WEB__;
                const mcpLoaded = await web.addins.loadUrl("/test-addins/oblikovati-mcp-bridge.wasm");
                const startWait = Date.now();
                while (!window.__OBLIKOVATI_MCP__ && Date.now() - startWait < 15000) {
                    await new Promise((r) => setTimeout(r, 50));
                }
                if (!window.__OBLIKOVATI_MCP__) {
                    throw new Error("window.__OBLIKOVATI_MCP__ was not registered");
                }
                const tools = window.__OBLIKOVATI_MCP__.listTools();
                const createRes = await window.__OBLIKOVATI_MCP__.callTool("create_document", { type: "part", name: "WasmPart" });
                const listRes = await window.__OBLIKOVATI_MCP__.callTool("list_documents", {});
                const jsonRpcPing = await window.__OBLIKOVATI_MCP__.request({
                    jsonrpc: "2.0",
                    id: 7,
                    method: "ping",
                });

                const motorLoaded = await web.addins.loadBundleZip("/test-addins/motor-designer-js-wasm.zip");
                const loadedList = await web.addins.list();
                const unloaded = await web.addins.unload(motorLoaded.probe.id);
                const afterUnload = await web.addins.list();
                return {
                    apiVersion: web.apiVersion,
                    mcpProbe: mcpLoaded.probe,
                    toolCount: tools.length,
                    createRes,
                    listRes,
                    jsonRpcPing,
                    motorProbe: motorLoaded.probe,
                    loadedIds: loadedList.map((a) => a.id),
                    unloaded,
                    remainingIds: afterUnload.map((a) => a.id),
                };
            }"""
            )
            browser.close()
    except Exception:
        if console_logs:
            print("Browser console logs:", file=sys.stderr)
            for line in console_logs:
                print(f"  {line}", file=sys.stderr)
        raise
    finally:
        server.shutdown()

    print(json.dumps(result, indent=2))
    assert result["mcpProbe"]["id"] == "com.oblikovati.mcp-bridge"
    assert result["toolCount"] > 500, f"expected >500 MCP tools, got {result['toolCount']}"
    assert not result["createRes"].get("isError"), f"create_document failed: {result['createRes']}"
    assert not result["listRes"].get("isError"), f"list_documents failed: {result['listRes']}"
    assert result["jsonRpcPing"].get("id") == 7 and "error" not in result["jsonRpcPing"]
    assert result["motorProbe"]["id"] == "com.oblikovati.motor-designer"
    assert sorted(result["loadedIds"]) == ["com.oblikovati.mcp-bridge", "com.oblikovati.motor-designer"]
    assert result["remainingIds"] == ["com.oblikovati.mcp-bridge"]
    print("E2E js-wasm add-in verification PASSED!")


if __name__ == "__main__":
    main()
