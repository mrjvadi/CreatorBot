import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Globe, Check } from "lucide-react";
import clsx from "clsx";
import { SUPPORTED_LANGUAGES, changeLanguage } from "@/i18n";

export function LanguageSwitcher({ compact }: { compact?: boolean }) {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const current = SUPPORTED_LANGUAGES.find((l) => l.code === i18n.language) ?? SUPPORTED_LANGUAGES[0];

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onClickOutside);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClickOutside);
      document.removeEventListener("keydown", onKey);
    };
  }, []);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={t("nav.language")}
        aria-haspopup="menu"
        aria-expanded={open}
        className={clsx(
          "flex items-center gap-1.5 rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800",
          compact ? "p-2" : "px-2.5 py-2 text-xs font-medium"
        )}
      >
        <Globe className="h-4 w-4" />
        {!compact && <span>{current.label}</span>}
      </button>

      {open && (
        <div
          role="menu"
          className="absolute end-0 z-40 mt-1.5 w-36 animate-fade-in rounded-xl border border-slate-200 bg-white py-1 shadow-popover dark:border-slate-700 dark:bg-slate-900"
        >
          {SUPPORTED_LANGUAGES.map((lang) => (
            <button
              key={lang.code}
              role="menuitem"
              onClick={() => {
                changeLanguage(lang.code);
                setOpen(false);
              }}
              className="flex w-full items-center justify-between px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 dark:text-slate-200 dark:hover:bg-slate-800"
            >
              {lang.label}
              {lang.code === i18n.language && <Check className="h-3.5 w-3.5 text-brand-600" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
