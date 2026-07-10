import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Activity, ChevronDown, Pause, Play, Trash2 } from "lucide-react";
import clsx from "clsx";
import { useRequestLogStore, type RequestLogEntry } from "@/lib/request-log-store";
import { Button } from "@/components/ui/Button";
import { Select } from "@/components/ui/Input";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";

type StatusCategory = "all" | "success" | "clientError" | "serverError";

function statusClasses(entry: RequestLogEntry): string {
  if (entry.status === null) return "bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400";
  if (entry.status >= 500) return "bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400";
  if (entry.status >= 400) return "bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400";
  return "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400";
}

function categoryOf(entry: RequestLogEntry): StatusCategory {
  if (entry.status === null || entry.status >= 500) return "serverError";
  if (entry.status >= 400) return "clientError";
  return "success";
}

function formatTime(ts: number): string {
  return new Intl.DateTimeFormat("fa-IR", { timeStyle: "medium" }).format(new Date(ts));
}

function safeStringify(value: unknown): string {
  if (value === undefined) return "—";
  if (typeof value === "string") {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export default function RequestLogs() {
  const { t } = useTranslation();
  const entries = useRequestLogStore((s) => s.entries);
  const paused = useRequestLogStore((s) => s.paused);
  const togglePaused = useRequestLogStore((s) => s.togglePaused);
  const clear = useRequestLogStore((s) => s.clear);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [methodFilter, setMethodFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState<StatusCategory>("all");

  const methods = useMemo(() => Array.from(new Set(entries.map((e) => e.method))).sort(), [entries]);

  const filtered = useMemo(() => {
    return entries.filter((e) => {
      if (methodFilter !== "all" && e.method !== methodFilter) return false;
      if (statusFilter !== "all" && categoryOf(e) !== statusFilter) return false;
      return true;
    });
  }, [entries, methodFilter, statusFilter]);

  const columns: DataTableColumn<RequestLogEntry>[] = [
    {
      key: "time",
      header: t("requestLogs.colTime"),
      sortValue: (row) => row.startedAt,
      cell: (row) => (
        <span className="whitespace-nowrap text-xs text-slate-400" dir="ltr">
          {formatTime(row.startedAt)}
        </span>
      ),
    },
    {
      key: "method",
      header: t("requestLogs.colMethod"),
      sortValue: (row) => row.method,
      cell: (row) => (
        <span className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs font-medium dark:bg-slate-800">
          {row.method}
        </span>
      ),
    },
    {
      key: "path",
      header: t("requestLogs.colPath"),
      sortValue: (row) => row.url,
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.url}
        </span>
      ),
    },
    {
      key: "status",
      header: t("requestLogs.colStatus"),
      sortValue: (row) => row.status ?? -1,
      cell: (row) => (
        <span className={clsx("inline-flex rounded-full px-2 py-0.5 text-xs font-medium", statusClasses(row))}>
          {row.status ?? t("status.network")}
        </span>
      ),
    },
    {
      key: "duration",
      header: t("requestLogs.colDuration"),
      sortValue: (row) => row.durationMs,
      cell: (row) => (
        <span className="whitespace-nowrap text-xs tabular-nums text-slate-500 dark:text-slate-400">
          {row.durationMs}ms
        </span>
      ),
    },
    {
      key: "expand",
      header: "",
      cell: (row) => (
        <ChevronDown
          className={clsx(
            "h-4 w-4 text-slate-400 transition-transform",
            expandedId === row.id && "rotate-180"
          )}
        />
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold">{t("requestLogs.title")}</h2>
          <p className="mt-1 max-w-xl text-sm text-slate-500 dark:text-slate-400">{t("requestLogs.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-400">{t("common.totalItems", { count: entries.length })}</span>
          <Button size="sm" variant="secondary" onClick={togglePaused}>
            {paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
            {paused ? t("requestLogs.resume") : t("requestLogs.pause")}
          </Button>
          <Button size="sm" variant="secondary" onClick={clear} disabled={entries.length === 0}>
            <Trash2 className="h-3.5 w-3.5" />
            {t("common.clear")}
          </Button>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={filtered}
        getRowId={(row) => row.id}
        onRowClick={(row) => setExpandedId(expandedId === row.id ? null : row.id)}
        isRowExpanded={(row) => expandedId === row.id}
        emptyIcon={<Activity className="h-8 w-8" />}
        emptyTitle={t("requestLogs.emptyTitle")}
        emptyDescription={t("requestLogs.emptyDescription")}
        searchPlaceholder={t("requestLogs.searchPlaceholder")}
        searchFn={(row, q) => row.url.toLowerCase().includes(q.toLowerCase())}
        toolbarExtra={
          entries.length > 0 ? (
            <div className="flex gap-2">
              <Select value={methodFilter} onChange={(e) => setMethodFilter(e.target.value)} className="w-auto">
                <option value="all">{t("requestLogs.filterAllMethods")}</option>
                {methods.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </Select>
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusCategory)}
                className="w-auto"
              >
                <option value="all">{t("requestLogs.filterAllStatus")}</option>
                <option value="success">{t("requestLogs.filterSuccess")}</option>
                <option value="clientError">{t("requestLogs.filterClientError")}</option>
                <option value="serverError">{t("requestLogs.filterServerError")}</option>
              </Select>
            </div>
          ) : undefined
        }
        renderExpanded={(entry) => (
          <div className="p-4">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <p className="mb-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
                  {t("requestLogs.requestBody")}
                </p>
                <pre
                  className="max-h-56 overflow-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-100"
                  dir="ltr"
                >
                  {safeStringify(entry.requestBody)}
                </pre>
              </div>
              <div>
                <p className="mb-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
                  {t("requestLogs.responseBody")}
                </p>
                <pre
                  className="max-h-56 overflow-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-100"
                  dir="ltr"
                >
                  {safeStringify(entry.responseBody)}
                </pre>
              </div>
            </div>
            {entry.errorMessage && (
              <p className="mt-3 text-xs text-red-600 dark:text-red-400">
                {t("requestLogs.errorPrefix")}: {entry.errorMessage}
              </p>
            )}
          </div>
        )}
      />
    </div>
  );
}
