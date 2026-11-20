// Boarding ledger: turns RFID taps into BOARD / ALIGHT decisions on the unit, so the driver and child get instant
// feedback (buzzer + display) without the network. The server re-checks every event (docs/05 §3, docs/14 §5).
// Pure C++17: no Arduino headers, no heap after construction, unit-tested on the host (`pio test -e native`).
#pragma once
#include <array>
#include <cstddef>
#include <cstdint>

namespace sf {

constexpr std::size_t kMaxUidLen = 10;   // MIFARE UIDs are 4, 7 or 10 bytes

struct Uid {
    std::array<std::uint8_t, kMaxUidLen> b{};
    std::uint8_t len = 0;
    bool operator==(const Uid& o) const;
    bool operator<(const Uid& o) const;    // ordering for the sorted roster
    bool valid() const { return len == 4 || len == 7 || len == 10; }
};

struct RosterEntry {
    Uid uid;
    char short_name[16];   // first name only, NUL-terminated (data minimisation, docs/07 §3)
};

// Roster = cards the unit may see, pushed in the config (uid + first name for the display). Sorted, binary search.
template <std::size_t N>
class Roster {
public:
    using Entry = RosterEntry;
    // Replaces the roster; entries need not be sorted. Returns false (and keeps the old roster) if count > N or
    // any uid is invalid or duplicated.
    bool load(const Entry* entries, std::size_t count);
    // Index of uid, or -1.
    int find(const Uid& uid) const;
    std::size_t size() const { return size_; }
    const Entry& at(std::size_t i) const { return items_[i]; }

private:
    std::array<Entry, N> items_{};
    std::size_t size_ = 0;
};

enum class TapResult : std::uint8_t {
    Board,      // first tap of the trip → on the bus
    Alight,     // second tap after min_ride_s → off the bus
    Duplicate,  // same card again too soon (double tap) → ignored, short beep
    Unknown,    // not on the roster → long beep, UNKNOWN_CARD event (server decides)
    NoTrip,     // tap with no trip running → stored as an event, no board/alight decision
};

struct LedgerConfig {
    std::uint32_t dedupe_s = 5;       // ignore the same card within this window
    std::uint32_t min_ride_s = 90;    // a second tap sooner than this is a double tap, not an alight
};

template <std::size_t N>
class BoardingLedger {
public:
    explicit BoardingLedger(LedgerConfig cfg = {}) : cfg_(cfg) {}
    void begin_trip(std::uint32_t trip_ref);   // clears state; trip_ref comes from the trip config/command
    void end_trip();
    bool trip_running() const { return trip_ref_ != 0; }
    // now_s: monotonic seconds (GPS/RTC). roster_index: Roster::find result.
    TapResult on_tap(int roster_index, std::uint32_t now_s);
    std::size_t on_board() const;               // count currently on the bus (for the "not dropped" check)

private:
    enum class State : std::uint8_t { Away, OnBoard, Done };
    LedgerConfig cfg_;
    std::uint32_t trip_ref_ = 0;
    std::array<State, N> state_{};
    std::array<std::uint32_t, N> last_tap_s_{};
    std::array<std::uint32_t, N> board_s_{};
};

}  // namespace sf

#include "boarding_impl.h"
