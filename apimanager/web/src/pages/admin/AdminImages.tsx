import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, Controller } from "react-hook-form";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import axios from "axios";
import { HardDrive, Plus, Trash2, Pencil, Upload, AlertTriangle, ShieldCheck, Power } from "lucide-react";
import { api, unwrap } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input, Select } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { DataTable, type DataTableColumn } from "@/components/ui/DataTable";
import { ErrorState, EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import type { RegistryImage } from "@/lib/types";

const CUSTOM_TYPE_VALUE = "__custom__";

interface AllowedCaller {
  id?: string;
  label: string;
  cidr?: string;
  domain?: string;
  can_write?: boolean;
  is_active?: boolean;
  [key: string]: unknown;
}

/** پاسخِ image-registry شکلش دقیق مشخص نیست (پروکسیِ کور، رجوع به بک‌اند) — این‌جا چند شکلِ
 * محتمل را امتحان می‌کنیم تا هرکدام که درست بود کار کند. */
function extractList(raw: unknown): RegistryImage[] {
  if (Array.isArray(raw)) return raw as RegistryImage[];
  if (raw && typeof raw === "object") {
    const obj = raw as Record<string, unknown>;
    if (Array.isArray(obj.images)) return obj.images as RegistryImage[];
    if (Array.isArray(obj.data)) return obj.data as RegistryImage[];
    if (obj.data && typeof obj.data === "object") return extractList(obj.data);
  }
  return [];
}

function extractErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data;
    if (data && typeof data === "object") {
      const obj = data as Record<string, unknown>;
      if (typeof obj.message === "string" && obj.message) return obj.message;
      if (typeof obj.error === "string" && obj.error) return obj.error;
    }
    if (typeof data === "string" && data) return data;
    if (!err.response) return "اتصال به image-registry برقرار نشد.";
  }
  return fallback;
}

interface CreateForm {
  name: string;
  tag: string;
  service_type: string;
  description: string;
}

/**
 * صفحه‌ی جدید (بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «باید یه صفحه برای آپلود ایمیج‌ها داشته باشم»).
 *
 * به‌روزرسانی بعد از بررسیِ NEEDS.md: مسیرهای زیر با کدِ واقعیِ image-registry تطبیق داده
 * شدند و درست‌اند. مشکلِ واقعی، مدل احرازِ هویتِ متفاوتِ خودِ image-registry است: `/v1/images*`
 * بر اساسِ IP فراخوان‌کننده تصمیم می‌گیرد، نه X-Admin-Key — یعنی تا وقتی IP خروجیِ apimanager
 * به‌عنوانِ یک «Allowed Caller» با دسترسیِ نوشتن ثبت نشود، هر ثبت/آپلود/حذف این‌جا ۴۰۳
 * می‌گیرد. دکمه‌ی «IP های مجاز» پایین برای همین اضافه شد — چون `/v1/callers/*` واقعاً
 * X-Admin-Key را چک می‌کند، از همین پنل قابل مدیریت است.
 */
