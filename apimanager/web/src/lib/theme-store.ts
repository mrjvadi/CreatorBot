import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ThemeState {
  dark: boolean;
  toggle: () => void;
}

// طراحی جدید (الهام از رفرنس دارک دشبورد) اساساً برای حالت تیره ساخته شده — پس پیش‌فرض
// را روی dark گذاشتیم، نه تشخیص خودکار از prefers-color-scheme. کاربر همچنان می‌تواند
// از تاپ‌بار به حالت روشن سوییچ کند.
export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      dark: true,
      toggle: () => set({ dark: !get().dark }),
    }),
    { name: "apimanager-theme" }
  )
);
