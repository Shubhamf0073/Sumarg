"""Tests for netlist_summary.py — run: python3 -m unittest discover -s hardware/scripts"""
import os
import unittest

import netlist_summary as ns

FIXTURE = os.path.join(os.path.dirname(__file__), "testdata", "netlist.xml")


class NetlistSummaryTest(unittest.TestCase):
    def setUp(self):
        self.comps, self.nets = ns.load(FIXTURE)

    def test_summary_lists_parts_and_flags_missing_mpn(self):
        text = ns.summary(self.comps, self.nets)
        self.assertIn("# 4 components, 6 nets", text)
        self.assertIn("missing MPN: R10", text)          # TP1 is exempt (test point)
        self.assertNotIn("TP1,", text.split("missing MPN:")[1].splitlines()[0])

    def test_summary_flags_single_node_nets(self):
        text = ns.summary(self.comps, self.nets)
        self.assertIn("single-node nets (unconnected?): /BATT_NTC, /KEEP_ALIVE", text)

    def test_pin_table_and_esp32s3_rules(self):
        text = ns.pins(self.comps, self.nets)
        self.assertIn("7 | GPIO4 | SOS_DRIVER", text)
        self.assertIn("GPIO18 (BATT_NTC) is analog but not on ADC1", text)
        self.assertIn("GPIO40 (TAMPER) must wake from deep sleep", text)
        self.assertIn("GPIO35 (KEEP_ALIVE) is reserved by octal PSRAM", text)
        self.assertNotIn("GPIO4 (SOS_DRIVER)", text)    # valid pin, no complaint
        self.assertNotIn("GPIO2 (VIN_SENSE)", text)

    def test_gpio_parsing(self):
        self.assertEqual(ns.gpio_of("GPIO04"), 4)
        self.assertEqual(ns.gpio_of("IO35"), 35)
        self.assertIsNone(ns.gpio_of("GND"))

    def test_no_mcu_role(self):
        comps = [c for c in self.comps if c["ref"] != "J5"]
        self.assertIn("Role=MCU", ns.pins(comps, self.nets))


if __name__ == "__main__":
    unittest.main()
