import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Bot, ServerCog, Users, PlayCircle, PauseCircle, Clock, XCircle } from "lucide-react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { StatCard, Card } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { ErrorState } from "@/components/ui/EmptyState";
import { CHART_COLORS } from "@/lib/chart-colors";
import type { AdminStats as AdminStatsType } from "@/lib/types";

function useDelta(value: number | undefined, key: string) {
  const prev = useRef<Record<string, number>>({});
  const [delta, setDelta] = useState(0);
  useEffect(() => {
    if (value === undefined) return;
    if (prev.current[key] !== undefined) {
      setDelta(value - prev.current[key]);
    }
    prev.current[key] = value;
  }, [value, key]);
  return delta;
}

export default function AdminStats() {
  const { t } = useTranslation();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: async () => unwrap<AdminStatsType>(await api.get("/admin/stats")),
    refetchInterval: 30_000,
  });

  const totalDelta = useDelta(data?.instances.total, "total");
  const usersDelta = useDelta(data?.users, "users");
  const onlineDelta = useDelta(data?.servers.online, "online");

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    );
  }

  if (error) {
    return <ErrorState message={apiErrorMessage(error, t("admin.stats.loadFailed"))} onRetry={refetch} />;
  }

  const instances = data?.instances;
  const total = instances?.total || 0;
  const breakdown = [
    { key: "running", value: instances?.running ?? 0, color: CHART_COLORS.running, icon: PlayCircle },
    { key: "stopped", value: instances?.stopped ?? 0, color: CHART_COLORS.stopped, icon: PauseCircle },
    { key: "pending", value: instances?.pending ?? 0, color: CHART_COLORS.pending, icon: Clock },
    { key: "error", value: instances?.error ?? 0, color: CHART_COLORS.error, icon: XCircle },
  ];
  const pieData = breakdown.filter((b) => b.value > 0);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("admin.stats.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.stats.subtitle")}</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          label={t("admin.stats.totalInstances")}
          value={instances?.total ?? 0}
          icon={<Bot className="h-5 w-5" />}
          delta={totalDelta}
        />
        <StatCard
          label={t("admin.stats.servers")}
          value={`${data?.servers.online ?? 0} / ${data?.servers.total ?? 0}`}
          hint={t("admin.stats.onlineOfTotal")}
          icon={<ServerCog className="h-5 w-5" />}
          delta={onlineDelta}
          accent="success"
        />
        <StatCard
          label={t("admin.stats.users")}
          value={data?.users ?? 0}
          icon={<Users className="h-5 w-5" />}
          delta={usersDelta}
        />
      </div>

      <Card>
        <h3 className="mb-4 font-semibold">{t("admin.stats.statusBreakdown")}</h3>
        <div className="grid grid-cols-1 items-center gap-6 md:grid-cols-2">
          <div className="relative mx-auto h-56 w-56">
            {total > 0 ? (
              <>
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={pieData}
                      dataKey="value"
                      nameKey="key"
                      innerRadius={62}
                      outerRadius={90}
                      paddingAngle={pieData.length > 1 ? 3 : 0}
                      strokeWidth={0}
                    >
                      {pieData.map((entry) => (
                        <Cell key={entry.key} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      formatter={(value, name) => [value as number, t(`status.${name}`)] as [number, string]}
                      contentStyle={{
                        borderRadius: 12,
                        border: "1px solid rgba(255,255,255,0.1)",
                        background: "rgba(23,18,38,0.9)",
                        color: "#fff",
                        fontSize: 12,
                        direction: "ltr",
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
                {/* برچسبِ مرکزیِ نمودار دونات — عدد واقعیِ کل instance ها (نه فرضی) */}
                <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-3xl font-bold tabular-nums">{total}</span>
                  <span className="text-xs text-slate-400">{t("admin.stats.totalInstances")}</span>
                </div>
              </>
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-slate-400">0</div>
            )}
          </div>
          <div className="space-y-3">
            {breakdown.map(({ key, value, color, icon: Icon }) => (
              <div key={key} className="flex items-center gap-3">
                <Icon className="h-4 w-4 shrink-0" style={{ color }} />
                <span className="w-32 shrink-0 text-sm text-slate-600 dark:text-slate-300">
                  {t(`status.${key}`)}
                </span>
                <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-100 dark:bg-white/10">
                  <div
                    className="h-full rounded-full transition-all"
                    style={{
                      width: `${total > 0 ? (value / total) * 100 : 0}%`,
                      backgroundColor: color,
                    }}
                  />
                </div>
                <span className="w-8 shrink-0 text-end text-sm font-medium tabular-nums">{value}</span>
              </div>
            ))}
          </div>
        </div>
      </Card>
    </div>
  );
}
