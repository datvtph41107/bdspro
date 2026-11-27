import tempfile
import unittest
from pathlib import Path

from verify_logging import CASES, events_for, local_base, verify_events


class VerificationTests(unittest.TestCase):
    def test_local_only(self):
        for value in ("http://example.org:8080", "http://localhost:8080", "https://127.0.0.1:8080", "http://127.0.0.1:8080/foo", "http://user@127.0.0.1:8080"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                local_base(value)
        self.assertEqual(local_base("http://127.0.0.1:8080/"), "http://127.0.0.1:8080")
        self.assertEqual(local_base("http://[::1]:8080"), "http://[::1]:8080")

    def test_events(self):
        for case in CASES:
            method, path, status, level, suffix = case
            start = {"message": "http.started", "method": method, "path": path}
            end = {"message": "http.request", "method": method, "path": path, "status": status, "level": level}
            if suffix:
                end.update(handler="pkg" + suffix, handler_source="file.go:1")
            verify_events([start, end], case, "secret-probe")
            for events in ([end], [end, start], [start, end, end], [start, dict(end, status=999)], [start, dict(end, extra="secret-probe")]):
                with self.subTest(path=path, events=events), self.assertRaises(ValueError):
                    verify_events(events, case, "secret-probe")

    def test_mixed_log(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "test.log"
            path.write_text('old text\n{"request_id":"other"}\n{"request_id":"wanted","message":"http.request"}\n{')
            self.assertEqual(events_for(path, "wanted"), [{"request_id": "wanted", "message": "http.request"}])


if __name__ == "__main__":
    unittest.main()
