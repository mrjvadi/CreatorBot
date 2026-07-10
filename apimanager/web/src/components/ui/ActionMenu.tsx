import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { MoreVertical } from "lucide-react";
import clsx from "clsx";

export interface ActionMenuItem {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  onClick: () => void;
  danger?: boolean;
  disabled?: boolean;
}

const MENU_WIDTH = 176; // معادل w-44

/**
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «تو قسمت همه‌ی ربات‌ها، منوی سه‌نقطه می‌ره زیر چیزهای پایینش، اصلاً
 * بالا نمایش داده نمی‌شه». علت: DataTable سطرها را داخل یک `overflow-x-auto` می‌گذارد؛ طبق
 * مشخصات CSS، وقتی overflow-x چیزی غیر از visible باشد، overflow-y هم عملاً auto محاسبه
 * می‌شود (نه visible) — یعنی هر چیزِ absolute-positioned که از پایین جدول بیرون بزند (مثل
 * منوی سه‌نقطه‌ی سطرهای پایینی) توسط همان container کراپ می‌شود، حتی با z-index بالا (چون
 * کراپ‌شدن به‌خاطر overflow است، نه استکینگ). راه‌حل: به‌جای absolute-positioning داخل جدول،
 * خودِ منو را با یک React Portal مستقیم به document.body می‌بریم و موقعیتش را با
 * getBoundingClientRect دکمه محاسبه می‌کنیم — دیگر هیچ ancestor ای نمی‌تواند کراپش کند.
 */
export function ActionMenu({ items, ariaLabel }: { items: ActionMenuItem[]; ariaLabel: string }) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<{ top?: number; bottom?: number; left: number } | null>(null);
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    function onClick(e: MouseEvent) {
      const target = e.target as Node;
      if (menuRef.current?.contains(target)) return;
      if (btnRef.current?.contains(target)) return;
      setOpen(false);
    }
    function onDismiss() {
      setOpen(false);
    }

    document.addEventListener("mousedown", onClick);
    // اسکرول/تغییر اندازه یعنی موقعیتِ ذخیره‌شده دیگر معتبر نیست — به‌جای دنبال‌کردن، فقط می‌بندیم.
    window.addEventListener("scroll", onDismiss, true);
    window.addEventListener("resize", onDismiss);
    return () => {
      document.removeEventListener("mousedown", onClick);
      window.removeEventListener("scroll", onDismiss, true);
      window.removeEventListener("resize", onDismiss);
    };
  }, [open]);

  function toggle(e: React.MouseEvent) {
    e.stopPropagation();
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      const spaceBelow = window.innerHeight - rect.bottom;
      const openUpward = spaceBelow < 220 && rect.top > 220;
      const left = Math.min(Math.max(8, rect.right - MENU_WIDTH), window.innerWidth - MENU_WIDTH - 8);
      setPos(
        openUpward
          ? { bottom: window.innerHeight - rect.top + 4, left }
          : { top: rect.bottom + 4, left }
      );
    }
    setOpen((v) => !v);
  }

  return (
    <>
      <button
        ref={btnRef}
        onClick={toggle}
        aria-label={ariaLabel}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
      >
        <MoreVertical className="h-4 w-4" />
      </button>
      {open &&
        pos &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            onClick={(e) => e.stopPropagation()}
            style={{ position: "fixed", top: pos.top, bottom: pos.bottom, left: pos.left, width: MENU_WIDTH }}
            className="glass-card z-[60] animate-fade-in py-1 shadow-popover"
          >
            {items.map((item) => (
              <button
                key={item.label}
                role="menuitem"
                disabled={item.disabled}
                onClick={() => {
                  setOpen(false);
                  item.onClick();
                }}
                className={clsx(
                  "flex w-full items-center gap-2 px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50",
                  item.danger
                    ? "text-red-600 hover:bg-red-50 dark:text-fuchsia-300 dark:hover:bg-fuchsia-500/10"
                    : "text-slate-700 hover:bg-slate-50 dark:text-slate-200 dark:hover:bg-white/10"
                )}
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </button>
            ))}
          </div>,
          document.body
        )}
    </>
  );
}
