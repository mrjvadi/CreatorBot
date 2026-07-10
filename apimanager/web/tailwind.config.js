/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Vazirmatn", "system-ui", "sans-serif"],
      },
      colors: {
        // پالت اصلی (فروشگاهی/آبی) — همچنان برای حالت روشن نگه داشته شده.
        brand: {
          50: "#eef6ff",
          100: "#d9ecff",
          200: "#bcdcff",
          300: "#8ec5ff",
          400: "#59a5ff",
          500: "#3182f6",
          600: "#1f63e0",
          700: "#194fb5",
          800: "#1a4392",
          900: "#1c3a75",
        },
        // پالت گرادیانی بنفش/صورتی — الهام‌گرفته از رفرنس تیره؛ حالا تم اصلیِ dark mode است.
        violet: {
          50: "#f4f1ff",
          100: "#ebe4ff",
          200: "#d5c8ff",
          300: "#b49bff",
          400: "#9166ff",
          500: "#7c3aed",
          600: "#6d28d9",
          700: "#5b21b6",
          800: "#4c1d95",
          900: "#3b1878",
        },
        fuchsia2: {
          400: "#e879f9",
          500: "#d946ef",
          600: "#c026d3",
        },
        success: {
          50: "#ecfdf5",
          400: "#34d399",
          500: "#10b981",
          600: "#059669",
          700: "#047857",
        },
        warning: {
          50: "#fffbeb",
          400: "#fbbf24",
          500: "#f59e0b",
          600: "#d97706",
          700: "#b45309",
        },
        danger: {
          50: "#fef2f2",
          400: "#f87171",
          500: "#ef4444",
          600: "#dc2626",
          700: "#b91c1c",
        },
        // زمینه‌ی خیلی تیره‌ی سرمه‌ای/بنفش (نه خاکستری خالص) — مطابق رفرنس.
        ink: {
          950: "#0a0713",
          900: "#0f0b1e",
          800: "#171226",
          700: "#211a35",
          600: "#2c2347",
        },
      },
      fontSize: {
        xs: ["0.75rem", { lineHeight: "1.1rem" }],
        sm: ["0.8125rem", { lineHeight: "1.25rem" }],
      },
      backgroundImage: {
        "brand-gradient": "linear-gradient(135deg, #7c3aed 0%, #c026d3 55%, #ec4899 100%)",
        "brand-gradient-soft": "linear-gradient(135deg, rgba(124,58,237,0.18) 0%, rgba(192,38,211,0.14) 55%, rgba(236,72,153,0.12) 100%)",
        "glow-radial": "radial-gradient(60% 50% at 50% 0%, rgba(124,58,237,0.25) 0%, rgba(10,7,19,0) 70%)",
      },
      boxShadow: {
        card: "0 1px 2px 0 rgb(0 0 0 / 0.04), 0 1px 3px 0 rgb(0 0 0 / 0.06)",
        popover: "0 4px 6px -1px rgb(0 0 0 / 0.08), 0 10px 20px -4px rgb(0 0 0 / 0.08)",
        glass: "0 1px 1px 0 rgb(255 255 255 / 0.06) inset, 0 8px 24px -8px rgb(0 0 0 / 0.5)",
        glow: "0 0 0 1px rgb(124 58 237 / 0.25), 0 8px 30px -6px rgb(124 58 237 / 0.35)",
      },
      spacing: {
        18: "4.5rem",
      },
    },
  },
  plugins: [],
};
