/** Money is integer paise everywhere; this is the only place it becomes a ₹ string. */
export function rupees(paise: number): string {
  const sign = paise < 0 ? "-" : "";
  const abs = Math.abs(Math.trunc(paise));
  const whole = Math.floor(abs / 100).toLocaleString("en-IN");
  const frac = abs % 100;
  return `${sign}₹${whole}${frac ? `.${String(frac).padStart(2, "0")}` : ""}`;
}
