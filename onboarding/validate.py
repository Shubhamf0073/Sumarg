#!/usr/bin/env python3
"""Offline pre-check of a school's onboarding CSVs against onboarding/templates/README.md.

    python3 -I onboarding/validate.py <folder with the CSVs>        # e.g. onboarding/demo

Prints problems as "<file> row <n>: <column>: <problem>" and exits 1 if any. Personal values (names, mobiles,
notes) are never printed — only row numbers and columns (docs/07 S18). The Go importer (plan M9.2) implements
the same rules; this script is the executable spec and a quick check before upload. Standard library only.
"""
import csv
import re
import sys
from datetime import datetime
from pathlib import Path

TEMPLATES = Path(__file__).resolve().parent / "templates"
ORDER = ["campuses", "staff", "vehicles", "crew", "routes_stops", "students_guardians", "cards"]
PERSONAL = re.compile(r"name$|mobile$|note$|email$|address$|licence_no$")

MOBILE = re.compile(r"^[6-9][0-9]{9}$")
CAMPUS = re.compile(r"^[A-Z0-9]{2,8}$")
REG = re.compile(r"^[A-Z]{2}[0-9]{1,2}[A-Z]{0,3}[0-9]{1,4}$")
ROUTE = re.compile(r"^[A-Z0-9-]{2,12}$")
UID = re.compile(r"^[0-9A-F]{8,20}$")
TIME = re.compile(r"^([01][0-9]|2[0-3]):[0-5][0-9]$")
RELATIONS = {"mother", "father", "guardian", "grandparent", "sibling", "other"}


class Report:
    def __init__(self):
        self.problems = []

    def add(self, file, row, col, msg, value=None):
        shown = "" if value is None or PERSONAL.search(col) else f" ({value!r})"
        self.problems.append(f"{file} row {row}: {col}: {msg}{shown}")


def norm_mobile(v):
    v = re.sub(r"[\s\-()]", "", v or "")
    if v.startswith("+91"):
        v = v[3:]
    elif v.startswith("91") and len(v) == 12:
        v = v[2:]
    elif v.startswith("0") and len(v) == 11:
        v = v[1:]
    return v


def norm_reg(v):
    return re.sub(r"[\s\-]", "", (v or "").upper())


def is_date(v):
    for fmt in ("%d-%m-%Y", "%Y-%m-%d"):
        try:
            datetime.strptime(v, fmt)
            return True
        except ValueError:
            pass
    return False


def header_of(path):
    with open(path, encoding="utf-8-sig", newline="") as fh:
        return next(csv.reader(fh), [])


def load(folder, name, rep):
    path = Path(folder) / f"{name}.csv"
    if not path.exists():
        return None
    expected = header_of(TEMPLATES / f"{name}.csv")
    with open(path, encoding="utf-8-sig", newline="") as fh:
        rows = list(csv.DictReader(fh))
    header = header_of(path)
    missing = [c for c in expected if c not in header]
    if missing:
        rep.add(f"{name}.csv", 1, ",".join(missing), "missing columns")
    return [{k: (v or "").strip() for k, v in r.items() if k} for r in rows]


def need(rep, f, n, r, col):
    if not r.get(col):
        rep.add(f, n, col, "required")
        return False
    return True


