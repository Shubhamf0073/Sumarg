// Sumarg bus unit — boot and task wiring (docs/14 §3). Modules arrive milestone by milestone (docs/08 M7);
// until then this file documents the shape so every session builds into the same structure.
//
// Tasks (FreeRTOS, pinned):                         core prio  job
//   gps      drivers/gnss      UART NMEA/UBX 1–5 Hz   1   4   fixes, time sync, moving flag
//   rfid     drivers/rc522     SPI poll 20 Hz          1   5   card UID → logic queue (debounced)
//   camera   drivers/cam       on demand               1   3   one JPEG (VGA, q≈12) per accepted tap → /p/ on SD
//   logic    app/logic         1 Hz + events           1   3   BoardingLedger, report policy, SOS, buzzer/display
//   store    app/journal       event                   0   3   append records/events to SD journal with seq
//   net      net/mqtt          event                   0   2   modem attach, MQTT/TLS, publish backlog oldest-first,
//                                                               config/cmd in, photos out, OTA trigger
//   health   app/health        30 s                    0   1   status message, watchdogs, SD/heap/battery
// Order is always: sample → journal (with seq) → publish → advance acked pointer (never publish unjournaled data).
#include <Arduino.h>

#include "board_pins.h"
#include "build_config.h"
#include "logic/boarding.h"

void setup() {
    Serial.begin(115200);
    // M7.2: read DIP/installer mode, NVS (unit uid, secret, seq), mount SD, start tasks above.
}

void loop() {
    vTaskDelay(pdMS_TO_TICKS(1000));   // all work happens in tasks
}
