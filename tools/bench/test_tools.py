#!/usr/bin/env python3
from __future__ import annotations
import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from npc_checks import check_response, read_jsonl, validate_messages, allowed_numbers
from bench_ollama import payload_for, request_chat

ROOT = Path(__file__).resolve().parent

class ChecksTests(unittest.TestCase):
    def test_spelled_number_is_rejected(self):
        result = check_response("[gniew] Trzy? Żądam 15 kapsli.",
                                {"allowed_numbers":["3","15"],"required_numbers":["15"]})
        self.assertTrue(any("spelled_numerals" in x for x in result["errors"]))

    def test_wrong_counteroffer_is_rejected(self):
        result = check_response("[spokój] Nadal 60 kapsli.",
                                {"allowed_numbers":["40","52","60"],"required_numbers":["52"]})
        self.assertTrue(any("missing_required_numbers" in x for x in result["errors"]))

    def test_no_price_in_gear_chat(self):
        result = check_response("[spokój] Służba kosztuje 60 kapsli.",
                                {"allowed_numbers":["60"],"no_numbers":True})
        self.assertIn("numbers_in_non_numeric_case",result["errors"])

    def test_reference_never_enters_payload(self):
        case = read_jsonl(ROOT/"eval"/"regression-80.jsonl")[0]
        payload = payload_for(case,"mock-model",{"seed":1})
        self.assertEqual(set(payload),{"model","messages","stream","options"})
        self.assertEqual(payload["messages"][-1]["role"],"user")
        self.assertNotIn("reference",payload)
        self.assertNotIn(case["reference"],[m["content"] for m in payload["messages"]])

    def test_all_reference_formats(self):
        for case in read_jsonl(ROOT/"eval"/"regression-80.jsonl"):
            with self.subTest(case=case["id"]):
                self.assertFalse(validate_messages(case["messages"],has_target=False))
                self.assertEqual(set(case["checks"]["allowed_numbers"]), allowed_numbers(case["messages"]))
                self.assertFalse(check_response(case["reference"],case["checks"])["errors"])

    def test_mock_http_only(self):
        observed = []
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                observed.append(json.loads(self.rfile.read(int(self.headers["Content-Length"]))))
                self.send_response(200)
                self.send_header("Content-Type","application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"message":{"role":"assistant",
                    "content":"[spokój] To odpowiedź serwera testowego."},"done":True}).encode())
            def log_message(self,*args): pass
        server=HTTPServer(("127.0.0.1",0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True)
        thread.start()
        try:
            case=read_jsonl(ROOT/"eval"/"regression-80.jsonl")[0]
            payload=payload_for(case,"mock-model",{"temperature":0})
            result=request_chat(f"http://127.0.0.1:{server.server_port}",payload,5)
            self.assertTrue(result["done"])
            self.assertEqual(observed,[payload])
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)

if __name__=="__main__":
    unittest.main()
