import { getLocales } from "expo-localization";
import en from "./en.json";
import gu from "./gu.json";
import hi from "./hi.json";

export type StringKey = keyof typeof en;
export type Lang = "en" | "gu" | "hi";

const TABLES: Record<Lang, Record<string, string>> = { en, gu, hi };

export function pickLang(code: string | null | undefined): Lang {
  return code === "gu" || code === "hi" ? code : "en";
}

let lang: Lang = "en";

/** Device language by default; call with a profile choice once one exists. */
export function initLang(override?: Lang): void {
  lang = override ?? pickLang(getLocales()[0]?.languageCode);
}

export function t(key: StringKey): string {
  return TABLES[lang][key] ?? en[key];
}
