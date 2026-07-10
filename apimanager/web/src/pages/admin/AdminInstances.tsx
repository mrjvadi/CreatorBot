import { useMemo, useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { Bot, ScrollText, Settings, ArrowRightLeft, Pencil } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import { StatusBadge } from "@/components/ui/Badge";
import { Select, Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Button } from "@/components/ui/Button";
import { ActionMenu } from "@/components/ui/ActionMenu";
import { Skeleton } from "@/components/ui/Skeleton";
import type { BotInstance, TemplateConfigField, Server, Plan } from "@/lib/types";

const LOCK_MODES = ["none", "free", "rented"];

const STATUS_OPTIONS = ["running", "pending", "stopped", "error"];

/**
 * صفحه‌ی جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۳): قبلاً ادمین هیچ راه مستقیمی برای دیدن همه‌ی
 * instance های پلتفرم نداشت — فقط غیرمستقیم از طریق «سرورها» (هر سرور جدا) یا «کاربران»
 * (هر کاربر جدا). این‌جا نمای یکپارچه‌ی همه چیز است، با جست‌وجو/فیلتر وضعیت.
 *
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «فقط لاگ نشون میده، هیچ چیز دیگه‌ای نیست — نمی‌تونم بیام
 * تنظیماتش رو نگاه کنم اگه یکی مشکل داشت». اضافه شد: عملیات «تنظیمات» که همان
 * GET/PUT /instances/:id/settings کاربر است، فقط این‌بار سمت بک‌اند چک مالکیت برای
 * نقش admin/owner کنار گذاشته می‌شود — یعنی ادمین می‌تواند تنظیمات هر رباتی را برای
 * پشتیبانی ببیند/ویرایش کند، بدون نیاز به لاگین با اکانت خودِ کاربر.
 * start/stop/delete همچنان عمداً این‌جا نیست — آن عملیات‌ها مسئولیت مالکِ instance می‌ماند.
 */
export default function AdminInstances() {
  const { t } = useTranslation();
  const [statusFilter, setStatusFilter] = useState("all");
  const [logsFor, setLogsFor] = useState<BotInstance | null>(null);
  const [settingsFor, setSettingsFor] = useState<BotInstance | null>(null);
  const [migrateFor, setMigrateFor] = useState<BotInstance | null>(null);
  const [editingFor, setEditingFor] = useState<BotInstance | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-instances"],
    queryFn: async () => unwrap<BotInstance[]>(await api.get("/admin/instances")),
  });

  const { data: servers } = useQuery({
    queryKey: ["admin-servers"],
    queryFn: async () => unwrap<Server[]>(await api.get("/admin/servers")),
  });
  const serverName = (id?: string) => servers?.find((s) => s.id === id)?.name ?? id?.slice(0, 8) + "…";

  const filtered = useMemo(() => {
    if (!data) return [];
    if (statusFilter === "all") return data;
    return data.filter((i) => i.status?.toLowerCase() === statusFilter);
  }, [data, statusFilter]);

  const columns: DataTableColumn<BotInstance>[] = [
    {
      key: "bot_id",
      header: t("instances.colBotId"),
      sortValue: (row) => row.bot_id ?? 0,
      cell: (row) => <span className="font-mono text-xs">{row.bot_id ?? row.id}</span>,
    },
    {
      key: "container_name",
      header: t("instances.colContainer"),
      sortValue: (row) => row.container_name ?? "",
      cell: (row) => row.container_name ?? "—",
    },
    {
      key: "owner",
      header: t("admin.instances.colOwner"),
      cell: (row) => (
        <span className="font-mono text-xs text-slate-400" dir="ltr">
          {row.owner_id ? String(row.owner_id).slice(0, 8) + "…" : "—"}
        </span>
      ),
    },
    {
      key: "server",
      header: t("admin.instances.colServer"),
      cell: (row) => <span className="text-xs text-slate-400">{serverName(row.server_id)}</span>,
    },
    {
      key: "status",
      header: t("instances.colStatus"),
      sortValue: (row) => row.status ?? "",
      cell: (row) => <StatusBadge status={row.status} size="sm" />,
    },
    {
      key: "actions",
      header: t("instances.colActions"),
      cell: (row) => (
        <ActionMenu
          ariaLabel={t("common.actions")}
          items={[
            { label: t("instances.actionSettings"), icon: Settings, onClick: () => setSettingsFor(row) },
            { label: t("instances.actionLogs"), icon: ScrollText, onClick: () => setLogsFor(row) },
            { label: t("admin.instances.actionMigrate"), icon: ArrowRightLeft, onClick: () => setMigrateFor(row) },
            { label: t("admin.instances.actionEdit"), icon: Pencil, onClick: () => setEditingFor(row) },
          ]}
        />
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("admin.instances.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.instances.subtitle")}</p>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.instances.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={filtered}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<Bot className="h-8 w-8" />}
          emptyTitle={t("admin.instances.emptyTitle")}
          searchPlaceholder={t("instances.searchPlaceholder")}
          searchFn={(row, q) =>
            (row.container_name ?? "").toLowerCase().includes(q.toLowerCase()) ||
            String(row.bot_id ?? row.id).toLowerCase().includes(q.toLowerCase())
          }
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

      <LogsModal instance={logsFor} onClose={() => setLogsFor(null)} />
      <SettingsModal instance={settingsFor} onClose={() => setSettingsFor(null)} />
      <MigrateModal instance={migrateFor} servers={servers ?? []} onClose={() => setMigrateFor(null)} />
      <EditInstanceModal instance={editingFor} onClose={() => setEditingFor(null)} />
    </div>
  );
}

