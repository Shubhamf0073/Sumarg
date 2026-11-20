import { describe, expect, it, jest } from "@jest/globals";

jest.mock("expo-secure-store", () => ({}));

import { INSTALL_ID_RE, newInstallId } from "./install";

describe("newInstallId", () => {
  it("uses the native UUID and matches what the server accepts", () => {
    const id = newInstallId(() => "1b4e28ba-2fa1-41d2-883f-0016d3cca427");
    expect(INSTALL_ID_RE.test(id)).toBe(true);
  });

  it("refuses to run without a secure source or with a bad id", () => {
    expect(() => newInstallId(null)).toThrow();
    expect(() => newInstallId(() => "short")).toThrow();
  });
});
