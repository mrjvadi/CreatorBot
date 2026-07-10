import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { CreditCard, Plus, Pencil, Trash2, SlidersHorizontal, X } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import type { Plan } from "@/lib/types";

interface PlanForm {
  name: string;
  price: number;
  duration_day: number;
  max_bots: number;
  is_free: boolean;
  is_active: boolean;
}

const BOT_TYPES = ["uploader", "vpn", "archive", "member"];

export default function AdminPlans() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Plan | null>(null);
  const [limitsFor, setLimitsFor] = useState<Plan | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-plans"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/admin/plans")),
  });

  const createForm = useForm<PlanForm>({
    defaultValues: { price: 0, duration_day: 30, max_bots: 1, is_free: false, is_active: true },
  });

  const createMutation = useMutation({
    mutationFn: (values: PlanForm) => api.post("/admin/plans", values),
    onSuccess: () => {
      toast.success(t("admin.plans.addSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-plans"] });
      setCreateOpen(false);
      createForm.reset();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.plans.addFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/plans/${id}`),
    onSuccess: () => {
      toast.success(t("admin.plans.deleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-plans"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.plans.deleteFailed"))),
  });

  const columns: DataTableColumn<Plan>[] = [
    {
      key: "name",
      header: t("admin.plans.colName"),
      sortValue: (row) => row.name,
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      key: "price",
      header: t("admin.plans.colPrice"),
      sortValue: (row) => row.price,
      cell: (row) => <span className="tabular-nums">{row.is_free ? t("admin.plans.isFreeLabel") : row.price}</span>,
    },
    {
      key: "duration",
      header: t("admin.plans.colDuration"),
      sortValue: (row) => row.duration_day,
      cell: (row) => (row.duration_day > 0 ? row.duration_day : t("admin.plans.unlimited")),
    },
    {
      key: "max_bots",
      header: t("admin.plans.colMaxBots"),
      sortValue: (row) => row.max_bots,
      cell: (row) => <span className="tabular-nums">{row.max_bots}</span>,
    },
    {
      key: "status",
      header: t("admin.plans.colStatus"),
      sortValue: (row) => (row.is_active ? 1 : 0),
      cell: (row) => (
        <span
          className={
            row.is_active
              ? "inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
              : "inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500 dark:bg-slate-800"
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
            onClick={() => setLimitsFor(row)}
            aria-label={t("admin.plans.limitsButton")}
            title={t("admin.plans.limitsButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
          >
            <SlidersHorizontal className="h-4 w-4" />
          </button>
          <button
            onClick={() => setEditing(row)}
            aria-label={t("admin.plans.editButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
          >
            <Pencil className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              if (confirm(t("admin.plans.deleteConfirm", { name: row.name }))) deleteMutation.mutate(row.id);
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
          <h2 className="text-xl font-bold">{t("admin.plans.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.plans.subtitle")}</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("admin.plans.addButton")}
        </Button>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.plans.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<CreditCard className="h-8 w-8" />}
          emptyTitle={t("admin.plans.emptyTitle")}
          searchPlaceholder={t("admin.plans.searchPlaceholder")}
          searchFn={(row, q) => row.name.toLowerCase().includes(q.toLowerCase())}
        />
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title={t("admin.plans.addTitle")}>
        <form className="space-y-4" onSubmit={createForm.handleSubmit((values) => createMutation.mutate(values))}>
          <Input
            label={t("admin.plans.nameLabel")}
            {...createForm.register("name", { required: t("admin.plans.nameRequired") })}
            error={createForm.formState.errors.name?.message}
          />
          <Input
            label={t("admin.plans.priceLabel")}
            type="number"
            step="any"
            dir="ltr"
            {...createForm.register("price", { valueAsNumber: true })}
          />
          <Input
            label={t("admin.plans.durationLabel")}
            type="number"
            dir="ltr"
            {...createForm.register("duration_day", { valueAsNumber: true })}
          />
          <Input
            label={t("admin.plans.maxBotsLabel")}
            type="number"
            dir="ltr"
            {...createForm.register("max_bots", { valueAsNumber: true })}
          />
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" {...createForm.register("is_free")} className="h-4 w-4 rounded border-slate-300" />
            {t("admin.plans.isFreeLabel")}
          </label>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setCreateOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={createForm.formState.isSubmitting || createMutation.isPending}>
              {t("common.add")}
            </Button>
          </div>
        </form>
      </Modal>

      <EditPlanModal plan={editing} onClose={() => setEditing(null)} />
      <PlanLimitsModal plan={limitsFor} onClose={() => setLimitsFor(null)} />
    </div>
  );
}

function EditPlanModal({ plan, onClose }: { plan: Plan | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { register, handleSubmit, formState } = useForm<PlanForm>({
    values: plan
      ? {
          name: plan.name,
          price: plan.price,
          duration_day: plan.duration_day,
          max_bots: plan.max_bots,
          is_free: plan.is_free,
          is_active: plan.is_active,
        }
      : undefined,
  });

  const mutation = useMutation({
    mutationFn: (values: PlanForm) => api.patch(`/admin/plans/${plan!.id}`, values),
    onSuccess: () => {
      toast.success(t("admin.plans.editSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-plans"] });
      onClose();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.plans.editFailed"))),
  });

  if (!plan) return null;

  return (
    <Modal open={!!plan} onClose={onClose} title={t("admin.plans.editTitle")}>
      <form className="space-y-4" onSubmit={handleSubmit((values) => mutation.mutate(values))}>
        <Input label={t("admin.plans.nameLabel")} {...register("name", { required: true })} />
        <Input label={t("admin.plans.priceLabel")} type="number" step="any" dir="ltr" {...register("price", { valueAsNumber: true })} />
        <Input label={t("admin.plans.durationLabel")} type="number" dir="ltr" {...register("duration_day", { valueAsNumber: true })} />
        <Input label={t("admin.plans.maxBotsLabel")} type="number" dir="ltr" {...register("max_bots", { valueAsNumber: true })} />
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...register("is_free")} className="h-4 w-4 rounded border-slate-300" />
          {t("admin.plans.isFreeLabel")}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...register("is_active")} className="h-4 w-4 rounded border-slate-300" />
          {t("admin.plans.isActiveLabel")}
        </label>
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" loading={formState.isSubmitting || mutation.isPending}>
            {t("common.save")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function PlanLimitsModal({ plan, onClose }: { plan: Plan | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [botType, setBotType] = useState(BOT_TYPES[0]);
  const [maxBots, setMaxBots] = useState(1);

  const mutation = useMutation({
    mutationFn: (payload: { bot_type: string; max_bots: number }) =>
      api.patch(`/admin/plans/${plan!.id}/limits`, payload),
    onSuccess: () => {
      toast.success(t("admin.plans.limitsSaveSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-plans"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.plans.limitsSaveFailed"))),
  });

  if (!plan) return null;

  return (
    <Modal open={!!plan} onClose={onClose} title={t("admin.plans.limitsTitle", { name: plan.name })}>
      <div className="space-y-4">
        <p className="text-xs text-slate-400">{t("admin.plans.limitsHint")}</p>

        {plan.limits && plan.limits.length > 0 && (
          <ul className="space-y-1.5">
            {plan.limits.map((l) => (
              <li
                key={l.bot_type}
                className="flex items-center justify-between rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-slate-800"
              >
                <span className="font-mono text-xs">{l.bot_type}</span>
                <span className="tabular-nums">{l.max_bots}</span>
              </li>
            ))}
          </ul>
        )}

        <div className="flex items-end gap-2 border-t border-dashed border-slate-200 pt-3 dark:border-slate-800">
          <div className="flex-1">
            <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
              {t("admin.plans.limitsBotTypeLabel")}
            </label>
            <select
              value={botType}
              onChange={(e) => setBotType(e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              dir="ltr"
            >
              {BOT_TYPES.map((bt) => (
                <option key={bt} value={bt}>
                  {bt}
                </option>
              ))}
            </select>
          </div>
          <div className="w-28">
            <Input
              label={t("admin.plans.limitsMaxBotsLabel")}
              type="number"
              dir="ltr"
              value={maxBots}
              onChange={(e) => setMaxBots(Number(e.target.value))}
            />
          </div>
          <Button
            type="button"
            loading={mutation.isPending}
            onClick={() => mutation.mutate({ bot_type: botType, max_bots: maxBots })}
          >
            {t("common.save")}
          </Button>
        </div>

        <div className="flex justify-end pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            <X className="h-3.5 w-3.5" />
            {t("common.close")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
