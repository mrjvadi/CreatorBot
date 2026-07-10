import { lazy, Suspense, useEffect } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { ProtectedRoute, AdminRoute } from "@/components/ProtectedRoute";
import AppShell from "@/components/Layout/AppShell";
import Login from "@/pages/Login";
import { useThemeStore } from "@/lib/theme-store";

// صفحاتی که فقط بعد از لاگین لازم می‌شوند lazy هستند تا bundle اولیه (خصوصاً صفحه‌ی لاگین)
// سبک بماند — مخصوصاً AdminStats که recharts (کتابخانه‌ی نسبتاً سنگین) را import می‌کند.
const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Instances = lazy(() => import("@/pages/Instances"));
const Plans = lazy(() => import("@/pages/Plans"));
const Payments = lazy(() => import("@/pages/Payments"));
const AdminStats = lazy(() => import("@/pages/admin/AdminStats"));
const AdminServers = lazy(() => import("@/pages/admin/AdminServers"));
const AdminTemplates = lazy(() => import("@/pages/admin/AdminTemplates"));
const AdminUsers = lazy(() => import("@/pages/admin/AdminUsers"));
const AdminPlans = lazy(() => import("@/pages/admin/AdminPlans"));
const AdminInstances = lazy(() => import("@/pages/admin/AdminInstances"));
const AdminPayments = lazy(() => import("@/pages/admin/AdminPayments"));
const AdminPromoCodes = lazy(() => import("@/pages/admin/AdminPromoCodes"));
const AdminImages = lazy(() => import("@/pages/admin/AdminImages"));
const RequestLogs = lazy(() => import("@/pages/RequestLogs"));
const NotFound = lazy(() => import("@/pages/NotFound"));

function PageFallback() {
  return (
    <div className="flex h-[60vh] items-center justify-center">
      <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
    </div>
  );
}

export default function App() {
  // کلاس dark باید همیشه فعال باشد (حتی روی صفحه‌ی لاگین، قبل از رندر شدنِ AppShell)،
  // پس این‌جا در بالاترین سطح ممکن سینک می‌شود، نه فقط داخل AppShell.
  const dark = useThemeStore((s) => s.dark);
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
  }, [dark]);

  return (
    <Suspense fallback={<PageFallback />}>
      <div className="app-dark-glow" aria-hidden="true" />
      <Routes>
        <Route path="/" element={<Navigate to="/app" replace />} />
        <Route path="/login" element={<Login />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/app" element={<AppShell variant="user" />}>
            <Route index element={<Dashboard />} />
            <Route path="instances" element={<Instances />} />
            <Route path="plans" element={<Plans />} />
            <Route path="payments" element={<Payments />} />
          </Route>

          <Route element={<AdminRoute />}>
            <Route path="/admin" element={<AppShell variant="admin" />}>
              <Route index element={<AdminStats />} />
              <Route path="instances" element={<AdminInstances />} />
              <Route path="servers" element={<AdminServers />} />
              <Route path="templates" element={<AdminTemplates />} />
              <Route path="plans" element={<AdminPlans />} />
              <Route path="users" element={<AdminUsers />} />
              <Route path="payments" element={<AdminPayments />} />
              <Route path="promo-codes" element={<AdminPromoCodes />} />
              <Route path="images" element={<AdminImages />} />
              <Route path="request-logs" element={<RequestLogs />} />
            </Route>
          </Route>
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
    </Suspense>
  );
}
