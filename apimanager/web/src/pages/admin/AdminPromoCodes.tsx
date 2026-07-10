import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { Ticket, Plus, Trash2, Power } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import { formatDate } from "@/lib/format";
import type { PromoCode } from "@/lib/types";

interface CreateForm {
  code: string;
  amount_ton: number;
  max_uses: number;
}

/**
 * صفحه‌ی جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۳): کدهای تخفیف/شارژ (PromoCode) از قبل
 * در دیتابیس و store پیاده‌سازی شده بودند (redeem اتمیک با row-lock) اما هیچ
 * راهی برای ادمین برای ساخت/دیدنشان از طریق وب وجود نداشت. اینجا CRUD پایه
 * (ساخت/فعال-غیرفعال/حذف) اضافه شده — ویرایش مبلغ/سقف عمداً غیرفعال است چون
 * تغییر AmountTON روی کدی که از قبل redeem شده می‌تواند گمراه‌کننده باشد؛ برای
 * تغییر واقعی باید کد قدیمی غیرفعال و کد جدید ساخته شود.
 */
export default function AdminPromoCodes() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-promo-codes"],
    queryFn: async () => unwrap<PromoCode[]>(await api.get("/admin/promo-codes")),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateForm>({ defaultValues: { max_uses: 0 } });

  const createMutation = useMutation({
    mutationFn: (values: CreateForm) => api.post("/admin/promo-codes", values),
    onSuccess: () => {
      toast.success(t("admin.promoCodes.addSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-promo-codes"] });
      setCreateOpen(false);
      reset();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.promoCodes.addFailed"))),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      api.patch(`/admin/promo-codes/${id}`, { is_active }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-promo-codes"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.promoCodes.toggleFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/promo-codes/${id}`),
    onSuccess: () => {
      toast.success(t("admin.promoCodes.deleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-promo-codes"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.promoCodes.deleteFailed"))),
  });

  const columns: DataTableColumn<PromoCode>[] = [
    {
      key: "code",
      header: t("admin.promoCodes.colCode"),
      sortValue: (row) => row.code,
      cell: (row) => (
        <span className="font-mono text-sm font-semibold" dir="ltr">
          {row.code}
        </span>
      ),
    },
    {
      key: "amount_ton",
      header: t("admin.promoCodes.colAmount"),
      sortValue: (row) => row.amount_ton,
      cell: (row) => (
        <span className="tabular-nums" dir="ltr">
          {row.amount_ton} TON
        </span>
      ),
    },
    {
      key: "usage",
      header: t("admin.promoCodes.colUsage"),
      sortValue: (row) => row.used_count,
      cell: (row) => (
        <span className="tabular-nums" dir="ltr">
          {row.used_count} / {row.max_uses > 0 ? row.max_uses : "∞"}
        </span>
      ),
    },
    {
      key: "expires_at",
      header: t("admin.promoCodes.colExpires"),
      cell: (row) => (row.expires_at ? formatDate(row.expires_at) : t("admin.promoCodes.noExpiry")),
    },
    {
      key: "status",
      header: t("admin.plans.colStatus"),
      sortValue: (row) => (row.is_active ? 1 : 0),
      cell: (row) => (
        <span
          className={
            row.is_active
              ? "inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-300"
              : "inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500 dark:bg-white/5 dark:text-slate-400"
          }
        >
          {row.is_active ? t("common.yes") : t("common.no")}
        </span>
      ),
    },
    {
      key: "actions",
      header: t("admin.plans.colActions"),
      cell: (row) => (
        <div className="flex items-center gap-1">
          <button
            onClick={() => toggleMutation.mutate({ id: row.id, is_active: !row.is_active })}
            aria-label={t("admin.promoCodes.toggleButton")}
            title={t("admin.promoCodes.toggleButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Power className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              if (confirm(t("admin.promoCodes.deleteConfirm", { code: row.code }))) deleteMutation.mutate(row.id);
            }}
            aria-label={t("common.delete")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold">{t("admin.promoCodes.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.promoCodes.subtitle")}</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("admin.promoCodes.addButton")}
        </Button>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.promoCodes.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<Ticket className="h-8 w-8" />}
          emptyTitle={t("admin.promoCodes.emptyTitle")}
          searchPlaceholder={t("admin.promoCodes.searchPlaceholder")}
          searchFn={(row, q) => row.code.toLowerCase().includes(q.toLowerCase())}
        />
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title={t("admin.promoCodes.addTitle")}>
        <form className="space-y-4" onSubmit={handleSubmit((values) => createMutation.mutate(values))}>
          <Input
            label={t("admin.promoCodes.codeLabel")}
            dir="ltr"
            placeholder="WELCOME10"
            {...register("code", { required: t("admin.promoCodes.codeRequired") })}
            error={errors.code?.message}
          />
          <Input
            label={t("admin.promoCodes.amountLabel")}
            type="number"
            step="any"
            dir="ltr"
            {...register("amount_ton", { required: true, valueAsNumber: true, min: 0.01 })}
            error={errors.amount_ton?.message}
          />
          <div>
            <Input
              label={t("admin.promoCodes.maxUsesLabel")}
              type="number"
              dir="ltr"
              {...register("max_uses", { valueAsNumber: true })}
            />
            <p className="mt-1 text-xs text-slate-400">{t("admin.promoCodes.maxUsesHint")}</p>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setCreateOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={isSubmitting || createMutation.isPending}>
              {t("common.add")}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