def validate(folder):
    rep = Report()
    data = {name: load(folder, name, rep) for name in ORDER}
    if data["campuses"] is None:
        rep.add("campuses.csv", 0, "-", "file missing")
        return rep
    campuses = set()
    for n, r in enumerate(data["campuses"], 2):
        f = "campuses.csv"
        if need(rep, f, n, r, "campus_code") and not CAMPUS.match(r["campus_code"]):
            rep.add(f, n, "campus_code", "2–8 upper-case letters/digits", r["campus_code"])
        campuses.add(r.get("campus_code"))
        need(rep, f, n, r, "name")
        if not (r.get("lat") and r.get("lon")) and not r.get("maps_link"):
            rep.add(f, n, "lat/lon", "give lat+lon or maps_link")

    def check_campus(f, n, r, blank_ok=False):
        c = r.get("campus_code", "")
        if not c and blank_ok:
            return
        if c not in campuses:
            rep.add(f, n, "campus_code", "not in campuses.csv", c)

    mobiles_users = {}
    for n, r in enumerate(data["staff"] or [], 2):
        f = "staff.csv"
        check_campus(f, n, r, blank_ok=True)
        need(rep, f, n, r, "name")
        m = norm_mobile(r.get("mobile"))
        if not MOBILE.match(m):
            rep.add(f, n, "mobile", "10 digits starting 6–9")
        elif m in mobiles_users:
            rep.add(f, n, "mobile", f"duplicate of row {mobiles_users[m]}")
        mobiles_users.setdefault(m, n)
        if r.get("role") not in {"owner", "admin", "transport_manager"}:
            rep.add(f, n, "role", "owner | admin | transport_manager", r.get("role"))

    vehicles = set()
    for n, r in enumerate(data["vehicles"] or [], 2):
        f = "vehicles.csv"
        check_campus(f, n, r)
        reg = norm_reg(r.get("reg_no"))
        if not REG.match(reg):
            rep.add(f, n, "reg_no", "like GJ01AB1234", r.get("reg_no"))
        elif reg in vehicles:
            rep.add(f, n, "reg_no", "duplicate", reg)
        vehicles.add(reg)
        need(rep, f, n, r, "label")
        if not r.get("seats", "").isdigit() or not 4 <= int(r["seats"]) <= 80:
            rep.add(f, n, "seats", "4–80", r.get("seats"))
        if r.get("ownership", "school") not in {"school", "contractor", ""}:
            rep.add(f, n, "ownership", "school | contractor", r.get("ownership"))
        for col in ("fitness_expiry", "insurance_expiry", "permit_expiry", "puc_expiry"):
            if r.get(col) and not is_date(r[col]):
                rep.add(f, n, col, "date DD-MM-YYYY or YYYY-MM-DD", r[col])

    crew = {}
    for n, r in enumerate(data["crew"] or [], 2):
        f = "crew.csv"
        check_campus(f, n, r)
        if r.get("kind") not in {"driver", "attendant"}:
            rep.add(f, n, "kind", "driver | attendant", r.get("kind"))
        need(rep, f, n, r, "name")
        m = norm_mobile(r.get("mobile"))
        if not MOBILE.match(m):
            rep.add(f, n, "mobile", "10 digits starting 6–9")
        elif m in crew or m in mobiles_users:
            rep.add(f, n, "mobile", "already used by another user of this school")
        crew[m] = r.get("kind")
        if r.get("kind") == "driver":
            need(rep, f, n, r, "licence_no")
            if need(rep, f, n, r, "licence_expiry") and not is_date(r["licence_expiry"]):
                rep.add(f, n, "licence_expiry", "date", r["licence_expiry"])

    routes = {}   # (code, direction, shift) -> {"stops": {name: seq}, "first_row": n, "fields": tuple}
    for n, r in enumerate(data["routes_stops"] or [], 2):
        f = "routes_stops.csv"
        check_campus(f, n, r)
        code = r.get("route_code", "")
        if not ROUTE.match(code):
            rep.add(f, n, "route_code", "2–12 of A–Z 0–9 -", code)
        if r.get("direction") not in {"to_school", "from_school"}:
            rep.add(f, n, "direction", "to_school | from_school", r.get("direction"))
        shift = r.get("shift") or "regular"
        if shift not in {"regular", "half_day", "activity"}:
            rep.add(f, n, "shift", "regular | half_day | activity", shift)
        if not TIME.match(r.get("planned_start", "")):
            rep.add(f, n, "planned_start", "HH:MM 24 h", r.get("planned_start"))
        if r.get("vehicle_reg_no") and norm_reg(r["vehicle_reg_no"]) not in vehicles:
            rep.add(f, n, "vehicle_reg_no", "not in vehicles.csv", r["vehicle_reg_no"])
        for col, kind in (("driver_mobile", "driver"), ("attendant_mobile", "attendant")):
            if r.get(col) and crew.get(norm_mobile(r[col])) != kind:
                rep.add(f, n, col, f"not a {kind} in crew.csv")
        key = (code, r.get("direction"), shift)
        fields = (r.get("route_name"), r.get("vehicle_reg_no"), r.get("driver_mobile"), r.get("planned_start"))
        route = routes.setdefault(key, {"stops": {}, "seqs": [], "fields": fields, "first": n})
        if route["fields"] != fields:
            rep.add(f, n, "route columns", f"differ from row {route['first']} of the same route")
        if not r.get("stop_seq", "").isdigit():
            rep.add(f, n, "stop_seq", "1, 2, 3 …", r.get("stop_seq"))
        else:
            route["seqs"].append(int(r["stop_seq"]))
        if need(rep, f, n, r, "stop_name"):
            if r["stop_name"] in route["stops"]:
                rep.add(f, n, "stop_name", "duplicate within the route", r["stop_name"])
            route["stops"][r["stop_name"]] = n
        has_pin = r.get("lat") and r.get("lon")
        if has_pin:
            try:
                lat, lon = float(r["lat"]), float(r["lon"])
                if not (6 <= lat <= 37 and 68 <= lon <= 98):
                    rep.add(f, n, "lat/lon", "outside India — swapped?", f"{lat},{lon}")
            except ValueError:
                rep.add(f, n, "lat/lon", "decimal degrees", f"{r['lat']},{r['lon']}")
        elif not (r.get("maps_link") or r.get("address")):
            rep.add(f, n, "lat/lon", "give lat+lon, maps_link or address")
    for (code, direction, shift), route in routes.items():
        seqs = sorted(route["seqs"])
        if seqs != list(range(1, len(seqs) + 1)):
            rep.add("routes_stops.csv", route["first"], "stop_seq", f"route {code} {direction} must be 1..{len(seqs)}")
        if len(seqs) > 25:
            rep.add("routes_stops.csv", route["first"], "stop_seq", f"route {code} has more than 25 stops")

    def stop_ok(code, stop):
        return any(k[0] == code and stop in v["stops"] for k, v in routes.items())

    admissions = set()
    for n, r in enumerate(data["students_guardians"] or [], 2):
        f = "students_guardians.csv"
        check_campus(f, n, r)
        if need(rep, f, n, r, "admission_no"):
            if r["admission_no"] in admissions:
                rep.add(f, n, "admission_no", "duplicate", r["admission_no"])
            admissions.add(r["admission_no"])
        need(rep, f, n, r, "student_name")
        need(rep, f, n, r, "class")
        legs = [(r.get("to_route_code"), r.get("to_stop_name"), "to"), (r.get("from_route_code"), r.get("from_stop_name"), "from")]
        if not any(c for c, _, _ in legs):
            rep.add(f, n, "to_route_code/from_route_code", "at least one route")
        for code, stop, leg in legs:
            if code and not stop_ok(code, stop):
                rep.add(f, n, f"{leg}_stop_name", f"not a stop on route {code}")
        for g in ("guardian1", "guardian2"):
            present = r.get(f"{g}_name") or r.get(f"{g}_mobile")
            if g == "guardian1" or present:
                need(rep, f, n, r, f"{g}_name")
                if not MOBILE.match(norm_mobile(r.get(f"{g}_mobile"))):
                    rep.add(f, n, f"{g}_mobile", "10 digits starting 6–9")
                if r.get(f"{g}_relation") not in RELATIONS:
                    rep.add(f, n, f"{g}_relation", " | ".join(sorted(RELATIONS)), r.get(f"{g}_relation"))
                if r.get(f"{g}_language", "en") not in {"en", "gu", "hi", ""}:
                    rep.add(f, n, f"{g}_language", "en | gu | hi", r.get(f"{g}_language"))

    uids = set()
    for n, r in enumerate(data["cards"] or [], 2):
        f = "cards.csv"
        if r.get("admission_no") not in admissions:
            rep.add(f, n, "admission_no", "not in students_guardians.csv", r.get("admission_no"))
        uid = (r.get("card_uid") or "").upper().replace(":", "").replace(" ", "")
        if not UID.match(uid):
            rep.add(f, n, "card_uid", "hex, 8–20 characters", r.get("card_uid"))
        elif uid in uids:
            rep.add(f, n, "card_uid", "duplicate", uid)
        uids.add(uid)
    return rep


def main(argv):
    if len(argv) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    rep = validate(argv[0])
    for p in rep.problems:
        print(p)
    print(f"{len(rep.problems)} problem(s)")
    return 1 if rep.problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
