// Template definitions for boarding.h (header-only so tests and firmware instantiate their own sizes).
#pragma once
#include <algorithm>
#include <cstring>

namespace sf {

inline bool Uid::operator==(const Uid& o) const {
    return len == o.len && std::equal(b.begin(), b.begin() + len, o.b.begin());
}
inline bool Uid::operator<(const Uid& o) const {
    if (len != o.len) return len < o.len;
    return std::lexicographical_compare(b.begin(), b.begin() + len, o.b.begin(), o.b.begin() + o.len);
}

template <std::size_t N>
bool Roster<N>::load(const Entry* entries, std::size_t count) {
    if (count > N) return false;
    std::array<Entry, N> tmp{};
    for (std::size_t i = 0; i < count; ++i) {
        if (!entries[i].uid.valid()) return false;
        tmp[i] = entries[i];
        tmp[i].short_name[sizeof(tmp[i].short_name) - 1] = '\0';
    }
    std::sort(tmp.begin(), tmp.begin() + count, [](const Entry& a, const Entry& b) { return a.uid < b.uid; });
    for (std::size_t i = 1; i < count; ++i) {
        if (tmp[i].uid == tmp[i - 1].uid) return false;
    }
    items_ = tmp;
    size_ = count;
    return true;
}

template <std::size_t N>
int Roster<N>::find(const Uid& uid) const {
    auto first = items_.begin(), last = items_.begin() + size_;
    auto it = std::lower_bound(first, last, uid, [](const Entry& e, const Uid& u) { return e.uid < u; });
    return (it != last && it->uid == uid) ? static_cast<int>(it - first) : -1;
}

template <std::size_t N>
void BoardingLedger<N>::begin_trip(std::uint32_t trip_ref) {
    trip_ref_ = trip_ref;
    state_.fill(State::Away);
    last_tap_s_.fill(0);
    board_s_.fill(0);
}

template <std::size_t N>
void BoardingLedger<N>::end_trip() {
    trip_ref_ = 0;
}

template <std::size_t N>
TapResult BoardingLedger<N>::on_tap(int idx, std::uint32_t now_s) {
    if (idx < 0 || static_cast<std::size_t>(idx) >= N) return TapResult::Unknown;
    if (!trip_running()) return TapResult::NoTrip;
    const auto i = static_cast<std::size_t>(idx);
    if (last_tap_s_[i] != 0 && now_s - last_tap_s_[i] < cfg_.dedupe_s) return TapResult::Duplicate;
    last_tap_s_[i] = now_s == 0 ? 1 : now_s;   // 0 means "never tapped"
    switch (state_[i]) {
        case State::Away:
            state_[i] = State::OnBoard;
            board_s_[i] = now_s;
            return TapResult::Board;
        case State::OnBoard:
            if (now_s - board_s_[i] < cfg_.min_ride_s) return TapResult::Duplicate;
            state_[i] = State::Done;
            return TapResult::Alight;
        case State::Done:
            // Got off and on again on the same trip (e.g. wrong stop): board again.
            state_[i] = State::OnBoard;
            board_s_[i] = now_s;
            return TapResult::Board;
    }
    return TapResult::Duplicate;
}

template <std::size_t N>
std::size_t BoardingLedger<N>::on_board() const {
    return static_cast<std::size_t>(std::count(state_.begin(), state_.end(), State::OnBoard));
}

}  // namespace sf
