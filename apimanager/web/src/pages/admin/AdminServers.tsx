import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { ServerCog, Plus, Trash2, Bot, Info, Pencil, Tag } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { StatusBadge } from "@/components/ui/Badge";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState, EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatDate, formatDuration } from "@/lib/format";
import type { Server, BotInstance } from "@/lib/types";

interface CreateForm {
  name: string;
  ip: string;
  tags: string;
  max_containers: number;
}

interface EditForm {
  name: string;
  ip: string;
  tags: string;
  max_containers: number;
}

function parseTagsInput(value: string): string[] {
  return value
    .split(",")
    .map((t) => t.trim().toLowerCase())
    .filter(Boolean);
}

/**
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۳: «بخش سرورها رو درست کن، رم/سی‌پی‌یو/تایم آنلاین/همه‌چیز رو نمایش بده».
 * چیزهایی که این‌جا اضافه شد:
 *  - «مدت آنلاین بودن» واقعی (online_seconds) — این یکی کاملاً واقعی و همیشه در دسترس است،
 *    چون خودِ apimanager transition آفلاین→آنلاین را مدیریت می‌کند.
 *  - لیست containers واقعیِ هر سرور طبق آخرین heartbeat — این داده از قبل به apimanager
 *    می‌رسید ولی هیچ‌جا ذخیره/نمایش داده نمی‌شد.
 *  - ستون‌های CPU/RAM — این دو فقط اگر agentmanager (که در این workspace نیست) در heartbeat
 *    بفرستد مقدار دارند؛ نسخه‌ی فعلی agentmanager این‌ها را نمی‌فرستد، پس همیشه
 *    «گزارش نشده» نشان داده می‌شوند تا زمانی که agentmanager ارتقا پیدا کند — به‌جای
 *    نمایش یک صفر یا عدد ساختگی که گمراه‌کننده باشد.
 *  - رفعِ یک باگِ واقعیِ از قبل موجود: جاروبِ دوره‌ای «سرورهایی که heartbeat شان قطع شده
 *    آفلاین علامت بخور» (MarkStaleServersOffline) در کد وجود داشت ولی هیچ‌جا صدا زده
 *    نمی‌شد — یعنی is_online بعد از قطع واقعیِ heartbeat هیچ‌وقت false نمی‌شد. این‌جا
 *    (سمت بک‌اند) با یک goroutine هر ۲۰ ثانیه فعال شد.
 *
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «یه تگ اضافه کن، مثلاً free — فقط پنل‌های free به این سرور
 * بیاد. تعداد مجاز کانتینر رو هم بتونیم اضافه کنیم». اضافه شد: تگ‌ها (چندتایی، comma-separated)
 * + سقف container؛ سمت بک‌اند SelectLeastLoadedServer این‌ها را در انتخاب سرورِ مقصدِ یک
 * instance جدید رعایت می‌کند (تمپلیت رایگان → اول سراغ سرور تگ‌خورده با "free"،
 * تعداد آنلاین instance ≥ سقف → آن سرور اصلاً انتخاب نمی‌شود). قبلاً هیچ راه ویرایشی برای
 * سرور نبود (فقط ساخت/حذف) — یک مودال ویرایش هم اضافه شد.
 */
