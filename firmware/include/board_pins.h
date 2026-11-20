// Every GPIO of the bus unit lives here, keyed by HW_REV (docs/12 §4): 0 bench, 1 SB-C1 carrier, 2 SB-R1. Pins not yet verified on the real board stay
// compile errors on purpose — do not guess pin numbers (a wrong pin can short the modem's power key).
#pragma once

#if defined(UNIT_TEST)
// Host tests never touch pins.
#elif HW_REV == 0
// Bench prototype: Waveshare ESP32-S3-SIM7670G-4G (camera, TF slot, SIM7670G with GNSS, 18650 on board)
// + PN532 (or RC522) on SPI + OLED on I2C + buzzer + SOS button input. Waveshare publishes V1 and V2 tables that
// differ, some V1 entries collide with octal PSRAM (GPIO 33-37), and the modem UART/power pins are not published:
// identify the board revision, then measure during M7.1 (/hw-bringup proto) and remove the #error.
#error "HW_REV 0: measure pins on the Waveshare ESP32-S3-SIM7670G-4G (docs/12 §4), then delete this line"
#elif HW_REV == 1
// SB-C1: the same Waveshare board on the SB-C1 carrier (hardware/sb-c1/SPEC.md). Camera/TF/gauge/modem pins as
// HW_REV 0; IGN, SOS_DRIVER, SOS_DOOR, TAMPER (RTC GPIO 0-21), VIN_SENSE and BATT_NTC (ADC1), KEEP_ALIVE, BUZZER,
// LED, PN532 SPI and OLED I2C on the header pins the carrier schematic assigns (hardware/sb-c1/out/pins.txt).
#error "HW_REV 1: fill pins from the SB-C1 carrier netlist (pins.txt) after bring-up (docs/12 §4), then delete this line"
#elif HW_REV == 2
#error "HW_REV 2: fill pins from the SB-R1 schematic netlist (docs/12 §4), then delete this line"
#else
#error "Unknown HW_REV"
#endif
