import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import fa from "./locales/fa.json";
import en from "./locales/en.json";

export const SUPPORTED_LANGUAGES = [
  { code: "fa", label: "فارسی", dir: "rtl" as const },
  { code: "en", label: "English", dir: "ltr" as const },
];

const RTL_LANGUAGES = new Set(["fa", "ar", "he", "ur"]);

const STORAGE_KEY = "apimanager-language";

export function dirForLanguage(lang: string): "rtl" | "ltr" {
  return RTL_LANGUAGES.has(lang) ? "rtl" : "ltr";
}

function getInitialLanguage(): string {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && SUPPORTED_LANGUAGES.some((l) => l.code === stored)) return stored;
  } catch {
    /* localStorage may be unavailable (privacy mode) — fall back silently */
  }
  return "fa";
}

const initialLanguage = getInitialLanguage();

i18n.use(initReactI18next).init({
  resources: {
    fa: { translation: fa },
    en: { translation: en },
  },
  lng: initialLanguage,
  fallbackLng: "fa",
  interpolation: { escapeValue: false },
  returnEmptyString: false,
});

// جلوگیری از فلش جهت اشتباه: قبل از رندر اول، dir/lang سند را هم‌زمان با زبان ذخیره‌شده تنظیم کن.
document.documentElement.dir = dirForLanguage(initialLanguage);
document.documentElement.lang = initialLanguage;

export function changeLanguage(lang: string) {
  i18n.changeLanguage(lang);
  document.documentElement.dir = dirForLanguage(lang);
  document.documentElement.lang = lang;
  try {
    localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    /* ignore */
  }
}

export default i18n;
