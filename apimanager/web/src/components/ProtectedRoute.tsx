import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/lib/auth-store";

export function ProtectedRoute() {
  const isAuthed = useAuthStore((s) => !!s.accessToken || !!s.refreshToken);
  if (!isAuthed) return <Navigate to="/login" replace />;
  return <Outlet />;
}

export function AdminRoute() {
  const isAdmin = useAuthStore((s) => s.isAdmin());
  if (!isAdmin) return <Navigate to="/app" replace />;
  return <Outlet />;
}
