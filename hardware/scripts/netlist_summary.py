#!/usr/bin/env python3
"""Condense a KiCad XML netlist (kicad-cli sch export netlist --format kicadxml) into short text for review.

    python3 -I hardware/scripts/netlist_summary.py out/netlist.xml          # components + nets + warnings
    python3 -I hardware/scripts/netlist_summary.py --pins out/netlist.xml   # MCU pin table for board_pins.h

A 200-part schematic's netlist XML is ~300 KB; this summary is a few KB, so Claude reviews it instead of the
sources. Standard library only (no new dependency).

--pins: finds the MCU part — a component whose `Role` field is `MCU` (set it on the ESP32-S3 module symbol, or on
the Waveshare header connector of SB-C1) — and prints "pin  function  net" for each of its pins, then flags
ESP32-S3 rule breaks: strapping pins 0/3/45/46 used, GPIO 33–37 used (octal PSRAM), analog nets off ADC1 (GPIO 1–10),
wake nets off GPIO 0–21. Pin functions must contain the GPIO number ("GPIO4", "IO4", "GPIO04").
"""
import re
import sys
import xml.etree.ElementTree as ET

STRAPPING = {0, 3, 45, 46}
OCTAL_PSRAM = set(range(33, 38))
ADC1 = set(range(1, 11))
RTC = set(range(0, 22))
WAKE_NETS = re.compile(r"^/?(IGN|SOS_|TAMPER)", re.I)
ANALOG_NETS = re.compile(r"(_SENSE|_NTC|_ADC)$", re.I)


def load(path):
    root = ET.parse(path).getroot()
    comps = []
    for c in root.iterfind("./components/comp"):
        fields = {f.get("name"): (f.text or "") for f in c.iterfind("./fields/field")}
        comps.append({
            "ref": c.get("ref"),
            "value": (c.findtext("value") or "").strip(),
            "footprint": (c.findtext("footprint") or "").strip(),
            "fields": fields,
        })
    nets = []
    for n in root.iterfind("./nets/net"):
        nodes = [(x.get("ref"), x.get("pin"), x.get("pinfunction") or "", x.get("pintype") or "")
                 for x in n.iterfind("node")]
        nets.append({"name": n.get("name"), "nodes": nodes})
    return comps, nets


def summary(comps, nets):
    out = [f"# {len(comps)} components, {len(nets)} nets", "", "## Components (ref | value | footprint | MPN)"]
    missing_mpn = []
    for c in sorted(comps, key=lambda c: natural(c["ref"])):
        mpn = c["fields"].get("MPN", "").strip()
        dnp = " DNP" if c["fields"].get("DNP", "").strip() not in ("", "0", "No", "no") else ""
        fp = c["footprint"].split(":")[-1]
        out.append(f"{c['ref']} | {c['value']} | {fp} | {mpn or '—'}{dnp}")
        if not mpn and not c["ref"].startswith(("TP", "H", "MH", "FID", "#")):
            missing_mpn.append(c["ref"])
    out += ["", "## Nets (name: ref.pin(function) …)"]
    single = []
    for n in sorted(nets, key=lambda n: n["name"]):
        nodes = " ".join(f"{r}.{p}" + (f"({f})" if f else "") for r, p, f, _ in n["nodes"])
        out.append(f"{n['name']}: {nodes}")
        if len(n["nodes"]) == 1 and not n["name"].startswith("unconnected-"):
            single.append(n["name"])
    out += ["", "## Warnings"]
    if missing_mpn:
        out.append(f"- missing MPN: {', '.join(missing_mpn)}")
    if single:
        out.append(f"- single-node nets (unconnected?): {', '.join(single)}")
    if not missing_mpn and not single:
        out.append("- none")
    return "\n".join(out)


def gpio_of(function):
    m = re.search(r"(?:GPIO|IO)0*(\d{1,2})\b", function, re.I)
    return int(m.group(1)) if m else None


def pins(comps, nets):
    mcu = [c["ref"] for c in comps if c["fields"].get("Role", "").strip().upper() == "MCU"]
    if not mcu:
        return "no component has field Role=MCU — set it on the ESP32-S3 module / Waveshare header symbol"
    ref = mcu[0]
    rows, problems = [], []
    for n in nets:
        for r, p, f, _ in n["nodes"]:
            if r != ref:
                continue
            g = gpio_of(f)
            name = n["name"].lstrip("/")
            rows.append((g if g is not None else 999, p, f, name))
            if g is None or name.startswith("unconnected-"):
                continue
            if g in STRAPPING and n["name"] not in ("GND",):
                problems.append(f"GPIO{g} ({name}) is a strapping pin — not for inputs")
            if g in OCTAL_PSRAM:
                problems.append(f"GPIO{g} ({name}) is reserved by octal PSRAM")
            if ANALOG_NETS.search(name) and g not in ADC1:
                problems.append(f"GPIO{g} ({name}) is analog but not on ADC1 (GPIO 1–10)")
            if WAKE_NETS.search(name) and g not in RTC:
                problems.append(f"GPIO{g} ({name}) must wake from deep sleep — use GPIO 0–21")
    out = [f"# MCU {ref}: pin | function | net"]
    for g, p, f, name in sorted(rows):
        out.append(f"{p} | {f} | {name}")
    out += ["", "## Rule check"] + ([f"- {x}" for x in problems] or ["- ok"])
    return "\n".join(out)


def natural(s):
    return [int(t) if t.isdigit() else t for t in re.split(r"(\d+)", s or "")]


def main(argv):
    pin_mode = "--pins" in argv
    args = [a for a in argv if a != "--pins"]
    if len(args) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    comps, nets = load(args[0])
    print(pins(comps, nets) if pin_mode else summary(comps, nets))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
