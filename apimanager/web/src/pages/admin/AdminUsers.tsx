import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { Users, Bot, Ban, ShieldCheck, Wallet, Receipt } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState, EmptyState } from "@/components/ui/EmptyState";
import { Modal } from "@/components/ui/Modal";
import { StatusBadge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { Button } from "@/components/ui/Button";
import { Input, Select } from "@/components/ui/Input";
import { formatDate } from "@/lib/format";
import type { User, BotInstance, Subscription, Plan, Payment } from "@/lib/types";

interface UserDetail {
  user: User;
  instances: BotInstance[] | null;
  subscription: Subscription | null;
}

const ROLES = ["user", "admin", "owner"];

/**
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۳ («پنل کاربران رو درست کن»): چیزهایی که این‌جا کم بود اضافه شد —
 * موجودی کیف‌پول (User.balance از قبل روی مدل بود ولی هیچ‌جا نمایش داده نمی‌شد، درحالی‌که
 * دقیقاً همان چیزی است که دکمه‌ی «افزودن اعتبار» رویش اثر می‌گذارد)، تاریخ عضویت، فیلتر نقش،
 * و به‌جای «دارد/ندارد» برای اشتراک، نام واقعی پلن + تاریخ انقضا (همان الگویی که قبلاً روی
 * داشبورد خودِ کاربر پیاده شد). یک لیست کوچک از آخرین پرداخت‌های همین کاربر هم اضافه شد
 * (با فیلتر سمت کلاینت روی endpoint ای که برای صفحه‌ی «پرداختی‌ها»ی ادمین ساخته شده بود).
 */
export default function AdminUsers() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<User | null>(null);
  const [roleFilter, setRoleFilter] = useState("all");

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-users"],
    queryFn: async () => unwrap<User[]>(await api.get("/admin/users")),
  });

  const filtered = useMemo(() => {
    if (!data) return [];
    if (roleFilter === "all") return data;
    return data.filter((u) => u.role === roleFilter);
  }, [data, roleFilter]);

  const columns: DataTableColumn<User>[] = [
    {
      key: "name",
      header: t("admin.users.colName"),
      sortValue: (row) => row.first_name ?? row.username ?? "",
      cell: (row) => <span className="font-medium">{row.first_name || row.username || "—"}</span>,
    },
    {
      key: "username",
      header: t("admin.users.colUsername"),
      sortValue: (row) => row.username ?? "",
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.username ? `@${row.username}` : "—"}
        </span>
      ),
    },
    {
      key: "telegram_id",
      header: t("admin.users.colTelegramId"),
      sortValue: (row) => row.telegram_id ?? 0,
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.telegram_id ?? "—"}
        </span>
      ),
    },
    {
      key: "balance",
      header: t("admin.users.colBalance"),
      sortValue: (row) => row.balance ?? 0,
      cell: (row) => (
        <span className="tabular-nums" dir="ltr">
          {(row.balance ?? 0).toLocaleString()} TON
        </span>
      ),
    },
    {
      key: "role",
      header: t("admin.users.colRole"),
      sortValue: (row) => row.role,
      cell: (row) => (
        <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-white/5 dark:text-slate-300">
          {row.role}
        </span>
      ),
    },
    {
      key: "status",
      header: t("admin.users.isBlockedLabel"),
      sortValue: (row) => (row.is_blocked ? 1 : 0),
      cell: (row) =>
        row.is_blocked ? <StatusBadge status="error" size="sm" /> : <StatusBadge status="active" size="sm" />,
    },
    {
      key: "created_at",
      header: t("admin.users.colJoined"),
      sortValue: (row) => (row.created_at) ?? "",
      cell: (row) => <span className="text-xs text-slate-400">{formatDate(row.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("admin.users.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.users.subtitle")}</p>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.users.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={filtered}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          onRowClick={(row) => setSelected(row)}
          emptyIcon={<Users className="h-8 w-8" />}
          emptyTitle={t("admin.users.emptyTitle")}
          searchPlaceholder={t("admin.users.searchPlaceholder")}
          searchFn={(row, q) => {
            const needle = q.toLowerCase();
            return (
              (row.first_name ?? "").toLowerCase().includes(needle) ||
              (row.username ?? "").toLowerCase().includes(needle) ||
              String(row.telegram_id ?? "").includes(needle)
            );
          }}
          toolbarExtra={
            data && data.length > 0 ? (
              <Select value={roleFilter} onChange={(e) => setRoleFilter(e.target.value)} className="w-auto">
                <option value="all">{t("admin.users.filterAllRoles")}</option>
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </Select>
            ) : undefined
          }
        />
      )}

      <UserDetailModal user={selected} onClose={() => setSelected(null)} />
    </div>
  );
}

