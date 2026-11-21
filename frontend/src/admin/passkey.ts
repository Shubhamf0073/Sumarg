/**
 * Browser side of the admin passkey ceremonies. The API sends WebAuthn options as JSON with base64url byte fields
 * and expects the credential back the same way (PublicKeyCredential.toJSON shape). Converted by hand so it works in
 * browsers without parseCreationOptionsFromJSON / toJSON, and needs no package.
 */

type Json = Record<string, unknown>;

function fromB64url(s: string): ArrayBuffer {
  const b64 = s
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(Math.ceil(s.length / 4) * 4, "=");
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out.buffer;
}

function toB64url(buf: ArrayBuffer | null | undefined): string | undefined {
  if (!buf) return undefined;
  const bytes = new Uint8Array(buf);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function descriptors(list: unknown): PublicKeyCredentialDescriptor[] | undefined {
  if (!Array.isArray(list)) return undefined;
  return list.map(
    (d: Json) => ({ ...d, id: fromB64url(d.id as string) }) as PublicKeyCredentialDescriptor,
  );
}

export function passkeysSupported(): boolean {
  return typeof window !== "undefined" && "PublicKeyCredential" in window;
}

/** navigator.credentials.create with the enrolment begin options; returns the JSON the finish step expects. */
export async function createPasskey(options: Json): Promise<Json> {
  const pk = options.publicKey as Json;
  const user = pk.user as Json;
  const cred = (await navigator.credentials.create({
    publicKey: {
      ...(pk as unknown as PublicKeyCredentialCreationOptions),
      challenge: fromB64url(pk.challenge as string),
      user: {
        ...(user as unknown as PublicKeyCredentialUserEntity),
        id: fromB64url(user.id as string),
      },
      excludeCredentials: descriptors(pk.excludeCredentials),
    },
  })) as PublicKeyCredential | null;
  if (!cred) throw new Error("No passkey was created.");
  const r = cred.response as AuthenticatorAttestationResponse;
  return {
    id: cred.id,
    rawId: toB64url(cred.rawId),
    type: cred.type,
    authenticatorAttachment: cred.authenticatorAttachment ?? undefined,
    clientExtensionResults: cred.getClientExtensionResults(),
    response: {
      clientDataJSON: toB64url(r.clientDataJSON),
      attestationObject: toB64url(r.attestationObject),
      transports: r.getTransports?.() ?? [],
    },
  };
}

/** navigator.credentials.get with the sign-in begin options; returns the JSON the finish step expects. */
export async function getPasskey(options: Json): Promise<Json> {
  const pk = options.publicKey as Json;
  const cred = (await navigator.credentials.get({
    publicKey: {
      ...(pk as unknown as PublicKeyCredentialRequestOptions),
      challenge: fromB64url(pk.challenge as string),
      allowCredentials: descriptors(pk.allowCredentials),
    },
  })) as PublicKeyCredential | null;
  if (!cred) throw new Error("No passkey was chosen.");
  const r = cred.response as AuthenticatorAssertionResponse;
  return {
    id: cred.id,
    rawId: toB64url(cred.rawId),
    type: cred.type,
    authenticatorAttachment: cred.authenticatorAttachment ?? undefined,
    clientExtensionResults: cred.getClientExtensionResults(),
    response: {
      clientDataJSON: toB64url(r.clientDataJSON),
      authenticatorData: toB64url(r.authenticatorData),
      signature: toB64url(r.signature),
      userHandle: toB64url(r.userHandle),
    },
  };
}

/** The browser's own messages for a cancelled or refused prompt are cryptic; say it plainly. */
export function passkeyMessage(e: unknown, fallback: string): string {
  if (e instanceof DOMException && (e.name === "NotAllowedError" || e.name === "AbortError")) {
    return "The passkey prompt was cancelled or timed out. Please try again.";
  }
  if (e instanceof DOMException && e.name === "InvalidStateError") {
    return "This device already has a passkey for your account. Sign in instead.";
  }
  return e instanceof Error && e.message ? e.message : fallback;
}
