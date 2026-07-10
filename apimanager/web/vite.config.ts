import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// apimanager هیچ CORS middleware ای ندارد، پس در dev درخواست‌های مستقیمِ مرورگر به یک
// آدرس/پورتِ دیگر (مثلاً http://localhost:8086) با خطای CORS بلاک می‌شوند. برای دور زدنش
// بدون دست‌زدن به بک‌اند، مسیر /api را از همین سرور Vite (که هم‌مبدأ صفحه است) به apimanager
// پروکسی می‌کنیم؛ آدرس واقعی apimanager را از همان VITE_API_BASE_URL در .env می‌خوانیم.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.VITE_API_BASE_URL || "http://localhost:8080";

  return {
    plugins: [react()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: 5173,
      proxy: {
        "/api": {
          target: apiTarget,
          changeOrigin: true,
        },
      },
    },
  };
});
