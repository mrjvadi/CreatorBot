import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Receipt } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import { StatusBadge } from "@/components/ui/Badge";
import { formatDate } from "@/lib/format";
import type { Payment } from "@/lib/types";

/**
 * صفحه‌ی جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۳): قبلاً هیچ‌جا تاریخچه‌ی پرداخت‌های
 * خودِ کاربر نمایش داده نمی‌شد — با این‌که مدل Payment و endpoint ثبت آن از قبل
 * در سیستم وجود داشت. فقط نمایشی است؛ خریدِ پلن جدید همچنان از صفحه‌ی «پلن‌ها»
 * انجام می‌شود.
 */
export default function Payments() {
  const { t } = useTranslation();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["my-payments"],
    queryFn: async () => unwrap<Payment[]>(await api.get("/payments")),
  });

  const columns: DataTableColumn<Payment>[] = [
    {
      key: "created_at",
      header: t("payments.colDate"),
      sortValue: (row) => row.created_at ?? "",
      cell: (row) => <span className="text-sm">{formatDate(row.created_at)}</span>,
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
        <h2 className="text-xl font-bold">{t("payments.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("payments.subtitle")}</p>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("payments.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<Receipt className="h-8 w-8" />}
          emptyTitle={t("payments.emptyTitle")}
          searchPlaceholder={t("payments.searchPlaceholder")}
          searchFn={(row, q) => {
            const needle = q.toLowerCase();
            return (row.invoice_id ?? "").toLowerCase().includes(needle) || (row.tx_hash ?? "").toLowerCase().includes(needle);
          }}
        />
      )}
    </div>
  );
}
