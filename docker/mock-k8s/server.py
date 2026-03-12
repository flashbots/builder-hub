#!/usr/bin/env python3
"""Minimal Kubernetes TokenReview API mock for local Vault Kubernetes auth testing."""
from http.server import HTTPServer, BaseHTTPRequestHandler
import json


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()

    def do_POST(self):
        content_length = int(self.headers.get("Content-Length", 0))
        self.rfile.read(content_length)
        body = json.dumps({
            "apiVersion": "authentication.k8s.io/v1",
            "kind": "TokenReview",
            "status": {
                "authenticated": True,
                "user": {
                    "username": "system:serviceaccount:default:builder-hub",
                    "uid": "builder-hub-dev-uid",
                    "groups": [
                        "system:serviceaccounts",
                        "system:serviceaccounts:default",
                        "system:authenticated",
                    ],
                },
            },
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        pass


HTTPServer(("0.0.0.0", 8443), Handler).serve_forever()
