import * as SecureStore from "expo-secure-store";
import { useSyncExternalStore } from "react";
import type { Session } from "./principal";

const KEY = "sumarg.session";

type State = { ready: boolean; session: Session | null };

let state: State = { ready: false, session: null };
const listeners = new Set<() => void>();

function set(next: State) {
  state = next;
  for (const l of listeners) l();
}

function parse(raw: string | null): Session | null {
  if (!raw) return null;
  try {
    const v = JSON.parse(raw) as Partial<Session>;
    if (typeof v.token === "string" && (v.principal === "parent" || v.principal === "driver")) {
      return { token: v.token, principal: v.principal };
    }
  } catch {
    // corrupt value: treat as signed out
  }
  return null;
}

/** Load the stored session once at startup. */
export async function loadSession(): Promise<void> {
  let session: Session | null = null;
  try {
    session = parse(await SecureStore.getItemAsync(KEY));
  } catch {
    session = null;
  }
  set({ ready: true, session });
}

export async function saveSession(session: Session): Promise<void> {
  await SecureStore.setItemAsync(KEY, JSON.stringify(session));
  set({ ready: true, session });
}

export async function clearSession(): Promise<void> {
  set({ ready: true, session: null });
  try {
    await SecureStore.deleteItemAsync(KEY);
  } catch {
    // nothing stored or store unavailable
  }
}

export function currentToken(): string | null {
  return state.session?.token ?? null;
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function useSession(): State {
  return useSyncExternalStore(subscribe, () => state);
}
