import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, Controller } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { Bot, Plus, Play, Square, RotateCw, Trash2, ScrollText, Settings, FolderOpen, FolderPlus, KeyRound } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input, Select } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { StatusBadge } from "@/components/ui/Badge";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ActionMenu } from "@/components/ui/ActionMenu";
import { Skeleton } from "@/components/ui/Skeleton";
import type { BotInstance, BotTemplate, TemplateConfigField, UploaderFolder, UploaderCode } from "@/lib/types";

interface CreateForm {
  bot_token: string;
  service_type: string;
  template_id: string;
}

const STATUS_OPTIONS = ["running", "pending", "stopped", "error"];

export default function Instances() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [logsFor, setLogsFor] = useState<BotInstance | null>(null);
  const [settingsFor, setSettingsFor] = useState<BotInstance | null>(null);
  const [contentFor, setContentFor] = useState<BotInstance | null>(null);
  const [pendingAction, setPendingAction] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["instances"],
    queryFn: async () => unwrap<BotInstance[]>(await api.get("/instances")),
  });

  const createMutation = useMutation({
    mutationFn: (payload: { bot_token: string; template_id: string }) => api.post("/instances", payload),
    onSuccess: () => {
      toast.success(t("instances.createSuccess"));
      queryClient.invalidateQueries({ queryKey: ["instances"] });
      setCreateOpen(false);
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("instances.createFailed"))),
  });

  async function runAction(id: string, action: "start" | "stop" | "restart" | "delete") {
    setPendingAction(id + action);
    try {
      if (action === "delete") {
        await api.delete(`/instances/${id}`);
        toast.success(t("instances.deleteSuccess"));
      } else {
        await api.post(`/instances/${id}/${action}`);
        toast.success(t("instances.actionSuccessGeneric"));
      }
      queryClient.invalidateQueries({ queryKey: ["instances"] });
    } catch (err) {
      toast.error(apiErrorMessage(err, t("instances.actionFailed")));
    } finally {
      setPendingAction(null);
    }
  }

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
      key: "status",
      header: t("instances.colStatus"),
      sortValue: (row) => row.status ?? "",
      cell: (row) => <StatusBadge status={row.status} size="sm" />,
    },
    {
      key: "actions",
      header: t("instances.colActions"),
      cell: (row) => (
        <div className="flex items-center gap-1">
          <IconAction
            label={t("instances.actionStart")}
            icon={Play}
            loading={pendingAction === row.id + "start"}
            onClick={() => runAction(row.id, "start")}
          />
          <IconAction
            label={t("instances.actionStop")}
            icon={Square}
            loading={pendingAction === row.id + "stop"}
            onClick={() => runAction(row.id, "stop")}
          />
          <IconAction
            label={t("instances.actionRestart")}
            icon={RotateCw}
            loading={pendingAction === row.id + "restart"}
            onClick={() => runAction(row.id, "restart")}
          />
          <ActionMenu
            ariaLabel={t("common.actions")}
            items={[
              { label: t("instances.actionSettings"), icon: Settings, onClick: () => setSettingsFor(row) },
              { label: t("instances.actionContent"), icon: FolderOpen, onClick: () => setContentFor(row) },
              { label: t("instances.actionLogs"), icon: ScrollText, onClick: () => setLogsFor(row) },
              {
                label: t("instances.actionDelete"),
                icon: Trash2,
                danger: true,
                onClick: () => {
                  if (confirm(t("instances.deleteConfirm"))) runAction(row.id, "delete");
                },
              },
            ]}
          />
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold">{t("instances.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("instances.subtitle")}</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("instances.newButton")}
        </Button>
      </div>

      {error ? (
        <div className="glass-card p-4 text-sm text-red-700 dark:text-fuchsia-300">
          {apiErrorMessage(error, t("instances.loadFailed"))}
          <button onClick={() => refetch()} className="ms-2 underline underline-offset-2">
            {t("common.retry")}
          </button>
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={filtered}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<Bot className="h-8 w-8" />}
          emptyTitle={t("instances.emptyTitle")}
          emptyDescription={t("instances.emptyDescription")}
          emptyAction={<Button onClick={() => setCreateOpen(true)}>{t("instances.newButton")}</Button>}
          searchPlaceholder={t("instances.searchPlaceholder")}
          searchFn={(row, q) =>
            (row.container_name ?? "").toLowerCase().includes(q.toLowerCase()) ||
            String(row.bot_id ?? row.id).toLowerCase().includes(q.toLowerCase())
          }
          toolbarExtra={
            data && data.length > 0 ? (
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-auto"
              >
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

      <CreateInstanceModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSubmit={(values) => createMutation.mutate(values)}
        submitting={createMutation.isPending}
      />

      <LogsModal instance={logsFor} onClose={() => setLogsFor(null)} />
      <SettingsModal instance={settingsFor} onClose={() => setSettingsFor(null)} />
      <ContentModal instance={contentFor} onClose={() => setContentFor(null)} />
    </div>
  );
}

function CreateInstanceModal({
  open,
  onClose,
  onSubmit,
  submitting,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: { bot_token: string; template_id: string }) => void;
  submitting: boolean;
}) {
  const { t } = useTranslation();
  const {
    register,
    handleSubmit,
    control,
    watch,
    reset,
    setValue,
    formState: { errors },
  } = useForm<CreateForm>({ defaultValues: { bot_token: "", service_type: "", template_id: "" } });

  const serviceType = watch("service_type");

  const { data: serviceTypes, isLoading: typesLoading } = useQuery({
    queryKey: ["service-types"],
    queryFn: async () => unwrap<string[]>(await api.get("/service-types")),
    enabled: open,
  });

  const { data: templates, isLoading: templatesLoading } = useQuery({
    queryKey: ["templates-by-type", serviceType],
    queryFn: async () => unwrap<BotTemplate[]>(await api.get("/templates", { params: { type: serviceType } })),
    enabled: open && !!serviceType,
  });

  // وقتی نوع سرویس عوض می‌شود، انتخاب قبلیِ تمپلیت دیگر معتبر نیست.
  useEffect(() => {
    setValue("template_id", "");
  }, [serviceType, setValue]);

  useEffect(() => {
    if (!open) reset();
  }, [open, reset]);

  return (
    <Modal open={open} onClose={onClose} title={t("instances.createTitle")}>
      <form
        className="space-y-4"
        onSubmit={handleSubmit((values) =>
          onSubmit({ bot_token: values.bot_token, template_id: values.template_id })
        )}
      >
        <Input
          label={t("instances.botTokenLabel")}
          placeholder="123456:ABC-..."
          dir="ltr"
          {...register("bot_token", { required: t("instances.botTokenRequired") })}
          error={errors.bot_token?.message}
        />

        <Controller
          control={control}
          name="service_type"
          rules={{ required: t("instances.serviceTypeRequired") }}
          render={({ field }) => (
            <Select label={t("instances.serviceTypeLabel")} {...field} error={errors.service_type?.message}>
              <option value="" disabled>
                {typesLoading ? t("common.loading") : t("instances.serviceTypePlaceholder")}
              </option>
              {serviceTypes?.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </Select>
          )}
        />

        {serviceType && (
          <Controller
            control={control}
            name="template_id"
            rules={{ required: t("instances.templateIdRequired") }}
            render={({ field }) =>
              templatesLoading ? (
                <Skeleton className="h-16" />
              ) : templates && templates.length > 0 ? (
                <Select label={t("instances.templateLabel")} {...field} error={errors.template_id?.message}>
                  <option value="" disabled>
                    {t("instances.templatePlaceholder")}
                  </option>
                  {templates.map((tmpl) => (
                    <option key={tmpl.id} value={tmpl.id}>
                      {tmpl.name} ({tmpl.image_tag})
                    </option>
                  ))}
                </Select>
              ) : (
                <p className="text-xs text-slate-400">{t("instances.noTemplatesForType")}</p>
              )
            }
          />
        )}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" loading={submitting}>
            {t("instances.newButton")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function IconAction({
  label,
  icon: Icon,
  onClick,
  loading,
}: {
  label: string;
  icon: typeof Play;
  onClick: () => void;
  loading?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      title={label}
      aria-label={label}
      className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 disabled:opacity-50 dark:text-slate-400 dark:hover:bg-white/10"
    >
      <Icon className="h-4 w-4" />
    </button>
  );
}

/**
 * صفحه/مودالِ جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۳: «برای تمپلیت‌ها چیزهایی که کاربر می‌تواند
 * تغییر بدهد را از او بگیر و ذخیره کن»). فیلدهای قابل‌تنظیم را ادمین روی خودِ قالب تعریف
 * می‌کند (AdminTemplates)؛ این‌جا مالکِ ربات مقدارِ آن‌ها را برای instance خودش وارد و
 * ذخیره می‌کند. اگر قالب هیچ فیلدی تعریف نکرده باشد، پیام «تنظیماتی وجود ندارد» نشان داده
 * می‌شود. مقادیر تا restart بعدیِ ربات روی کانتینر واقعی اعمال نمی‌شوند — همین‌جا هم گفته شده.
 */
function SettingsModal({ instance, onClose }: { instance: BotInstance | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["instance-settings", instance?.id],
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
      queryClient.invalidateQueries({ queryKey: ["instance-settings", instance?.id] });
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
      {error && <p className="text-sm text-red-600 dark:text-red-400">{apiErrorMessage(error, t("instances.settingsLoadFailed"))}</p>}
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
            <SettingField
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

function SettingField({
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
    queryKey: ["instance-logs", instance?.id],
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

/**
 * مدیریتِ محتوای uploader-bot (کدها/پوشه‌ها) از پنل وب — بازخورد کاربر ۲۰۲۶-۰۷-۰۵ (رجوع
 * apimanager/NEEDS.md، بخش «از uploader-bot»). فقط برای instance هایی که واقعاً از نوع
 * uploader باشند معنا دارد؛ چون این صفحه نمی‌داند نوع هر instance چیست (BotInstance آن را
 * حمل نمی‌کند)، عمداً همیشه در دسترسِ همه‌ی instance هاست و بک‌اند خودش با پیام روشن رد
 * می‌کند اگر نوع ربات uploader نباشد — به‌جای پیچیده‌کردنِ فرانت‌اند برای این تشخیص.
 */
function ContentModal({ instance, onClose }: { instance: BotInstance | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [folderId, setFolderId] = useState("");
  const [page, setPage] = useState(1);
  const [newFolderName, setNewFolderName] = useState("");

  const foldersQuery = useQuery({
    queryKey: ["uploader-folders", instance?.id],
    queryFn: async () => unwrap<{ folders: UploaderFolder[] }>(await api.get(`/instances/${instance!.id}/uploader/folders`)),
    enabled: !!instance,
  });

  const codesQuery = useQuery({
    queryKey: ["uploader-codes", instance?.id, folderId, page],
    queryFn: async () =>
      unwrap<{ codes: UploaderCode[]; total: number }>(
        await api.get(`/instances/${instance!.id}/uploader/codes`, { params: { folder_id: folderId, page, limit: 20 } })
      ),
    enabled: !!instance,
  });

  const createFolderMutation = useMutation({
    mutationFn: (name: string) => api.post(`/instances/${instance!.id}/uploader/folders`, { name, parent_id: "" }),
    onSuccess: () => {
      toast.success(t("instances.folderAddSuccess"));
      queryClient.invalidateQueries({ queryKey: ["uploader-folders", instance?.id] });
      setNewFolderName("");
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("instances.folderAddFailed"))),
  });

  const deleteFolderMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/instances/${instance!.id}/uploader/folders/${id}`),
    onSuccess: () => {
      toast.success(t("instances.folderDeleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["uploader-folders", instance?.id] });
      if (folderId) setFolderId("");
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("instances.folderDeleteFailed"))),
  });

  const deleteCodeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/instances/${instance!.id}/uploader/codes/${id}`),
    onSuccess: () => {
      toast.success(t("instances.codeDeleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["uploader-codes", instance?.id] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("instances.codeDeleteFailed"))),
  });

  if (!instance) return null;
  const notUploaderError =
    (foldersQuery.error && apiErrorMessage(foldersQuery.error, "")) ||
    (codesQuery.error && apiErrorMessage(codesQuery.error, ""));

  return (
    <Modal
      open={!!instance}
      onClose={onClose}
      title={t("instances.contentTitle", { name: instance.container_name ?? instance.id })}
    >
      {notUploaderError ? (
        <p className="text-sm text-red-600 dark:text-red-400">{notUploaderError}</p>
      ) : (
        <div className="space-y-5">
          <div>
            <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <FolderOpen className="h-3.5 w-3.5" />
              {t("instances.foldersLabel")}
            </p>
            {foldersQuery.isLoading ? (
              <Skeleton className="h-9" />
            ) : (
              <Select
                value={folderId}
                onChange={(e) => {
                  setFolderId(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">{t("instances.rootFolderOption")}</option>
                {(foldersQuery.data?.folders ?? []).map((f) => (
                  <option key={f.id} value={f.id}>
                    {f.name}
                  </option>
                ))}
              </Select>
            )}
            {folderId && (
              <button
                onClick={() => {
                  if (confirm(t("instances.folderDeleteConfirm"))) deleteFolderMutation.mutate(folderId);
                }}
                className="mt-1.5 flex items-center gap-1 text-xs font-medium text-red-500 hover:underline"
              >
                <Trash2 className="h-3 w-3" />
                {t("instances.folderDeleteButton")}
              </button>
            )}
            <div className="mt-2 flex gap-2">
              <Input
                placeholder={t("instances.newFolderPlaceholder")}
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                className="flex-1"
              />
              <Button
                type="button"
                size="sm"
                variant="secondary"
                disabled={!newFolderName.trim()}
                loading={createFolderMutation.isPending}
                onClick={() => createFolderMutation.mutate(newFolderName.trim())}
              >
                <FolderPlus className="h-3.5 w-3.5" />
                {t("common.add")}
              </Button>
            </div>
          </div>

          <div>
            <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <KeyRound className="h-3.5 w-3.5" />
              {t("instances.codesLabel")} ({codesQuery.data?.total ?? 0})
            </p>
            {codesQuery.isLoading && (
              <div className="space-y-2">
                <Skeleton className="h-9" />
                <Skeleton className="h-9" />
              </div>
            )}
            {!codesQuery.isLoading && (codesQuery.data?.codes.length ?? 0) === 0 && (
              <p className="text-xs text-slate-400">{t("instances.codesEmpty")}</p>
            )}
            {!codesQuery.isLoading && (codesQuery.data?.codes.length ?? 0) > 0 && (
              <ul className="max-h-56 space-y-1.5 overflow-auto">
                {codesQuery.data!.codes.map((code) => (
                  <li
                    key={code.id}
                    className="flex items-center justify-between gap-2 rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-white/10"
                  >
                    <div>
                      <p className="font-mono text-xs" dir="ltr">
                        {code.code}
                      </p>
                      <p className="text-[11px] text-slate-400">
                        {t(`instances.codeType_${code.type}`, { defaultValue: code.type })} · {code.used_count}/
                        {code.max_use || "∞"}
                      </p>
                    </div>
                    <button
                      onClick={() => {
                        if (confirm(t("instances.codeDeleteConfirm", { code: code.code }))) deleteCodeMutation.mutate(code.id);
                      }}
                      className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
            {(codesQuery.data?.total ?? 0) > 20 && (
              <div className="mt-2 flex items-center justify-between text-xs">
                <Button size="sm" variant="secondary" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                  {t("common.previous")}
                </Button>
                <span className="text-slate-400">{t("common.pageOf", { current: page, total: Math.ceil((codesQuery.data?.total ?? 0) / 20) })}</span>
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={page >= Math.ceil((codesQuery.data?.total ?? 0) / 20)}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t("common.next")}
                </Button>
              </div>
            )}
          </div>

          <div className="flex justify-end pt-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {t("common.close")}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}
