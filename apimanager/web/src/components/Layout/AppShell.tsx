import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  LayoutDashboard,
  Bot,
  CreditCard,
  ServerCog,
  LayoutTemplate,
  Gauge,
  Sun,
  Moon,
  LogOut,
  Menu,
  X,
  Activity,
  Bell,
  ChevronDown,
  Users,
  ArrowLeftRight,
  Receipt,
  Ticket,
  HardDrive,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import clsx from "clsx";
import { api, unwrap } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";
import { useThemeStore } from "@/lib/theme-store";
import { useSidebarStore } from "@/lib/sidebar-store";
import { useRequestLogStore } from "@/lib/request-log-store";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import type { Me } from "@/lib/types";

type NavItem = { to: string; labelKey: string; icon: typeof LayoutDashboard; end?: boolean };

// «لاگ درخواست‌ها» عمداً این‌جا نیست — یک ابزار دیباگ برای توسعه‌دهنده است، نه چیزی که یک
// مشتری واقعی به آن نیاز داشته باشد؛ فقط در ناوبری ادمین می‌ماند (رجوع به بازخورد کاربر
// ۲۰۲۶-۰۷-۰۳: چیزهای بی‌ربط به کاربر نباید نشانش داد).
const userNav: NavItem[] = [
  { to: "/app", labelKey: "nav.dashboard", icon: LayoutDashboard, end: true },
  { to: "/app/instances", labelKey: "nav.instances", icon: Bot },
  { to: "/app/plans", labelKey: "nav.plans", icon: CreditCard },
  { to: "/app/payments", labelKey: "nav.payments", icon: Receipt },
];

const adminNav: NavItem[] = [
  { to: "/admin", labelKey: "nav.statsOverview", icon: Gauge, end: true },
  { to: "/admin/instances", labelKey: "nav.instancesAdmin", icon: Bot },
  { to: "/admin/servers", labelKey: "nav.servers", icon: ServerCog },
  { to: "/admin/templates", labelKey: "nav.templates", icon: LayoutTemplate },
  { to: "/admin/images", labelKey: "nav.images", icon: HardDrive },
  { to: "/admin/plans", labelKey: "nav.plansAdmin", icon: CreditCard },
  { to: "/admin/payments", labelKey: "nav.paymentsAdmin", icon: Receipt },
  { to: "/admin/promo-codes", labelKey: "nav.promoCodes", icon: Ticket },
  { to: "/admin/users", labelKey: "nav.users", icon: Users },
  { to: "/admin/request-logs", labelKey: "nav.requestLogs", icon: Activity },
];

function initialsOf(name?: string): string {
  if (!name) return "?";
  return name.trim().slice(0, 2).toUpperCase();
}

/**
 * چیدمانِ تازه (۲۰۲۶-۰۷-۰۳، دومین بازطراحی): با اضافه‌شدن پرداختی‌ها/کدهای تخفیف تعداد
 * آیتم‌های ناوبری ادمین به ۹ رسید و نوار تبِ بالا (segmented pill) دیگر جا نمی‌شد و شلوغ شده
 * بود — طبق درخواست کاربر، ناوبری برگشت به یک ستون کناری (sidebar) شیشه‌ای، با قابلیت جمع‌شدن
 * به حالت فقط-آیکون (وضعیتش در `sidebar-store` که از قبل بود ولی بلااستفاده مانده بود). سمتِ
 * قرارگیری از `start` استفاده می‌کند تا در RTL خودکار به راست برود، بدون هیچ کلاس جهت‌دارِ
 * دستی. نوار بالا فقط برای کنترل‌های سراسری (زبان/تم/کاربر/سوییچ ادمین) باقی مانده است.
 */