/**
 * ویرایشِ دستیِ instance (بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «باید بتونم ربات‌ها رو از پنل ادمین ادیت
 * کنم»). فقط فیلدهایی که ویرایشِ دستی برایشان امن است: انقضا، پلن، lock mode. نام کانتینر/
 * سرور/توکن عمداً این‌جا نیستند — سرور از طریق «انتقال» عوض می‌شود، نه این‌جا.
 */
function EditInstanceModal({ instance, onClose }: { instance: BotInstance | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [expiresAt, setExpiresAt] = useState("");
  const [planId, setPlanId] = useState("");
  const [lockMode, setLockMode] = useState("none");

  const { data: plans } = useQuery({
    queryKey: ["admin-plans"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/admin/plans")),
    enabled: !!instance,
  });

  useEffect(() => {
    if (!instance) return;
    setExpiresAt(instance.expires_at ? String(instance.expires_at).slice(0, 10) : "");
    setPlanId((instance.plan_id as string) ?? "");
    setLockMode((instance.lock_mode as string) ?? "none");
  }, [instance]);

  const mutation = useMutation({
    mutationFn: () =>
      api.patch(`/admin/instances/${instance!.id}`, {
        expires_at: expiresAt ? new Date(expiresAt + "T00:00:00Z").toISOString() : "",
        plan_id: planId,
        lock_mode: lockMode,
      }),
    onSuccess: () => {
      toast.success(t("admin.instances.editSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-instances"] });
      onClose();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.instances.editFailed"))),
  });

  if (!instance) return null;

  return (
    <Modal
      open={!!instance}
      onClose={onClose}
      title={t("admin.instances.editTitle", { name: instance.container_name ?? instance.id })}
    >
      <div className="space-y-4">
        <Input
          label={t("admin.instances.expiresAtLabel")}
          type="date"
          dir="ltr"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
        />
        <p className="-mt-2 text-xs text-slate-400">{t("admin.instances.expiresAtHint")}</p>

        <Select label={t("admin.instances.planLabel")} value={planId} onChange={(e) => setPlanId(e.target.value)}>
          <option value="">{t("admin.instances.noPlanOption")}</option>
          {(plans ?? []).map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>

        <Select label={t("admin.instances.lockModeLabel")} value={lockMode} onChange={(e) => setLockMode(e.target.value)}>
          {LOCK_MODES.map((m) => (
            <option key={m} value={m}>
              {t(`admin.instances.lockMode_${m}`)}
            </option>
          ))}
        </Select>

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button type="button" loading={mutation.isPending} onClick={() => mutation.mutate()}>
            {t("common.save")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * انتقالِ instance به سرور دیگر (بازخورد کاربر ۲۰۲۶-۰۷-۰۵): سرورِ فعلی از لیست کنار گذاشته
 * می‌شود و فقط سرورهای آنلاین قابل‌انتخاب‌اند. توضیح می‌دهد که کانتینرِ قدیمی روی سرورِ قبلی
 * فقط stop می‌شود، نه حذف — تا اگر deploy روی سرور جدید مشکلی داشت، راه برگشتی باشد.
 */
function MigrateModal({
  instance,
  servers,
  onClose,
}: {
  instance: BotInstance | null;
  servers: Server[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [targetServerId, setTargetServerId] = useState("");

  const candidates = servers.filter((s) => s.id !== instance?.server_id && s.is_online);

  const mutation = useMutation({
    mutationFn: (serverId: string) => api.post(`/admin/instances/${instance!.id}/migrate`, { server_id: serverId }),
    onSuccess: () => {
      toast.success(t("admin.instances.migrateSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-instances"] });
      onClose();
      setTargetServerId("");
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.instances.migrateFailed"))),
  });

  if (!instance) return null;

  return (
    <Modal
      open={!!instance}
      onClose={onClose}
      title={t("admin.instances.migrateTitle", { name: instance.container_name ?? instance.id })}
    >
      <div className="space-y-4">
        <p className="text-xs text-slate-400">{t("admin.instances.migrateHint")}</p>
        {candidates.length === 0 ? (
          <p className="text-sm text-slate-500 dark:text-slate-400">{t("admin.instances.migrateNoTargets")}</p>
        ) : (
          <Select
            label={t("admin.instances.migrateTargetLabel")}
            value={targetServerId}
            onChange={(e) => setTargetServerId(e.target.value)}
          >
            <option value="" disabled>
              {t("admin.instances.migrateTargetPlaceholder")}
            </option>
            {candidates.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} ({s.containers?.length ?? 0}/{s.max_containers && s.max_containers > 0 ? s.max_containers : "∞"})
              </option>
            ))}
          </Select>
        )}
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            disabled={!targetServerId}
            loading={mutation.isPending}
            onClick={() => mutation.mutate(targetServerId)}
          >
            <ArrowRightLeft className="h-4 w-4" />
            {t("admin.instances.migrateButton")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/** نسخه‌ی ادمینِ همان مودالِ تنظیمات که در Instances.tsx برای خودِ کاربر هست — بک‌اند
 * (GET/PUT /instances/:id/settings) برای نقش admin/owner چک مالکیت را کنار می‌گذارد. */
function SettingsModal({ instance, onClose }: { instance: BotInstance | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-instance-settings", instance?.id],
    queryFn: async () =>
      unwrap<{ schema: TemplateConfigField[]; values: Record<string, string> }>(
        await api.get(`/instances/${instance!.id}/settings`)
      ),
    enabled: !!instance,
  });

  const [values, setValues] = useState<Record<string, string>>({});
  useEffect(() => {
    setValues(data?.values ?? {});
  }, [data]);

  const mutation = useMutation({
    mutationFn: (vals: Record<string, string>) => api.put(`/instances/${instance!.id}/settings`, { values: vals }),
    onSuccess: () => {
      toast.success(t("instances.settingsSaveSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-instance-settings", instance?.id] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("instances.settingsSaveFailed"))),
  });

  if (!instance) return null;
  const schema = data?.schema ?? [];

  return (
    <Modal
      open={!!instance}
      onClose={onClose}
      title={t("instances.settingsTitle", { name: instance.container_name ?? instance.id })}
    >
      {isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
        </div>
      )}
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {apiErrorMessage(error, t("instances.settingsLoadFailed"))}
        </p>
      )}
      {!isLoading && !error && schema.length === 0 && (
        <p className="text-sm text-slate-400">{t("instances.noSettingsForThisBot")}</p>
      )}
      {!isLoading && !error && schema.length > 0 && (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate(values);
          }}
        >
          {schema.map((field) => (
            <AdminSettingField
              key={field.key}
              field={field}
              value={values[field.key] ?? ""}
              onChange={(v) => setValues((prev) => ({ ...prev, [field.key]: v }))}
            />
          ))}
          <p className="text-xs text-slate-400">{t("instances.settingsApplyHint")}</p>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {t("common.close")}
            </Button>
            <Button type="submit" loading={mutation.isPending}>
              {t("common.save")}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}

function AdminSettingField({
  field,
  value,
  onChange,
}: {
  field: TemplateConfigField;
  value: string;
  onChange: (v: string) => void;
}) {
  const { t } = useTranslation();
  if (field.type === "boolean") {
    return (
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={value === "true"}
          onChange={(e) => onChange(e.target.checked ? "true" : "false")}
          className="h-4 w-4 rounded border-slate-300"
        />
        {field.label}
        {field.required && <span className="text-red-500"> *</span>}
      </label>
    );
  }
  if (field.type === "select") {
    return (
      <Select label={field.label + (field.required ? " *" : "")} value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="" disabled>
          {t("instances.settingsSelectPlaceholder")}
        </option>
        {(field.options ?? []).map((opt) => (
          <option key={opt} value={opt}>
            {opt}
          </option>
        ))}
      </Select>
    );
  }
  return (
    <Input
      label={field.label + (field.required ? " *" : "")}
      type={field.type === "number" ? "number" : "text"}
      dir="ltr"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}

function LogsModal({ instance, onClose }: { instance: BotInstance | null; onClose: () => void }) {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-instance-logs", instance?.id],
    queryFn: async () => unwrap<{ logs: string }>(await api.get(`/instances/${instance!.id}/logs`)),
    enabled: !!instance,
  });

  if (!instance) return null;

  return (
    <Modal
      open={!!instance}
      onClose={onClose}
      title={t("instances.logsTitle", { name: instance.container_name ?? instance.id })}
    >
      <div className="max-h-96 overflow-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-100">
        {isLoading && <p className="text-slate-400">{t("instances.logsLoading")}</p>}
        {error && <p className="text-red-400">{apiErrorMessage(error, t("instances.logsFailed"))}</p>}
        {!isLoading && !error && (
          <pre className="whitespace-pre-wrap" dir="ltr">
            {data?.logs || t("instances.logsEmpty")}
          </pre>
        )}
      </div>
    </Modal>
  );
}
