import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, Controller } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { LayoutTemplate, Plus, Copy, Pencil, Trash2, FileUp } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input, Select } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState } from "@/components/ui/EmptyState";
import { parseTemplateYaml } from "@/lib/template-yaml";
import type { BotTemplate, TemplateConfigField } from "@/lib/types";

const CUSTOM_TYPE_VALUE = "__custom__";

interface CreateForm {
  name: string;
  type: string;
  image_name: string;
  image_tag: string;
}

interface EditForm {
  name: string;
  image_name: string;
  image_tag: string;
  is_active: boolean;
}

export default function AdminTemplates() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [createSchema, setCreateSchema] = useState<TemplateConfigField[]>([]);
  const [editing, setEditing] = useState<BotTemplate | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-templates"],
    queryFn: async () => unwrap<BotTemplate[]>(await api.get("/admin/templates")),
  });

  const {
    register,
    handleSubmit,
    control,
    setValue,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateForm>({ defaultValues: { type: "" } });

  const { data: serviceTypes } = useQuery({
    queryKey: ["service-types"],
    queryFn: async () => unwrap<string[]>(await api.get("/service-types")),
    enabled: createOpen,
  });

  const yamlInputRef = useRef<HTMLInputElement>(null);

  function handleYamlFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // اجازه بده همون فایل دوباره هم انتخاب بشه
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const manifest = parseTemplateYaml(String(reader.result ?? ""));
        if (!manifest.name && !manifest.type && !manifest.image_name) {
          toast.error(t("admin.templates.yamlImportEmpty"));
          return;
        }
        setCreateOpen(true);
        if (manifest.name) setValue("name", manifest.name);
        if (manifest.type) setValue("type", manifest.type);
        if (manifest.image_name) setValue("image_name", manifest.image_name);
        if (manifest.image_tag) setValue("image_tag", manifest.image_tag);
        if (manifest.config_schema) setCreateSchema(manifest.config_schema as TemplateConfigField[]);
        toast.success(t("admin.templates.yamlImportSuccess"));
      } catch {
        toast.error(t("admin.templates.yamlImportFailed"));
      }
    };
    reader.readAsText(file);
  }

  const createMutation = useMutation({
    mutationFn: (payload: CreateForm & { config_schema: TemplateConfigField[] }) => api.post("/admin/templates", payload),
    onSuccess: () => {
      toast.success(t("admin.templates.addSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-templates"] });
      setCreateOpen(false);
      setCreateSchema([]);
      reset();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.templates.addFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/templates/${id}`),
    onSuccess: () => {
      toast.success(t("admin.templates.deleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-templates"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.templates.deleteFailed"))),
  });

  function copyId(id: string) {
    navigator.clipboard?.writeText(id);
    toast.success(t("admin.templates.copyIdSuccess"));
  }

  const columns: DataTableColumn<BotTemplate>[] = [
    {
      key: "name",
      header: t("admin.templates.colName"),
      sortValue: (row) => row.name,
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      key: "type",
      header: t("admin.templates.colType"),
      sortValue: (row) => row.type,
      cell: (row) => row.type,
    },
    {
      key: "image",
      header: t("admin.templates.colImage"),
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.image_name}:{row.image_tag}
        </span>
      ),
    },
    {
      key: "config_schema",
      header: t("admin.templates.colSettings"),
      cell: (row) => (
        <span className="text-xs text-slate-400">
          {row.config_schema && row.config_schema.length > 0
            ? t("admin.templates.fieldCount", { count: row.config_schema.length })
            : t("admin.templates.noConfigFields")}
        </span>
      ),
    },
    {
      key: "status",
      header: t("admin.templates.isActiveLabel"),
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
      key: "id",
      header: t("admin.templates.colId"),
      cell: (row) => (
        <button
          onClick={() => copyId(row.id)}
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-2 py-1 font-mono text-xs text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:hover:bg-white/10"
          dir="ltr"
        >
          <Copy className="h-3 w-3" />
          {row.id.slice(0, 8)}…
        </button>
      ),
    },
    {
      key: "actions",
      header: t("admin.servers.colActions"),
      cell: (row) => (
        <div className="flex items-center gap-1">
          <button
            onClick={() => setEditing(row)}
            aria-label={t("admin.templates.editButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Pencil className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              if (confirm(t("admin.templates.deleteConfirm", { name: row.name }))) deleteMutation.mutate(row.id);
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
          <h2 className="text-xl font-bold">{t("admin.templates.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.templates.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <input ref={yamlInputRef} type="file" accept=".yaml,.yml" className="hidden" onChange={handleYamlFile} />
          <Button variant="secondary" onClick={() => yamlInputRef.current?.click()}>
            <FileUp className="h-4 w-4" />
            {t("admin.templates.importYamlButton")}
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            {t("admin.templates.addButton")}
          </Button>
        </div>
      </div>

      {error ? (
        <ErrorState message={apiErrorMessage(error, t("admin.templates.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          emptyIcon={<LayoutTemplate className="h-8 w-8" />}
          emptyTitle={t("admin.templates.emptyTitle")}
          searchPlaceholder={t("admin.templates.searchPlaceholder")}
          searchFn={(row, q) =>
            row.name.toLowerCase().includes(q.toLowerCase()) || row.type.toLowerCase().includes(q.toLowerCase())
          }
        />
      )}

      <Modal
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          setCreateSchema([]);
        }}
        title={t("admin.templates.addTitle")}
      >
        <form
          className="space-y-4"
          onSubmit={handleSubmit((values) => createMutation.mutate({ ...values, config_schema: createSchema }))}
        >
          <Input
            label={t("admin.templates.nameLabel")}
            {...register("name", { required: t("admin.templates.nameRequired") })}
            error={errors.name?.message}
          />
          <Controller
            control={control}
            name="type"
            rules={{ required: t("admin.templates.typeRequired") }}
            render={({ field }) => {
              const isKnown = !!field.value && (serviceTypes ?? []).includes(field.value);
              const isCustomMode = !isKnown;
              return (
                <div className="space-y-2">
                  <Select
                    label={t("admin.templates.typeLabel")}
                    dir="ltr"
                    value={isKnown ? field.value : field.value ? CUSTOM_TYPE_VALUE : ""}
                    onChange={(e) => field.onChange(e.target.value === CUSTOM_TYPE_VALUE ? "" : e.target.value)}
                    error={errors.type?.message}
                  >
                    <option value="" disabled>
                      {t("admin.templates.typePlaceholder")}
                    </option>
                    {(serviceTypes ?? []).map((ty) => (
                      <option key={ty} value={ty}>
                        {ty}
                      </option>
                    ))}
                    <option value={CUSTOM_TYPE_VALUE}>{t("admin.templates.customTypeOption")}</option>
                  </Select>
                  {isCustomMode && (
                    <Input
                      dir="ltr"
                      placeholder="e.g. music-bot"
                      value={field.value}
                      onChange={(e) => field.onChange(e.target.value)}
                    />
                  )}
                </div>
              );
            }}
          />
          <Input
            label={t("admin.templates.imageNameLabel")}
            dir="ltr"
            {...register("image_name", { required: t("admin.templates.imageNameRequired") })}
            error={errors.image_name?.message}
          />
          <Input
            label={t("admin.templates.imageTagLabel")}
            dir="ltr"
            placeholder="latest"
            {...register("image_tag", { required: t("admin.templates.imageTagRequired") })}
            error={errors.image_tag?.message}
          />

          <ConfigSchemaEditor fields={createSchema} onChange={setCreateSchema} />

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setCreateOpen(false);
                setCreateSchema([]);
              }}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={isSubmitting || createMutation.isPending}>
              {t("common.add")}
            </Button>
          </div>
        </form>
      </Modal>

      <EditTemplateModal template={editing} onClose={() => setEditing(null)} />
    </div>
  );
}

/**
 * ویرایشگر عمومیِ «چه چیزی کاربرِ نهایی می‌تواند برای این نوع ربات تنظیم کند» — بازخورد کاربر
 * ۲۰۲۶-۰۷-۰۳: «برای تمپلیت‌ها چیزهایی که کاربر می‌تواند تغییر بدهد را از او بگیر و ذخیره کن».
 * لیستی از فیلدها با state ساده مدیریت می‌شود (نه react-hook-form) چون آرایه‌ی پویا با
 * تعداد نامشخص فیلد است؛ مقدارِ نهایی هنگام submit فرم بیرونی به همراه بقیه‌ی مقادیر
 * فرستاده می‌شود.
 */
function ConfigSchemaEditor({
  fields,
  onChange,
}: {
  fields: TemplateConfigField[];
  onChange: (fields: TemplateConfigField[]) => void;
}) {
  const { t } = useTranslation();

  function updateField(index: number, patch: Partial<TemplateConfigField>) {
    onChange(fields.map((f, i) => (i === index ? { ...f, ...patch } : f)));
  }
  function removeField(index: number) {
    onChange(fields.filter((_, i) => i !== index));
  }
  function addField() {
    onChange([...fields, { key: "", label: "", type: "string", required: false }]);
  }

  return (
    <div className="space-y-3 border-t border-dashed border-slate-200 pt-3 dark:border-white/10">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-slate-700 dark:text-slate-300">{t("admin.templates.configSchemaLabel")}</p>
        <button
          type="button"
          onClick={addField}
          className="flex items-center gap-1 text-xs font-medium text-violet-600 hover:underline dark:text-violet-400"
        >
          <Plus className="h-3.5 w-3.5" />
          {t("admin.templates.addFieldButton")}
        </button>
      </div>
      <p className="text-xs text-slate-400">{t("admin.templates.configSchemaHint")}</p>

      {fields.length === 0 && <p className="text-xs text-slate-400">{t("admin.templates.noConfigFields")}</p>}

      {fields.map((field, i) => (
        <div key={i} className="space-y-2 rounded-xl border border-slate-200 p-3 dark:border-white/10">
          <div className="grid grid-cols-2 gap-2">
            <Input
              label={t("admin.templates.fieldKeyLabel")}
              dir="ltr"
              placeholder="CHANNEL_ID"
              value={field.key}
              onChange={(e) => updateField(i, { key: e.target.value })}
            />
            <Input
              label={t("admin.templates.fieldLabelLabel")}
              value={field.label}
              onChange={(e) => updateField(i, { label: e.target.value })}
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <Select
              label={t("admin.templates.fieldTypeLabel")}
              value={field.type}
              onChange={(e) => updateField(i, { type: e.target.value as TemplateConfigField["type"] })}
            >
              <option value="string">{t("admin.templates.fieldTypeString")}</option>
              <option value="number">{t("admin.templates.fieldTypeNumber")}</option>
              <option value="boolean">{t("admin.templates.fieldTypeBoolean")}</option>
              <option value="select">{t("admin.templates.fieldTypeSelect")}</option>
            </Select>
            <label className="flex items-center gap-2 self-end pb-2 text-sm">
              <input
                type="checkbox"
                checked={field.required ?? false}
                onChange={(e) => updateField(i, { required: e.target.checked })}
                className="h-4 w-4 rounded border-slate-300"
              />
              {t("admin.templates.fieldRequiredLabel")}
            </label>
          </div>
          {field.type === "select" && (
            <Input
              label={t("admin.templates.fieldOptionsLabel")}
              dir="ltr"
              placeholder="option1, option2, option3"
              value={(field.options ?? []).join(", ")}
              onChange={(e) =>
                updateField(i, {
                  options: e.target.value
                    .split(",")
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
            />
          )}
          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => removeField(i)}
              className="flex items-center gap-1 text-xs font-medium text-red-500 hover:underline"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t("common.delete")}
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

function EditTemplateModal({ template, onClose }: { template: BotTemplate | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [schema, setSchema] = useState<TemplateConfigField[]>([]);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<EditForm>({
    values: template
      ? {
          name: template.name,
          image_name: template.image_name,
          image_tag: template.image_tag,
          is_active: template.is_active ?? true,
        }
      : undefined,
  });

  useEffect(() => {
    setSchema(template?.config_schema ?? []);
  }, [template?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const mutation = useMutation({
    mutationFn: (values: EditForm & { config_schema: TemplateConfigField[] }) =>
      api.patch(`/admin/templates/${template!.id}`, values),
    onSuccess: () => {
      toast.success(t("admin.templates.editSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-templates"] });
      onClose();
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("admin.templates.editFailed"))),
  });

  if (!template) return null;

  return (
    <Modal open={!!template} onClose={onClose} title={t("admin.templates.editTitle")}>
      <form
        className="space-y-4"
        onSubmit={handleSubmit((values) => mutation.mutate({ ...values, config_schema: schema }))}
      >
        <Input label={t("admin.templates.nameLabel")} {...register("name", { required: true })} error={errors.name?.message} />
        <Input label={t("admin.templates.imageNameLabel")} dir="ltr" {...register("image_name", { required: true })} />
        <Input label={t("admin.templates.imageTagLabel")} dir="ltr" {...register("image_tag", { required: true })} />
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...register("is_active")} className="h-4 w-4 rounded border-slate-300" />
          {t("admin.templates.isActiveLabel")}
        </label>

        <ConfigSchemaEditor fields={schema} onChange={setSchema} />

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
