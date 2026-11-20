// Build-time constants. Runtime behaviour comes from the cloud config (docs/13 §6); pins from board_pins.h.
#pragma once
#ifndef FW_VERSION
#define FW_VERSION "0.0.0-dev"
#endif
#define PROTOCOL_MAJOR 1
#define UNIT_ID_PREFIX "SB-"          // + last 3 bytes of the factory MAC, hex (docs/13 §1)
#define MAX_ROSTER 128                // students per unit across its routes (docs/14 §5)
#define JOURNAL_RECORD_MAX 512        // bytes per journal entry (positions/events; photos go to /p/)