export default function AdminServers() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [detailFor, setDetailFor] = useState<Server | null>(null);
  const [editing, setEditing] = useState<Server | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-servers"],
    queryFn: async () => unwrap<Server[]>(await api.get("/admin/servers")),
    refetchInterval: 20_000,
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateForm>({ defaultValues: { max_containers: 0 } });

  const createMutation = useMutation({
    mutationFn: (values: CreateForm) =>
      api.post("/admin/servers", {
        name: values.name,
        ip: values.ip,
        tags: parseTagsInput(values.tags),
        max_containers: Number(values.max_containers) || 0,
      }),
    onSuccess: () => {
      toast.success(t("admin.servers.addSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-servers"] });
      setCreateOpen(false);
      reset();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.servers.addFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/servers/${id}`),
    onSuccess: () => {
      toast.success(t("admin.servers.deleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-servers"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.servers.deleteFailed"))),
  });

  const columns: DataTableColumn<Server>[] = [
    {
      key: "name",
      header: t("admin.servers.colName"),
      sortValue: (row) => row.name,
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      key: "ip",
      header: t("admin.servers.colIp"),
      sortValue: (row) => row.ip,
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.ip}
        </span>
      ),
    },
    {
      key: "status",
      header: t("admin.servers.colStatus"),
      sortValue: (row) => (row.is_online ? 1 : 0),
      cell: (row) => (
        <div className="space-y-0.5">
          <StatusBadge status={row.is_online ? "online" : "offline"} size="sm" />
          {row.is_online && (
            <p className="text-[11px] text-slate-400">
              {t("admin.servers.onlineFor", { duration: formatDuration(row.online_seconds) })}
            </p>
          )}
        </div>
      ),
    },
    {
      key: "cpu",
      header: t("admin.servers.colCpu"),
      sortValue: (row) => row.cpu_percent ?? -1,
      cell: (row) =>
        row.cpu_percent != null ? (
          <span className="tabular-nums" dir="ltr">
            {row.cpu_percent.toFixed(0)}%
          </span>
        ) : (
          <span className="text-xs text-slate-400">{t("admin.servers.notReported")}</span>
        ),
    },
    {
      key: "ram",
      header: t("admin.servers.colRam"),
      sortValue: (row) => row.memory_used_mb ?? -1,
      cell: (row) =>
        row.memory_used_mb != null && row.memory_total_mb != null ? (
          <span className="tabular-nums" dir="ltr">
            {row.memory_used_mb} / {row.memory_total_mb} MB
          </span>
        ) : (
          <span className="text-xs text-slate-400">{t("admin.servers.notReported")}</span>
        ),
    },
    {
      key: "tags",
      header: t("admin.servers.colTags"),
      cell: (row) =>
        row.tags && row.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {row.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 rounded-full bg-violet-50 px-2 py-0.5 text-[11px] font-medium text-violet-700 dark:bg-violet-500/15 dark:text-violet-300"
              >
                <Tag className="h-2.5 w-2.5" />
                {tag}
              </span>
            ))}
          </div>
        ) : (
          <span className="text-xs text-slate-400">{t("admin.servers.noTags")}</span>
        ),
    },
    {
      key: "containers",
      header: t("admin.servers.colContainers"),
      sortValue: (row) => row.containers?.length ?? 0,
      cell: (row) => (
        <span className="tabular-nums" dir="ltr">
          {row.containers?.length ?? 0} / {row.max_containers && row.max_containers > 0 ? row.max_containers : "∞"}
        </span>
      ),
    },
    {
      key: "last_seen",
      header: t("admin.servers.colLastSeen"),
      sortValue: (row) => row.last_seen ?? "",
      cell: (row) => <span className="text-xs text-slate-400">{formatDate(row.last_seen)}</span>,
    },
    {
      key: "actions",
      header: t("admin.servers.colActions"),
      cell: (row) => (
        <div className="flex items-center gap-1">
          <button
            onClick={() => setDetailFor(row)}
            aria-label={t("admin.servers.viewDetail")}
            title={t("admin.servers.viewDetail")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Info className="h-4 w-4" />
          </button>
          <button
            onClick={() => setEditing(row)}
            aria-label={t("admin.servers.editButton")}
            title={t("admin.servers.editButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Pencil className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              if (confirm(t("admin.servers.deleteConfirm", { name: row.name }))) deleteMutation.mutate(row.id);
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
          <h2 className="text-xl font-bold">{t("admin.servers.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.servers.subtitle")}</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("admin.servers.addButton")}
        </Button>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.servers.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<ServerCog className="h-8 w-8" />}
          emptyTitle={t("admin.servers.emptyTitle")}
          searchPlaceholder={t("admin.servers.searchPlaceholder")}
          searchFn={(row, q) =>
            row.name.toLowerCase().includes(q.toLowerCase()) || row.ip.toLowerCase().includes(q.toLowerCase())
          }
        />
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title={t("admin.servers.addTitle")}>
        <form className="space-y-4" onSubmit={handleSubmit((values) => createMutation.mutate(values))}>
          <Input
            label={t("admin.servers.nameLabel")}
            {...register("name", { required: t("admin.servers.nameRequired") })}
            error={errors.name?.message}
          />
          <Input
            label={t("admin.servers.ipLabel")}
            dir="ltr"
            placeholder="203.0.113.10"
            {...register("ip", { required: t("admin.servers.ipRequired") })}
            error={errors.ip?.message}
          />
          <div>
            <Input
              label={t("admin.servers.tagsLabel")}
              dir="ltr"
              placeholder="free, eu"
              {...register("tags")}
            />
            <p className="mt-1 text-xs text-slate-400">{t("admin.servers.tagsHint")}</p>
          </div>
          <Input
            label={t("admin.servers.maxContainersLabel")}
            type="number"
            dir="ltr"
            {...register("max_containers", { valueAsNumber: true, min: 0 })}
          />
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

      <EditServerModal server={editing} onClose={() => setEditing(null)} />
      <ServerDetailModal server={detailFor} onClose={() => setDetailFor(null)} />
    </div>
  );
}

function EditServerModal({ server, onClose }: { server: Server | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<EditForm>({
    values: server
      ? {
          name: server.name,
          ip: server.ip,
          tags: (server.tags ?? []).join(", "),
          max_containers: server.max_containers ?? 0,
        }
      : undefined,
  });

  const mutation = useMutation({
    mutationFn: (values: EditForm) =>
      api.patch(`/admin/servers/${server!.id}`, {
        name: values.name,
        ip: values.ip,
        tags: parseTagsInput(values.tags),
        max_containers: Number(values.max_containers) || 0,
      }),
    onSuccess: () => {
      toast.success(t("admin.servers.editSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-servers"] });
      onClose();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.servers.editFailed"))),
  });

  if (!server) return null;

  return (
    <Modal open={!!server} onClose={onClose} title={t("admin.servers.editTitle", { name: server.name })}>
      <form className="space-y-4" onSubmit={handleSubmit((values) => mutation.mutate(values))}>
        <Input label={t("admin.servers.nameLabel")} {...register("name", { required: true })} error={errors.name?.message} />
        <Input label={t("admin.servers.ipLabel")} dir="ltr" {...register("ip", { required: true })} />
        <div>
          <Input label={t("admin.servers.tagsLabel")} dir="ltr" placeholder="free, eu" {...register("tags")} />
          <p className="mt-1 text-xs text-slate-400">{t("admin.servers.tagsHint")}</p>
        </div>
        <Input
          label={t("admin.servers.maxContainersLabel")}
          type="number"
          dir="ltr"
          {...register("max_containers", { valueAsNumber: true, min: 0 })}
        />
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" loading={isSubmitting || mutation.isPending}>
            {t("common.save")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function ServerDetailModal({ server, onClose }: { server: Server | null; onClose: () => void }) {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-server-instances", server?.id],
    queryFn: async () => unwrap<BotInstance[]>(await api.get(`/admin/servers/${server!.id}/instances`)),
    enabled: !!server,
  });

  if (!server) return null;

  return (
    <Modal open={!!server} onClose={onClose} title={t("admin.servers.detailTitle", { name: server.name })}>
      <div className="space-y-5">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <dt className="text-slate-400">{t("admin.servers.colStatus")}</dt>
          <dd className="text-end">
            <StatusBadge status={server.is_online ? "online" : "offline"} size="sm" />
          </dd>
          <dt className="text-slate-400">{t("admin.servers.onlineDurationLabel")}</dt>
          <dd className="text-end font-medium">
            {server.is_online ? formatDuration(server.online_seconds) : "—"}
          </dd>
          <dt className="text-slate-400">{t("admin.servers.colLastSeen")}</dt>
          <dd className="text-end font-medium">{formatDate(server.last_seen)}</dd>
          <dt className="text-slate-400">{t("admin.servers.colCpu")}</dt>
          <dd className="text-end font-medium">
            {server.cpu_percent != null ? `${server.cpu_percent.toFixed(0)}%` : t("admin.servers.notReported")}
          </dd>
          <dt className="text-slate-400">{t("admin.servers.colRam")}</dt>
          <dd className="text-end font-medium">
            {server.memory_used_mb != null && server.memory_total_mb != null
              ? `${server.memory_used_mb} / ${server.memory_total_mb} MB`
              : t("admin.servers.notReported")}
          </dd>
          <dt className="text-slate-400">{t("admin.servers.colContainers")}</dt>
          <dd className="text-end font-medium" dir="ltr">
            {server.containers?.length ?? 0} / {server.max_containers && server.max_containers > 0 ? server.max_containers : "∞"}
          </dd>
          <dt className="text-slate-400">{t("admin.servers.colTags")}</dt>
          <dd className="text-end">
            {server.tags && server.tags.length > 0 ? (
              <div className="flex flex-wrap justify-end gap-1">
                {server.tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 rounded-full bg-violet-50 px-2 py-0.5 text-[11px] font-medium text-violet-700 dark:bg-violet-500/15 dark:text-violet-300"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            ) : (
              <span className="text-xs text-slate-400">{t("admin.servers.noTags")}</span>
            )}
          </dd>
        </dl>

        <div>
          <p className="mb-2 text-xs font-medium text-slate-500 dark:text-slate-400">
            {t("admin.servers.dbInstancesLabel")} ({data?.length ?? 0})
          </p>
          {isLoading && (
            <div className="space-y-2">
              <Skeleton className="h-9" />
              <Skeleton className="h-9" />
            </div>
          )}
          {error && (
            <p className="text-sm text-red-600 dark:text-red-400">
              {apiErrorMessage(error, t("admin.servers.instancesLoadFailed"))}
            </p>
          )}
          {!isLoading && !error && (!data || data.length === 0) && (
            <EmptyState icon={<Bot className="h-6 w-6" />} title={t("instances.emptyTitle")} />
          )}
          {!isLoading && !error && data && data.length > 0 && (
            <ul className="max-h-48 space-y-1.5 overflow-auto">
              {data.map((inst) => (
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

        <div>
          <p className="mb-2 text-xs font-medium text-slate-500 dark:text-slate-400">
            {t("admin.servers.liveContainersLabel")} ({server.containers?.length ?? 0})
          </p>
          <p className="mb-2 text-[11px] text-slate-400">{t("admin.servers.liveContainersHint")}</p>
          {!server.containers || server.containers.length === 0 ? (
            <p className="text-xs text-slate-400">{t("admin.servers.noLiveContainers")}</p>
          ) : (
            <ul className="max-h-48 space-y-1.5 overflow-auto">
              {server.containers.map((c) => (
                <li key={c.name} className="rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-white/10">
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs">{c.name}</span>
                    <StatusBadge status={c.state} size="sm" />
                  </div>
                  <p className="mt-1 text-[11px] text-slate-400" dir="ltr">
                    {c.image} — {c.status}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </Modal>
  );
}
