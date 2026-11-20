import { describe, expect, it } from "@jest/globals";
import { homeGroup, LANDING, type Session } from "./principal";

const parent: Session = { token: "t1.x", principal: "parent" };
const driver: Session = { token: "t1.y", principal: "driver" };

describe("homeGroup", () => {
  it("sends signed-out users to the auth stack", () => {
    expect(homeGroup(null)).toBe("(auth)");
  });

  it("picks the tab set from the principal", () => {
    expect(homeGroup(parent)).toBe("(parent)");
    expect(homeGroup(driver)).toBe("(driver)");
  });

  it("has a landing screen for every group", () => {
    for (const g of ["(auth)", "(parent)", "(driver)"] as const) {
      expect(LANDING[g]).toContain(g);
    }
  });
});
