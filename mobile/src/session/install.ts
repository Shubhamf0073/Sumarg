import * as SecureStore from "expo-secure-store";

const KEY = "sumarg.install";

/** What the server accepts as install_id / X-Install-Id (docs/04 "Auth"). */
export const INSTALL_ID_RE = /^[A-Za-z0-9-]{16,64}$/;

type ExpoGlobal = { expo?: { uuidv4?: () => string } };

/** A fresh random id: Expo's native UUIDv4 (the OS secure random source); no new dependency. */
export function newInstallId(
  gen: (() => string) | null | undefined = (globalThis as ExpoGlobal).expo?.uuidv4,
): string {
  if (!gen) throw new Error("No secure random source for the install id");
  const id = gen();
  if (!INSTALL_ID_RE.test(id)) throw new Error("Bad install id");
  return id;
}

let cached: string | null = null;

/**
 * The id this install binds its sessions to. Made once and kept in the keystore; a reinstall makes a new one, which
 * signs the app out (the server binds every app session to its install).
 */
export async function installId(): Promise<string> {
  if (cached) return cached;
  let id: string | null = null;
  try {
    id = await SecureStore.getItemAsync(KEY);
  } catch {
    id = null;
  }
  if (!id || !INSTALL_ID_RE.test(id)) {
    id = newInstallId();
    await SecureStore.setItemAsync(KEY, id);
  }
  cached = id;
  return id;
}
