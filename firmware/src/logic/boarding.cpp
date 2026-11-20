// boarding.h is header-only (templates). This translation unit makes sure the header compiles on its own in the
// native build and holds the instantiation used by the firmware.
#include "boarding.h"
#include "../../include/build_config.h"

template class sf::Roster<MAX_ROSTER>;
template class sf::BoardingLedger<MAX_ROSTER>;
