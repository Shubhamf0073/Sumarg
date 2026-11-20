// Host tests for the boarding ledger (docs/14 §5). Run: pio test -e native
#include <unity.h>

#include "../../src/logic/boarding.h"

using namespace sf;

static Uid uid4(std::uint8_t a, std::uint8_t b, std::uint8_t c, std::uint8_t d) {
    Uid u;
    u.b = {a, b, c, d};
    u.len = 4;
    return u;
}

void setUp() {}
void tearDown() {}

static void test_roster_find_and_reject() {
    Roster<8>::Entry e[3] = {{uid4(9, 9, 9, 9), "Zara"}, {uid4(1, 2, 3, 4), "Aarav"}, {uid4(5, 6, 7, 8), "Diya"}};
    Roster<8> r;
    TEST_ASSERT_TRUE(r.load(e, 3));
    TEST_ASSERT_EQUAL_INT(0, r.find(uid4(1, 2, 3, 4)));
    TEST_ASSERT_EQUAL_STRING("Zara", r.at(static_cast<std::size_t>(r.find(uid4(9, 9, 9, 9)))).short_name);
    TEST_ASSERT_EQUAL_INT(-1, r.find(uid4(0, 0, 0, 1)));
    Roster<8>::Entry dup[2] = {{uid4(1, 1, 1, 1), "A"}, {uid4(1, 1, 1, 1), "B"}};
    TEST_ASSERT_FALSE(r.load(dup, 2));
    TEST_ASSERT_EQUAL_UINT(3, r.size());             // old roster kept
    Roster<2> small;
    TEST_ASSERT_FALSE(small.load(e, 3));             // over capacity
}

static void test_board_then_alight() {
    BoardingLedger<8> l;
    l.begin_trip(42);
    TEST_ASSERT_EQUAL(TapResult::Board, l.on_tap(0, 1000));
    TEST_ASSERT_EQUAL_UINT(1, l.on_board());
    TEST_ASSERT_EQUAL(TapResult::Duplicate, l.on_tap(0, 1002));     // double tap
    TEST_ASSERT_EQUAL(TapResult::Duplicate, l.on_tap(0, 1060));     // < min ride
    TEST_ASSERT_EQUAL(TapResult::Alight, l.on_tap(0, 1900));
    TEST_ASSERT_EQUAL_UINT(0, l.on_board());
    TEST_ASSERT_EQUAL(TapResult::Board, l.on_tap(0, 1910));         // got back on
}

static void test_unknown_and_no_trip() {
    BoardingLedger<8> l;
    TEST_ASSERT_EQUAL(TapResult::NoTrip, l.on_tap(0, 10));
    l.begin_trip(7);
    TEST_ASSERT_EQUAL(TapResult::Unknown, l.on_tap(-1, 20));
    TEST_ASSERT_EQUAL(TapResult::Unknown, l.on_tap(8, 20));          // out of range
    TEST_ASSERT_EQUAL(TapResult::Board, l.on_tap(3, 20));
    l.begin_trip(8);                                                  // new trip clears state
    TEST_ASSERT_EQUAL_UINT(0, l.on_board());
    TEST_ASSERT_EQUAL(TapResult::Board, l.on_tap(3, 25));
}

static void test_time_zero_is_not_never() {
    BoardingLedger<4> l;
    l.begin_trip(1);
    TEST_ASSERT_EQUAL(TapResult::Board, l.on_tap(1, 0));
    TEST_ASSERT_EQUAL(TapResult::Duplicate, l.on_tap(1, 1));
}

int main() {
    UNITY_BEGIN();
    RUN_TEST(test_roster_find_and_reject);
    RUN_TEST(test_board_then_alight);
    RUN_TEST(test_unknown_and_no_trip);
    RUN_TEST(test_time_zero_is_not_never);
    return UNITY_END();
}
