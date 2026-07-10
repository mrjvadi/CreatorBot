import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Receipt } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import { StatusBadge } from "@/components/ui/Badge";
import { Select } from "@/components/ui/Input";
import { formatDate } from "@/lib/format";
import type { Payment } from "@/lib/types";

// مقادیرِ واقعیِ models.PaymentStatus (بررسی‌شده ۲۰۲۶-۰۷-۰۵ روی خودِ Go source) — قبلاً
// "confirmed"/"expired" این‌جا اشتباهاً حدس زده شده بود.
const STATUS_OPTIONS = ["pending", "done", "failed"];

/**
 * صفحه‌ی جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۳): «پرداختی‌ها» تا الان هیچ نمایی در پنل
 * ادمین نداشت. فعلاً فقط نمایشی است (لیست + فیلتر وضعیت) — عمداً بدون امکان
 * تغییر دستیِ status، چون هیچ سرویسی که واقعاً روی این مدل بنویسد در apimanager
 * تأیید نشده؛ نوشتن دستی روی این جدول بدون آن تأیید ریسک ناهم‌خوانی با botpay دارد.
 */
export default function AdminPayments() {
  const { t } = useTranslation();
  const [statusFilter, setStatusFilter] = useState("all");

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-payments"],
    queryFn: async () => unwrap<Payment[]>(await api.get("/admin/payments")),
  });

  const filtered = useMemo(() => {
    if (!data) return [];
    if (statusFilter === "all") return data;
    return data.filter((p) => p.status?.toLowerCase() === statusFilter);
  }, [data, statusFilter]);

  const columns: DataTableColumn<Payment>[] = [
    {
      key: "created_at",
      header: t("payments.colDate"),
      sortValue: (row) => row.created_at ?? "",
      cell: (row) => <span className="text-sm">{formatDate(row.created_at)}</span>,
    },
    {
      key: "user_id",
      header: t("admin.payments.colUser"),
      cell: (row) => (
        <span className="font-mono text-xs text-slate-400" dir="ltr">
          {String(row.user_id).slice(0, 8)}…
        </span>
      ),
    },
    {
      key: "amount",
      header: t("payments.colAmount"),
      sortValue: (row) => row.amount,
      cell: (row) => (
        <span className="font-medium tabular-nums" dir="ltr">
          {row.amount} {row.currency}
        </span>
      ),
    },
    {
      key: "status",
      header: t("payments.colStatus"),
      sortValue: (row) => row.status ?? "",
      cell: (row) => <StatusBadge status={row.status} size="sm" />,
    },
    {
      key: "invoice_id",
      header: t("payments.colInvoice"),
      cell: (row) => (
        <span className="font-mono text-xs text-slate-400" dir="ltr">
          {row.invoice_id || row.tx_hash || "—"}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("admin.payments.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.payments.subtitle")}</p>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("payments.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={filtered}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<Receipt className="h-8 w-8" />}
          emptyTitle={t("admin.payments.emptyTitle")}
          searchPlaceholder={t("payments.searchPlaceholder")}
          searchFn={(row, q) => {
            const needle = q.toLowerCase();
            return (row.invoice_id ?? "").toLowerCase().includes(needle) || (row.tx_hash ?? "").toLowerCase().includes(needle);
          }}
          toolbarExtra={
            data && data.length > 0 ? (
              <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="w-auto">
                <option value="all">{t("instances.filterAllStatus")}</option>
                {STATUS_OPTIONS.map((s) => (
                  <option key={s} value={s}>
                    {t(`status.${s}`)}
                  </option>
                ))}
              </Select>
            ) : undefined
          }
        />
      )}
    </div>
  );
}