export default function AdminImages() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<RegistryImage | null>(null);
  const [uploadFor, setUploadFor] = useState<RegistryImage | null>(null);
  const [callersOpen, setCallersOpen] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["admin-images"],
    queryFn: async () => {
      const res = await api.get("/admin/images");
      return extractList(res.data);
    },
  });

  const { data: serviceTypes } = useQuery({
    queryKey: ["service-types"],
    queryFn: async () => unwrap<string[]>(await api.get("/service-types")),
    enabled: createOpen,
  });

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateForm>({ defaultValues: { service_type: "" } });

  const createMutation = useMutation({
    mutationFn: (values: CreateForm) => api.post("/admin/images", { ...values, is_active: true }),
    onSuccess: () => {
      toast.success(t("admin.images.addSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-images"] });
      setCreateOpen(false);
      reset();
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.addFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/images/${id}`),
    onSuccess: () => {
      toast.success(t("admin.images.deleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-images"] });
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.deleteFailed"))),
  });

  const columns: DataTableColumn<RegistryImage>[] = [
    {
      key: "name",
      header: t("admin.images.colName"),
      sortValue: (row) => row.name,
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      key: "tag",
      header: t("admin.images.colTag"),
      cell: (row) => (
        <span className="font-mono text-xs" dir="ltr">
          {row.tag}
        </span>
      ),
    },
    {
      key: "service_type",
      header: t("admin.images.colType"),
      cell: (row) => row.service_type ?? "—",
    },
    {
      key: "has_file",
      header: t("admin.images.colFile"),
      cell: (row) => (
        <span
          className={
            row.file_sha256
              ? "inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-300"
              : "inline-flex rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-400/10 dark:text-amber-300"
          }
        >
          {row.file_sha256 ? t("admin.images.fileUploaded") : t("admin.images.fileMissing")}
        </span>
      ),
    },
    {
      key: "is_active",
      header: t("admin.templates.isActiveLabel"),
      cell: (row) => (row.is_active ? t("common.yes") : t("common.no")),
    },
    {
      key: "actions",
      header: t("admin.servers.colActions"),
      cell: (row) => (
        <div className="flex items-center gap-1">
          <button
            onClick={() => setUploadFor(row)}
            aria-label={t("admin.images.uploadButton")}
            title={t("admin.images.uploadButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Upload className="h-4 w-4" />
          </button>
          <button
            onClick={() => setEditing(row)}
            aria-label={t("admin.templates.editButton")}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
          >
            <Pencil className="h-4 w-4" />
          </button>
          <button
            onClick={() => {
              if (row.id && confirm(t("admin.images.deleteConfirm", { name: row.name }))) deleteMutation.mutate(row.id);
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
          <h2 className="text-xl font-bold">{t("admin.images.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("admin.images.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={() => setCallersOpen(true)}>
            <ShieldCheck className="h-4 w-4" />
            {t("admin.images.callersButton")}
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            {t("admin.images.addButton")}
          </Button>
        </div>
      </div>

      <div className="glass-card flex items-start gap-2 p-3 text-xs text-amber-700 dark:text-amber-300">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        {t("admin.images.unverifiedWarning")}
      </div>

      {error ? (
        <ErrorState message={extractErrorMessage(error, t("admin.images.loadFailed"))} onRetry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          data={data ?? []}
          getRowId={(row) => row.id ?? `${row.name}:${row.tag}`}
          isLoading={isLoading}
          emptyIcon={<HardDrive className="h-8 w-8" />}
          emptyTitle={t("admin.images.emptyTitle")}
          searchPlaceholder={t("admin.images.searchPlaceholder")}
          searchFn={(row, q) => row.name.toLowerCase().includes(q.toLowerCase()) || row.tag.toLowerCase().includes(q.toLowerCase())}
        />
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title={t("admin.images.addTitle")}>
        <form className="space-y-4" onSubmit={handleSubmit((values) => createMutation.mutate(values))}>
          <Input
            label={t("admin.images.nameLabel")}
            dir="ltr"
            placeholder="creatorbot/uploader-bot"
            {...register("name", { required: t("admin.images.nameRequired") })}
            error={errors.name?.message}
          />
          <Input
            label={t("admin.images.tagLabel")}
            dir="ltr"
            placeholder="v2.3.1"
            {...register("tag", { required: t("admin.images.tagRequired") })}
            error={errors.tag?.message}
          />
          <Controller
            control={control}
            name="service_type"
            render={({ field }) => {
              const isKnown = !!field.value && (serviceTypes ?? []).includes(field.value);
              return (
                <div className="space-y-2">
                  <Select
                    label={t("admin.templates.typeLabel")}
                    dir="ltr"
                    value={isKnown ? field.value : field.value ? CUSTOM_TYPE_VALUE : ""}
                    onChange={(e) => field.onChange(e.target.value === CUSTOM_TYPE_VALUE ? "" : e.target.value)}
                  >
                    <option value="">{t("admin.templates.typePlaceholder")}</option>
                    {(serviceTypes ?? []).map((ty) => (
                      <option key={ty} value={ty}>
                        {ty}
                      </option>
                    ))}
                    <option value={CUSTOM_TYPE_VALUE}>{t("admin.templates.customTypeOption")}</option>
                  </Select>
                  {(!isKnown && field.value) || field.value === "" ? (
                    <Input dir="ltr" placeholder="uploader" value={field.value} onChange={(e) => field.onChange(e.target.value)} />
                  ) : null}
                </div>
              );
            }}
          />
          <Input label={t("admin.images.descriptionLabel")} {...register("description")} />
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

      <EditImageModal image={editing} onClose={() => setEditing(null)} />
      <UploadImageModal image={uploadFor} onClose={() => setUploadFor(null)} />
      <CallersModal open={callersOpen} onClose={() => setCallersOpen(false)} />
    </div>
  );
}

interface CallerForm {
  label: string;
  cidr: string;
  can_write: boolean;
}

/**
 * مدیریتِ «Allowed Caller» های image-registry — رفعِ گپِ NEEDS.md بخش ۰: بدون این، ثبتِ IP
 * خروجیِ apimanager فقط با curl دستی روی خودِ image-registry ممکن بود. `/v1/callers/*`
 * (برخلافِ `/v1/images*`) واقعاً X-Admin-Key را چک می‌کند، پس همان پروکسیِ کورِ موجود این‌جا
 * هم درست کار می‌کند.
 */
function CallersModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CallerForm>({ defaultValues: { can_write: true } });

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-image-callers"],
    queryFn: async () => {
      const res = await api.get("/admin/image-callers");
      return extractList(res.data) as unknown as AllowedCaller[];
    },
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: (values: CallerForm) => api.post("/admin/image-callers", { ...values, is_active: true }),
    onSuccess: () => {
      toast.success(t("admin.images.callerAddSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-image-callers"] });
      reset();
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.callerAddFailed"))),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      api.patch(`/admin/image-callers/${id}`, { is_active }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-image-callers"] }),
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.callerToggleFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/image-callers/${id}`),
    onSuccess: () => {
      toast.success(t("admin.images.callerDeleteSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-image-callers"] });
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.callerDeleteFailed"))),
  });

  return (
    <Modal open={open} onClose={onClose} title={t("admin.images.callersTitle")}>
      <div className="space-y-4">
        <p className="text-xs text-slate-400">{t("admin.images.callersHint")}</p>

        {isLoading && (
          <div className="space-y-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {error && <p className="text-sm text-red-600 dark:text-red-400">{extractErrorMessage(error, t("admin.images.callersLoadFailed"))}</p>}
        {!isLoading && !error && (!data || data.length === 0) && (
          <EmptyState icon={<ShieldCheck className="h-6 w-6" />} title={t("admin.images.callersEmptyTitle")} />
        )}
        {!isLoading && !error && data && data.length > 0 && (
          <ul className="max-h-56 space-y-1.5 overflow-auto">
            {data.map((caller) => (
              <li
                key={caller.id ?? caller.label}
                className="flex items-center justify-between gap-2 rounded-lg border border-slate-100 px-3 py-2 text-sm dark:border-white/10"
              >
                <div>
                  <p className="font-medium">{caller.label}</p>
                  <p className="font-mono text-xs text-slate-400" dir="ltr">
                    {caller.cidr ?? caller.domain ?? "—"} {caller.can_write ? "· write" : "· read-only"}
                  </p>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => caller.id && toggleMutation.mutate({ id: caller.id, is_active: !caller.is_active })}
                    className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10"
                    title={t("admin.images.callerToggleButton")}
                  >
                    <Power className="h-3.5 w-3.5" />
                  </button>
                  <button
                    onClick={() => caller.id && deleteMutation.mutate(caller.id)}
                    className="flex h-7 w-7 items-center justify-center rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                    title={t("common.delete")}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}

        <form
          className="space-y-3 border-t border-dashed border-slate-200 pt-3 dark:border-white/10"
          onSubmit={handleSubmit((values) => createMutation.mutate(values))}
        >
          <Input
            label={t("admin.images.callerLabelLabel")}
            placeholder="apimanager"
            {...register("label", { required: t("admin.images.callerLabelRequired") })}
            error={errors.label?.message}
          />
          <Input
            label={t("admin.images.callerCidrLabel")}
            dir="ltr"
            placeholder="172.16.0.5/32"
            {...register("cidr", { required: t("admin.images.callerCidrRequired") })}
            error={errors.cidr?.message}
          />
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" {...register("can_write")} className="h-4 w-4 rounded border-slate-300" />
            {t("admin.images.callerCanWriteLabel")}
          </label>
          <div className="flex justify-end gap-2">
            <Button type="submit" size="sm" loading={isSubmitting || createMutation.isPending}>
              <Plus className="h-3.5 w-3.5" />
              {t("admin.images.callerAddButton")}
            </Button>
          </div>
        </form>

        <div className="flex justify-end pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.close")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

interface EditForm {
  name: string;
  tag: string;
  description: string;
  is_active: boolean;
}

function EditImageModal({ image, onClose }: { image: RegistryImage | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { register, handleSubmit, formState } = useForm<EditForm>({
    values: image
      ? { name: image.name, tag: image.tag, description: image.description ?? "", is_active: image.is_active ?? true }
      : undefined,
  });

  const mutation = useMutation({
    mutationFn: (values: EditForm) => api.patch(`/admin/images/${image!.id}`, values),
    onSuccess: () => {
      toast.success(t("admin.images.editSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-images"] });
      onClose();
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.editFailed"))),
  });

  if (!image) return null;

  return (
    <Modal open={!!image} onClose={onClose} title={t("admin.images.editTitle")}>
      <form className="space-y-4" onSubmit={handleSubmit((values) => mutation.mutate(values))}>
        <Input label={t("admin.images.nameLabel")} dir="ltr" {...register("name", { required: true })} />
        <Input label={t("admin.images.tagLabel")} dir="ltr" {...register("tag", { required: true })} />
        <Input label={t("admin.images.descriptionLabel")} {...register("description")} />
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...register("is_active")} className="h-4 w-4 rounded border-slate-300" />
          {t("admin.templates.isActiveLabel")}
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

function UploadImageModal({ image, onClose }: { image: RegistryImage | null; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const form = new FormData();
      form.append("file", file!);
      return api.post(`/admin/images/${image!.id}/file`, form, {
        headers: { "Content-Type": "multipart/form-data" },
      });
    },
    onSuccess: () => {
      toast.success(t("admin.images.uploadSuccess"));
      queryClient.invalidateQueries({ queryKey: ["admin-images"] });
      setFile(null);
      onClose();
    },
    onError: (err) => toast.error(extractErrorMessage(err, t("admin.images.uploadFailed"))),
  });

  if (!image) return null;

  return (
    <Modal open={!!image} onClose={onClose} title={t("admin.images.uploadTitle", { name: image.name })}>
      <div className="space-y-4">
        <p className="text-xs text-slate-400">{t("admin.images.uploadHint")}</p>
        <input
          ref={fileInputRef}
          type="file"
          accept=".tar,.tar.gz"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="block w-full text-sm text-slate-500 file:me-3 file:rounded-lg file:border-0 file:bg-violet-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-violet-700 dark:file:bg-violet-500/15 dark:file:text-violet-300"
        />
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button type="button" disabled={!file} loading={mutation.isPending} onClick={() => mutation.mutate()}>
            <Upload className="h-4 w-4" />
            {t("admin.images.uploadSubmitButton")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
