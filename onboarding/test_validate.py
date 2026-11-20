"""Tests for validate.py — run: python3 -m unittest discover -s onboarding"""
import csv
import shutil
import tempfile
import unittest
from pathlib import Path

import validate

DEMO = Path(__file__).resolve().parent / "demo"


class ValidateTest(unittest.TestCase):
    def setUp(self):
        self.dir = Path(tempfile.mkdtemp())
        for f in DEMO.glob("*.csv"):
            shutil.copy(f, self.dir)

    def tearDown(self):
        shutil.rmtree(self.dir)

    def edit(self, name, fn):
        path = self.dir / f"{name}.csv"
        with open(path, encoding="utf-8") as fh:
            rows = list(csv.reader(fh))
        fn(rows)
        with open(path, "w", newline="", encoding="utf-8") as fh:
            csv.writer(fh, lineterminator="\n").writerows(rows)

    def test_demo_school_is_clean(self):
        self.assertEqual(validate.validate(DEMO).problems, [])

    def test_templates_match_demo_headers(self):
        for t in (Path(__file__).parent / "templates").glob("*.csv"):
            demo = DEMO / t.name
            self.assertTrue(demo.exists(), t.name)
            self.assertEqual(validate.header_of(t), validate.header_of(demo), t.name)

    def test_normalises_reg_no_and_mobile(self):
        self.edit("vehicles", lambda r: r[1].__setitem__(1, "GJ 01 AB 1001"))
        self.edit("crew", lambda r: r[1].__setitem__(3, "+91 90000 00011"))
        self.assertEqual(validate.validate(self.dir).problems, [])

    def test_reports_reference_and_format_errors_without_personal_values(self):
        self.edit("students_guardians", lambda r: (r[1].__setitem__(6, "Nowhere"), r[2].__setitem__(10, "12345")))
        probs = validate.validate(self.dir).problems
        self.assertIn("students_guardians.csv row 2: to_stop_name: not a stop on route R-01", probs)
        self.assertIn("students_guardians.csv row 3: guardian1_mobile: 10 digits starting 6–9", probs)
        self.assertFalse(any("12345" in p for p in probs))      # mobiles never echoed

    def test_route_rows_must_agree_and_seq_be_contiguous(self):
        self.edit("routes_stops", lambda r: (r[2].__setitem__(8, "07:10"), r[3].__setitem__(9, "7")))
        probs = "\n".join(validate.validate(self.dir).problems)
        self.assertIn("route columns: differ from row 2", probs)
        self.assertIn("route R-01 to_school must be 1..4", probs)

    def test_duplicate_card_uid(self):
        self.edit("cards", lambda r: r[2].__setitem__(1, r[1][1]))
        self.assertIn("cards.csv row 3: card_uid: duplicate ('DE100001')", validate.validate(self.dir).problems)


if __name__ == "__main__":
    unittest.main()
