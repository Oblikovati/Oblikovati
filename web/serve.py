#!/usr/bin/env python3
"""Serve the Oblikovati WebAssembly + WebGPU shell with required COOP/COEP headers."""
import functools
import http.server
import pathlib
import sys

WEB_ROOT = pathlib.Path(__file__).resolve().parent


class IsolatedHandler(http.server.SimpleHTTPRequestHandler):
    extensions_map = {
        **http.server.SimpleHTTPRequestHandler.extensions_map,
        ".wasm": "application/wasm",
        ".js": "application/javascript",
    }

    def end_headers(self):
        self.send_header("Cross-Origin-Opener-Policy", "same-origin")
        self.send_header("Cross-Origin-Embedder-Policy", "require-corp")
        self.send_header("Cross-Origin-Resource-Policy", "same-origin")
        super().end_headers()


def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8080
    handler = functools.partial(IsolatedHandler, directory=str(WEB_ROOT))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", port), handler)
    print(f"Serving Oblikovati Web at http://127.0.0.1:{port}/", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