interface CreditForm {
  amount_ton: number;
  reason: string;
}

function UserDetailModal({ user, onClose }: { user: User | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [creditOpen, setCreditOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-user-detail", user?.id],
    queryFn: async () => unwrap<UserDetail>(await api.get(`/admin/users/${user!.id}`)),
    enabled: !!user,
  });

  const { data: plans } = useQuery({
    queryKey: ["admin-plans-lookup"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/admin/plans")),
    enabled: !!data?.subscription,
  });
  const planName = plans?.find((p) => p.id === data?.subscription?.plan_id)?.name;

  // آخرین پرداخت‌های همین کاربر — از همان endpoint صفحه‌ی «پرداختی‌ها»ی ادمین، فیلترشده سمت
  // کلاینت (هیچ endpoint اختصاصیِ «پرداخت‌های یک کاربر» وجود ندارد، و برای مقیاس فعلی همین کافی است).
  const { data: allPayments } = useQuery({
    queryKey: ["admin-payments"],
    queryFn: async () => unwrap<Payment[]>(await api.get("/admin/payments")),
    enabled: !!user,
  });
  const userPayments = (allPayments ?? []).filter((p) => p.user_id === user?.id).slice(0, 5);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["admin-user-detail", user?.id] });
    queryClient.invalidateQueries({ queryKey: ["admin-users"] });
  };

  const roleMutation = useMutation({
    mutationFn: (role: string) => api.post(`/admin/users/${user!.id}/role`, { role }),
    onSuccess: () => {
      toast.success(t("admin.users.roleChangeSuccess"));
      invalidate();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.users.roleChangeFailed"))),
  });

  const blockMutation = useMutation({
    mutationFn: (blocked: boolean) => api.post(`/admin/users/${user!.id}/${blocked ? "block" : "unblock"}`),
    onSuccess: (_res, blocked) => {
      toast.success(blocked ? t("admin.users.blockSuccess") : t("admin.users.unblockSuccess"));
      invalidate();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.users.blockFailed"))),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreditForm>();

  const creditMutation = useMutation({
    mutationFn: (values: CreditForm) => api.post(`/admin/users/${user!.id}/credit`, values),
    onSuccess: () => {
      toast.success(t("admin.users.creditSuccess"));
      invalidate();
      queryClient.invalidateQueries({ queryKey: ["admin-payments"] });
      setCreditOpen(false);
      reset();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.users.creditFailed"))),
  });

  if (!user) return null;
  const name = user.first_name || user.username || "—";

  return (
    <Modal open={!!user} onClose={onClose} title={name}>
      {isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-5 w-2/3" />
          <Skeleton className="h-16" />
        </div>
      )}
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{apiErrorMessage(error, t("admin.users.loadFailed"))}</p>
      )}
      {!isLoading && !error && data && (
        <div className="space-y-4">
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-slate-400">{t("admin.users.colUsername")}</dt>
            <dd dir="ltr" className="text-end font-mono text-xs">
              {data.user.username ? `@${data.user.username}` : "—"}
            </dd>
            <dt className="text-slate-400">{t("admin.users.colTelegramId")}</dt>
            <dd dir="ltr" className="text-end font-mono text-xs">
              {data.user.telegram_id ?? "—"}
            </dd>
            <dt className="text-slate-400">{t("admin.users.colBalance")}</dt>
            <dd dir="ltr" className="text-end font-medium tabular-nums">
              {(data.user.balance ?? 0).toLocaleString()} TON
            </dd>
            <dt className="text-slate-400">{t("admin.users.colJoined")}</dt>
            <dd className="text-end font-medium">{formatDate(data.user.created_at)}</dd>
            <dt className="text-slate-400">{t("admin.users.isBlockedLabel")}</dt>
            <dd className="text-end font-medium">
              {data.user.is_blocked ? t("admin.users.blocked") : t("admin.users.notBlocked")}
            </dd>
            <dt className="text-slate-400">{t("dashboard.activeSubscription")}</dt>
            <dd className="text-end font-medium">
              {data.subscription ? planName ?? t("dashboard.hasSubscription") : t("dashboard.noSubscription")}
            </dd>
            {data.subscription && (
              <>
                <dt className="text-slate-400">{t("dashboard.expiresAt")}</dt>
                <dd className="text-end font-medium">
                  {data.subscription.expires_at ? formatDate(data.subscription.expires_at) : t("dashboard.neverExpires")}
                </dd>
              </>
            )}
          </dl>

          {/* عملیات ادمین */}
          <div className="space-y-2.5 rounded-xl border border-slate-100 p-3 dark:border-white/10">
            <div className="flex items-center gap-2">
              <label className="w-28 shrink-0 text-xs text-slate-500 dark:text-slate-400">
                {t("admin.users.roleChangeLabel")}
              </label>
              <Select
                value={data.user.role}
                disabled={roleMutation.isPending}
                onChange={(e) => roleMutation.mutate(e.target.value)}
                className="flex-1"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </Select>
            </div>

            <div className="flex flex-wrap gap-2">
              {data.user.is_blocked ? (
                <Button
                  size="sm"
                  variant="secondary"
                  loading={blockMutation.isPending}
                  onClick={() => {
                    if (confirm(t("admin.users.unblockConfirm"))) blockMutation.mutate(false);
                  }}
                >
                  <ShieldCheck className="h-3.5 w-3.5" />
                  {t("admin.users.unblockAction")}
                </Button>
              ) : (
                <Button
                  size="sm"
                  variant="danger"
                  loading={blockMutation.isPending}
                  onClick={() => {
                    if (confirm(t("admin.users.blockConfirm"))) blockMutation.mutate(true);
                  }}
                >
                  <Ban className="h-3.5 w-3.5" />
                  {t("admin.users.blockAction")}
                </Button>
              )}
              <Button size="sm" variant="secondary" onClick={() => setCreditOpen(true)}>
                <Wallet className="h-3.5 w-3.5" />
                {t("admin.users.creditButton")}
              </Button>
            </div>
          </div>

          <div>
            <p className="mb-2 text-xs font-medium text-slate-500 dark:text-slate-400">
              {t("dashboard.botCount")} ({data.instances?.length ?? 0})
            </p>
            {!data.instances || data.instances.length === 0 ? (
              <EmptyState icon={<Bot className="h-6 w-6" />} title={t("instances.emptyTitle")} />
            ) : (
              <ul className="space-y-1.5">
                {data.instances.map((inst) => (
                  <li
                    key={inst.id}
                    className="flex items-center justify-between rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-white/10"
                  >
                    <span className="font-mono text-xs">{inst.container_name ?? inst.id}</span>
                    <StatusBadge status={inst.status} size="sm" />
                  </li>
                ))}
              </ul>
            )}
          </div>

          {userPayments.length > 0 && (
            <div>
              <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
                <Receipt className="h-3.5 w-3.5" />
                {t("admin.users.recentPaymentsLabel")}
              </p>
              <ul className="space-y-1.5">
                {userPayments.map((p) => (
                  <li
                    key={p.id}
                    className="flex items-center justify-between rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-white/10"
                  >
                    <span className="tabular-nums" dir="ltr">
                      {p.amount} {p.currency}
                    </span>
                    <StatusBadge status={p.status} size="sm" />
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <Modal open={creditOpen} onClose={() => setCreditOpen(false)} title={t("admin.users.creditTitle")}>
        <form className="space-y-4" onSubmit={handleSubmit((values) => creditMutation.mutate(values))}>
          <p className="text-xs text-slate-400">{t("admin.users.creditHint")}</p>
          <Input
            label={t("admin.users.creditAmountLabel")}
            type="number"
            step="any"
            dir="ltr"
            {...register("amount_ton", {
              required: t("admin.users.creditAmountRequired"),
              valueAsNumber: true,
              min: { value: 0.0001, message: t("admin.users.creditAmountRequired") },
            })}
            error={errors.amount_ton?.message}
          />
          <Input
            label={t("admin.users.creditReasonLabel")}
            {...register("reason", { required: t("admin.users.creditReasonRequired") })}
            error={errors.reason?.message}
          />
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setCreditOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={isSubmitting || creditMutation.isPending}>
              {t("admin.users.creditButton")}
            </Button>
          </div>
        </form>
      </Modal>
    </Modal>
  );
}
