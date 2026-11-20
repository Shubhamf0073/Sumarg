# Compliance notes for the SB unit (India)

**Not legal advice.** This lists what to ask a WPC/TEC compliance consultant before paid installs at scale
(docs/09 H8). Facts were checked on public sources in Nov 2026; the consultant's answer wins.

The unit contains: ESP32-S3 (2.4 GHz Wi-Fi + Bluetooth LE), a 13.56 MHz RFID reader, an LTE Cat-1 modem with GNSS,
and a SIM supplied by Previa. Previa owns the units and provides the service to schools.

## 1. WPC Equipment Type Approval (ETA) — short-range radios
- ETA is required for wireless equipment in de-licensed bands (Wi-Fi, Bluetooth, RFID), for imported **and**
  India-made products.
- Products in de-licensed bands can usually get ETA through **self-declaration** on the Saral Sanchar portal, using
  the RF test reports of the radio modules; consultant guides quote a government fee of about ₹10,000 per model.
- Ask: one ETA per model (SB-C1, SB-R1) or per radio module; whether the Waveshare board's importer already holds an
  ETA we can reference; whether disabling Wi-Fi in firmware changes anything (expect: no).

## 2. TEC MTCTE — telecom equipment
- MTCTE lists product categories that need TEC certification before sale/import/use. Categories that may touch the
  SB unit: **"Tracking device (with embedded SIM)"**, **"IoT gateway"**, and **"Equipment operating in 2.4 GHz and
  5 GHz bands"**. TEC has extended general exemptions for some categories in the past; check the current list.
- Ask: does a bus unit with a removable nano-SIM (SB-C1) or an MFF2 eSIM (SB-R1 option) fall under "tracking
  device"? If yes: test lab, sample count, cost, timeline, and whether module-level reports can be reused.

## 3. M2M service provider (M2MSP) registration — DoT
- Entities offering M2M services based on SIM/LAN register with DoT as **M2MSP** on the Saral Sanchar portal
  (application fee ₹5,000; ₹10,000 if also registering as a WPAN/WLAN connectivity provider). Telcos ask for the
  DoT registration number before giving telecom resources (SIMs) for M2M services.
- The M2MSP guidelines also require inducting only devices that meet TEC standards where those are mandatory — so
  §2's answer matters here too.
- This is the first registration to do: IoT SIM procurement waits on it (docs/15 §6).

## 4. Vehicle-side
- Our unit is an **add-on**; the bus keeps its AIS-140 tracker (docs/02 A29). Installation must not modify the
  vehicle's certified wiring (fused tap on permanent and ignition lines only, docs/12 §8).
- AIS-140 type approval (ARAI/ICAT), including the vehicle EMC requirements it brings, is Phase 3 (docs/12 §10).

## 5. Other items to ask about once
- E-waste EPR obligations as a "producer" of electronic equipment, and battery EPR for the 18650 cells.
- Whether any BIS registration applies to the unit or its power accessories (none expected; confirm).

## Record
| Item | Consultant answer | Date | Cost | Status |
|---|---|---|---|---|
| WPC ETA | | | | |
| MTCTE | | | | |
| M2MSP | | | | |
| EPR (e-waste, battery) | | | | |
