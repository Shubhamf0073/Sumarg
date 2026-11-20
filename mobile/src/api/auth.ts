import { useMutation } from "@tanstack/react-query";
import * as Notifications from "expo-notifications";
import { Platform } from "react-native";
import appJson from "../../app.json";
import type { StringKey } from "../i18n";
import { installId } from "../session/install";
import { saveSession } from "../session/store";
import { api } from "./client";

/** A sign-in failure the screen shows as one translated line. */
export class SignInError extends Error {
  constructor(readonly key: StringKey) {
    super(key);
  }
}

/** Maps a sign-in response to the message the parent sees. No message says whether the mobile is known. */
export function signInErrorKey(status: number, code?: string): StringKey {
  if (status === 429 || code === "rate_limited") return "auth.err.tooMany";
  if (status === 423 || code === "school_closed") return "auth.err.schoolClosed";
  if (status === 401) return "auth.err.badCode";
  if (status === 422 || status === 400) return "auth.err.invalid";
  return "auth.err.network";
}

export type CodeRequest = { school_code: string; mobile: string };

/** Normalises what the parent typed: school code trimmed and lower-cased, mobile without spaces or +91 / leading 0. */
export function normalise(input: CodeRequest): CodeRequest {
  const digits = input.mobile.replace(/\D/g, "");
  const mobile = digits.length > 10 ? digits.replace(/^(91|0)/, "") : digits;
  return { school_code: input.school_code.trim().toLowerCase(), mobile };
}

export const SCHOOL_CODE_RE = /^[a-z0-9-]{3,30}$/;
export const MOBILE_RE = /^[6-9][0-9]{9}$/;
export const CODE_RE = /^[0-9]{6}$/;

async function fail(response: Response, error: unknown): Promise<never> {
  const code = (error as { code?: string } | undefined)?.code;
  throw new SignInError(signInErrorKey(response.status, code));
}

/** Step 1: ask for an SMS code. The server answers 202 for every mobile, so success says nothing about the number. */
export function useRequestCode() {
  return useMutation({
    mutationFn: async (body: CodeRequest) => {
      try {
        const { error, response } = await api.POST("/api/auth/app/otp", { body });
        if (response.status !== 202) await fail(response, error);
      } catch (e) {
        throw e instanceof SignInError ? e : new SignInError("auth.err.network");
      }
    },
  });
}

/** Step 2: the code → a parent session, kept in the keystore. The parent tabs then register for push. */
export function useVerifyCode() {
  return useMutation({
    mutationFn: async (input: CodeRequest & { code: string }) => {
      try {
        const { data, error, response } = await api.POST("/api/auth/app/otp/verify", {
          body: { ...input, install_id: await installId() },
        });
        if (!data) return await fail(response, error);
        await saveSession({ token: data.token, principal: "parent" });
      } catch (e) {
        throw e instanceof SignInError ? e : new SignInError("auth.err.network");
      }
    },
  });
}

/**
 * Registers this install's device token (FCM on Android, APNs on iOS) for the signed-in session. Never the Expo push
 * service (docs/02 A10). Silent on refusal or failure: the app works without push, and it is retried on next sign-in.
 */
export async function registerPushToken(): Promise<void> {
  try {
    const perm = await Notifications.requestPermissionsAsync();
    if (!perm.granted) return;
    const token = await Notifications.getDevicePushTokenAsync();
    if (typeof token.data !== "string") return;
    await api.POST("/api/auth/push-token", {
      body: {
        platform: Platform.OS === "ios" ? "ios" : "android",
        fcm_token: token.data,
        app_version: appJson.expo.version,
      },
    });
  } catch {
    // no permission, no Play services, offline: try again next time
  }
}