export default function AppShell({ variant }: { variant: "user" | "admin" }) {
  const { t } = useTranslation();
  const nav = variant === "admin" ? adminNav : userNav;
  const { dark, toggle } = useThemeStore();
  const { collapsed, toggle: toggleCollapsed } = useSidebarStore();
  const logout = useAuthStore((s) => s.logout);
  const isAdmin = useAuthStore((s) => s.isAdmin());
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);

  const failedCount = useRequestLogStore((s) => s.entries.filter((e) => !e.ok).length);

  const { data: me } = useQuery({
    queryKey: ["me"],
    queryFn: async () => unwrap<Me>(await api.get("/me")),
    staleTime: 60_000,
  });

  function handleLogout() {
    logout();
    navigate("/login", { replace: true });
  }

  const isActiveItem = (item: NavItem) =>
    item.end ? location.pathname === item.to : location.pathname.startsWith(item.to);

  function NavLinks({ showLabels }: { showLabels: boolean }) {
    return (
      <>
        {nav.map((item) => {
          const active = isActiveItem(item);
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              onClick={() => setMobileOpen(false)}
              title={showLabels ? undefined : t(item.labelKey)}
              className={clsx(
                "flex items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
                !showLabels && "justify-center px-0",
                active
                  ? "bg-brand-gradient text-white shadow-glow"
                  : "text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-white/10"
              )}
            >
              <item.icon className="h-4.5 w-4.5 shrink-0" />
              {showLabels && <span className="truncate">{t(item.labelKey)}</span>}
            </NavLink>
          );
        })}
      </>
    );
  }

  return (
    <div className="min-h-screen">
      {/* سایدبار — دسکتاپ */}
      <aside
        className={clsx(
          "glass-card fixed inset-y-3 start-3 z-30 hidden flex-col p-3 transition-[width] duration-200 lg:flex",
          collapsed ? "w-[4.5rem]" : "w-64"
        )}
      >
        <div className={clsx("flex items-center gap-2 px-1 pb-4", collapsed && "justify-center px-0")}>
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-brand-gradient text-sm font-bold text-white shadow-glow">
            C
          </div>
          {!collapsed && <span className="truncate font-semibold">CreatorBot</span>}
        </div>

        <nav aria-label="ناوبری اصلی" className="flex-1 space-y-1 overflow-y-auto">
          <NavLinks showLabels={!collapsed} />
        </nav>

        <button
          onClick={toggleCollapsed}
          aria-label={t(collapsed ? "nav.expandSidebar" : "nav.collapseSidebar")}
          title={t(collapsed ? "nav.expandSidebar" : "nav.collapseSidebar")}
          className={clsx(
            "mt-2 flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10",
            collapsed && "justify-center px-0"
          )}
        >
          {collapsed ? <PanelLeftOpen className="h-4.5 w-4.5 shrink-0" /> : <PanelLeftClose className="h-4.5 w-4.5 shrink-0" />}
          {!collapsed && <span>{t("nav.collapseSidebar")}</span>}
        </button>
      </aside>

      {/* سایدبار — موبایل (روی صفحه شناور) */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div className="absolute inset-0 bg-black/50" onClick={() => setMobileOpen(false)} />
          <aside className="glass-card absolute inset-y-3 start-3 flex w-64 max-w-[80vw] flex-col p-3">
            <div className="mb-2 flex items-center justify-between px-1">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-brand-gradient text-sm font-bold text-white shadow-glow">
                  C
                </div>
                <span className="font-semibold">CreatorBot</span>
              </div>
              <button
                onClick={() => setMobileOpen(false)}
                aria-label={t("nav.closeMenu")}
                className="rounded-lg p-1.5 hover:bg-slate-100 dark:hover:bg-white/10"
              >
                <X className="h-4.5 w-4.5" />
              </button>
            </div>
            <nav className="flex-1 space-y-1 overflow-y-auto">
              <NavLinks showLabels />
            </nav>
            {isAdmin && (
              <button
                onClick={() => {
                  setMobileOpen(false);
                  navigate(variant === "admin" ? "/app" : "/admin");
                }}
                className="mt-2 flex w-full items-center justify-center gap-1.5 rounded-xl border border-slate-200 px-3 py-2 text-xs font-medium text-slate-600 dark:border-white/10 dark:text-slate-300"
              >
                <ArrowLeftRight className="h-3.5 w-3.5" />
                {t(variant === "admin" ? "nav.goToUserPanel" : "nav.goToAdminPanel")}
              </button>
            )}
          </aside>
        </div>
      )}

      {/* ستونِ محتوا — با margin-inline-start متناسب با عرضِ سایدبار (که با dir خودکار جابه‌جا می‌شود) */}
      <div
        className={clsx(
          "min-h-screen p-3 transition-[margin] duration-200 sm:p-5",
          collapsed ? "lg:ms-[4.5rem]" : "lg:ms-64"
        )}
      >
        <header className="glass-card sticky top-3 z-20 flex items-center gap-2 px-3 py-2.5 sm:px-4">
          <button
            onClick={() => setMobileOpen(true)}
            aria-label={t("nav.openMenu")}
            className="flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1.5 text-sm text-slate-500 dark:bg-white/5 dark:text-slate-400 lg:hidden"
          >
            <Menu className="h-4 w-4" />
            {t(nav.find(isActiveItem)?.labelKey ?? nav[0].labelKey)}
          </button>

          <span className="hidden text-sm font-semibold text-slate-500 dark:text-slate-400 lg:inline">
            {t(nav.find(isActiveItem)?.labelKey ?? nav[0].labelKey)}
          </span>

          <div className="flex flex-1 items-center justify-end gap-0.5">
            {isAdmin && (
              <button
                onClick={() => navigate(variant === "admin" ? "/app" : "/admin")}
                title={t(variant === "admin" ? "nav.goToUserPanel" : "nav.goToAdminPanel")}
                aria-label={t(variant === "admin" ? "nav.goToUserPanel" : "nav.goToAdminPanel")}
                className="hidden items-center gap-1.5 rounded-full border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10 sm:flex"
              >
                <ArrowLeftRight className="h-3.5 w-3.5" />
                {t(variant === "admin" ? "nav.goToUserPanel" : "nav.goToAdminPanel")}
              </button>
            )}
            {/* زنگ اعلان = تعداد درخواست‌های ناموفق همین session — یک سیگنال فنی برای
                ادمین/پشتیبانی، نه چیزی که برای مشتری معنا داشته باشد؛ فقط در پنل ادمین است. */}
            {variant === "admin" && (
              <button
                onClick={() => navigate("/admin/request-logs")}
                aria-label={t("nav.notifications")}
                className="relative rounded-full p-2 text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
              >
                <Bell className="h-4.5 w-4.5" />
                {failedCount > 0 && (
                  <span className="absolute -top-0.5 -end-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-fuchsia-500 px-1 text-[10px] font-bold text-white">
                    {failedCount > 9 ? "9+" : failedCount}
                  </span>
                )}
              </button>
            )}
            <LanguageSwitcher compact />
            <button
              onClick={toggle}
              aria-label={dark ? t("nav.toggleThemeLight") : t("nav.toggleThemeDark")}
              className="rounded-full p-2 text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
            >
              {dark ? <Sun className="h-4.5 w-4.5" /> : <Moon className="h-4.5 w-4.5" />}
            </button>
            <UserMenu me={me} onLogout={handleLogout} />
          </div>
        </header>

        <main className="pt-5">
          <div className="animate-fade-in">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

function UserMenu({ me, onLogout }: { me?: Me; onLogout: () => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const name = me?.user.first_name || me?.user.username || t("dashboard.user");

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div className="relative ms-1" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 rounded-full p-1 ps-1 pe-1.5 hover:bg-slate-100 dark:hover:bg-white/10"
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <span className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-gradient text-xs font-bold text-white">
          {initialsOf(name)}
        </span>
        <ChevronDown className="hidden h-3.5 w-3.5 text-slate-400 sm:block" />
      </button>
      {open && (
        <div className="glass-card absolute end-0 z-40 mt-1.5 w-48 animate-fade-in py-1">
          <div className="border-b border-slate-100 px-3 py-2 dark:border-white/10">
            <p className="truncate text-sm font-medium text-slate-700 dark:text-slate-200">{name}</p>
            <p className="text-xs text-slate-400">{me?.user.role ?? "—"}</p>
          </div>
          <button
            onClick={onLogout}
            className="flex w-full items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-fuchsia-300 dark:hover:bg-fuchsia-500/10"
          >
            <LogOut className="h-4 w-4" />
            {t("nav.logout")}
          </button>
        </div>
      )}
    </div>
  );
}
