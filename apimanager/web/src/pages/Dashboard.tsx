import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Bot, CreditCard, ArrowLeft, ArrowRight, ScrollText } from "lucide-react";
import { Link } from "react-router-dom";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { StatCard, Card } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { ErrorState } from "@/components/ui/EmptyState";
import { Button } from "@/components/ui/Button";
import { formatDate } from "@/lib/format";
import type { Me, Plan, Subscription } from "@/lib/types";

/**
 * بازطراحی ۲۰۲۶-۰۷-۰۳: قبلاً «نقش حساب» به‌عنوان KPI برای یک کاربر عادی نشان داده می‌شد
 * (تقریباً همیشه همان «user» است — اطلاعاتی که به کاربر واقعی هیچ کمکی نمی‌کند) و یک کارت
 * جداگانه همان چیزها (telegram_id/username/role/bot_count) را دوباره تکرار می‌کرد. حذف شد؛
 * به‌جایش جزئیات واقعیِ اشتراک (نام پلن + تاریخ انقضا، نه فقط «دارد/ندارد») نشان داده می‌شود.
 */
export default function Dashboard() {
  const { t, i18n } = useTranslation();
  const { data, isLoading, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: ["me"],
    queryFn: async () => unwrap<Me>(await api.get("/me")),
    refetchInterval: 20_000,
  });

  const subscription = data?.subscription as Subscription | null | undefined;

  const { data: plans } = useQuery({
    queryKey: ["plans"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/plans")),
    enabled: !!subscription,
  });
  const planName = plans?.find((p) => p.id === subscription?.plan_id)?.name;

  const prevBotCount = useRef<number | null>(null);
  const [botDelta, setBotDelta] = useState(0);
  useEffect(() => {
    if (data == null) return;
    if (prevBotCount.current !== null) {
      setBotDelta(data.bot_count - prevBotCount.current);
    }
    prevBotCount.current = data.bot_count;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dataUpdatedAt]);

  const ArrowIcon = i18n.dir() === "rtl" ? ArrowLeft : ArrowRight;

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
    );
  }

  if (error) {
    return <ErrorState message={apiErrorMessage(error, t("dashboard.loadFailed"))} onRetry={refetch} />;
  }

  const displayName = data?.user.first_name || data?.user.username || t("dashboard.user");

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("dashboard.greeting", { name: displayName })}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("dashboard.subtitle")}</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <StatCard
          label={t("dashboard.botCount")}
          value={data?.bot_count ?? 0}
          icon={<Bot className="h-5 w-5" />}
          delta={botDelta}
          hint={botDelta !== 0 ? t("dashboard.sinceLastCheck") : undefined}
        />
        <StatCard
          label={t("dashboard.activeSubscription")}
          value={
            subscription
              ? planName ?? t("dashboard.hasSubscription")
              : t("dashboard.noSubscription")
          }
          hint={
            subscription?.expires_at
              ? `${t("dashboard.expiresAt")}: ${formatDate(subscription.expires_at)}`
              : subscription
                ? t("dashboard.neverExpires")
                : undefined
          }
          icon={<CreditCard className="h-5 w-5" />}
          accent={subscription ? "success" : "warning"}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Card className="flex flex-col justify-between gap-4">
          <div className="flex items-start gap-3">
            <span className="rounded-xl bg-violet-50 p-2.5 text-violet-600 dark:bg-violet-500/15 dark:text-violet-300">
              <Bot className="h-5 w-5" />
            </span>
            <div>
              <p className="font-medium">{t("dashboard.ctaTitle")}</p>
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("dashboard.ctaSubtitle")}</p>
            </div>
          </div>
          <Link to="/app/instances">
            <Button className="w-full justify-center">
              {t("dashboard.ctaButton")}
              <ArrowIcon className="h-4 w-4" />
            </Button>
          </Link>
        </Card>

        <Card className="flex flex-col justify-between gap-4">
          <div className="flex items-start gap-3">
            <span className="rounded-xl bg-violet-50 p-2.5 text-violet-600 dark:bg-violet-500/15 dark:text-violet-300">
              <ScrollText className="h-5 w-5" />
            </span>
            <div>
              <p className="font-medium">{t("dashboard.plansCtaTitle")}</p>
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("dashboard.plansCtaSubtitle")}</p>
            </div>
          </div>
          <Link to="/app/plans">
            <Button variant="secondary" className="w-full justify-center">
              {t("dashboard.plansCtaButton")}
              <ArrowIcon className="h-4 w-4" />
            </Button>
          </Link>
        </Card>
      </div>
    </div>
  );
}
