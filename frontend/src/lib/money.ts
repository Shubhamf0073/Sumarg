/** Money is integer paise everywhere; this is the only place it becomes a ₹ string. */
export function rupees(paise: number): string {
  const sign = paise < 0 ? "-" : "";
  const abs = Math.abs(Math.trunc(paise));
  const whole = Math.floor(abs / 100).toLocaleString("en-IN");
  const frac = abs % 100;
  return `${sign}₹${whole}${frac ? `.${String(frac).padStart(2, "0")}` : ""}`;
}

/** A ₹ amount typed in a form ("1000", "1,000.50") → integer paise; null if it isn't one. */
export function paiseFromRupees(input: string): number | null {
  const m = /^(\d{1,9})(?:\.(\d{1,2}))?$/.exec(input.replaceAll(",", "").trim());
  if (!m) return null;
  return Number(m[1]) * 100 + Number((m[2] ?? "").padEnd(2, "0"));
}
