import { describe, expect, it, jest } from "@jest/globals";

jest.mock("expo-notifications", () => ({}));
jest.mock("expo-secure-store", () => ({}));

import { CODE_RE, MOBILE_RE, normalise, SCHOOL_CODE_RE, signInErrorKey } from "./auth";

describe("signInErrorKey", () => {
  it("maps statuses to one line each, without saying whether the mobile is known", () => {
    expect(signInErrorKey(401, "bad_credentials")).toBe("auth.err.badCode");
    expect(signInErrorKey(429, "rate_limited")).toBe("auth.err.tooMany");
    expect(signInErrorKey(423, "school_closed")).toBe("auth.err.schoolClosed");
    expect(signInErrorKey(422)).toBe("auth.err.invalid");
    expect(signInErrorKey(500)).toBe("auth.err.network");
  });
});

describe("normalise", () => {
  it.each([
    ["98765 43210", "9876543210"],
    ["+91 98765-43210", "9876543210"],
    ["09876543210", "9876543210"],
    ["9876543210", "9876543210"],
  ])("mobile %s", (raw, want) => {
    const n = normalise({ school_code: "  DPS-Bopal ", mobile: raw });
    expect(n).toEqual({ school_code: "dps-bopal", mobile: want });
    expect(SCHOOL_CODE_RE.test(n.school_code) && MOBILE_RE.test(n.mobile)).toBe(true);
  });

  it("does not invent a valid mobile", () => {
    expect(MOBILE_RE.test(normalise({ school_code: "x", mobile: "12345" }).mobile)).toBe(false);
    expect(MOBILE_RE.test(normalise({ school_code: "x", mobile: "5876543210" }).mobile)).toBe(
      false,
    );
  });

  it("checks codes", () => {
    expect(CODE_RE.test("012345")).toBe(true);
    expect(CODE_RE.test("12345")).toBe(false);
  });
});
