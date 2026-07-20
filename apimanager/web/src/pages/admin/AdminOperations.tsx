import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { Activity, Radio, Satellite, Trash2, Power } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Input, Select } from "@/components/ui/Input";
import { formatDate } from "@/lib/format";
import type { Plan } from "@/lib/types";

interface SourceWorker {
  id: string;
  label: string;
  worker_id: string;
  app_id: number;
  phone: string;
  is_active: boolean;
  is_online: boolean;
  last_status?: string;
  last_heartbeat_at?: string;
  license_key?: string;
}

interface AuditLog {
  ID: number;
  CreatedAt: string;
  ActorRole: string;
  Action: string;
  TargetType: string;
  TargetID: string;
  Description: string;
  IPAddress?: string;
}

export default function AdminOperations() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [message, setMessage] = useState("");
  const [filter, setFilter] = useState("all");
  const [planId, setPlanId] = useState("");
  const [createdLicense, setCreatedLicense] = useState("");
  const [worker, setWorker] = useState({ label: "", app_id: "", app_hash: "", phone: "" });

  const workers = useQuery({
    queryKey: ["admin-source-workers"],
    queryFn: async () => unwrap<SourceWorker[]>(await api.get("/admin/source-workers")),
  });
  const audits = useQuery({
    queryKey: ["admin-audit-logs"],
    queryFn: async () => unwrap<AuditLog[]>(await api.get("/admin/audit-logs?limit=100")),
  });
  const plans = useQuery({
    queryKey: ["admin-plans"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/admin/plans")),
  });

  const broadcast = useMutation({
    mutationFn: () => api.post("/admin/broadcasts", { message, filter, plan_id: filter === "plan" ? planId : "" }),
    onSuccess: (res) => {
      const result = unwrap<{ queued: number }>(res);
      toast.success(t("operations.broadcastQueued", { count: result.queued }));
      setMessage("");
      queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("operations.broadcastFailed"))),
  });

  const createWorker = useMutation({
    mutationFn: () => api.post("/admin/source-workers", { ...worker, app_id: Number(worker.app_id) }),
    onSuccess: (res) => {
      const result = unwrap<SourceWorker>(res);
      setCreatedLicense(result.license_key ?? "");
      setWorker({ label: "", app_id: "", app_hash: "", phone: "" });
      queryClient.invalidateQueries({ queryKey: ["admin-source-workers"] });
      queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
      toast.success(t("operations.workerCreated"));
    },
    onError: (err) => toast.error(apiErrorMessage(err, t("operations.workerCreateFailed"))),
  });

  const toggleWorker = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) => api.patch(`/admin/source-workers/${id}`, { is_active }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-source-workers"] }),
    onError: (err) => toast.error(apiErrorMessage(err, t("operations.workerUpdateFailed"))),
  });
  const deleteWorker = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/source-workers/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-source-workers"] }),
    onError: (err) => toast.error(apiErrorMessage(err, t("operations.workerDeleteFailed"))),
  });

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">{t("operations.title")}</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("operations.subtitle")}</p>
      </div>

      <Card className="space-y-4">
        <div className="flex items-center gap-2">
          <Radio className="h-5 w-5 text-violet-500" />
          <h3 className="font-semibold">{t("operations.broadcastTitle")}</h3>
        </div>
        <textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          maxLength={4096}
          rows={5}
          placeholder={t("operations.broadcastPlaceholder")}
          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20 dark:border-white/10 dark:bg-white/5"
        />
        <div className="grid gap-3 sm:grid-cols-2">
          <Select label={t("operations.audience")} value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">{t("operations.audienceAll")}</option>
            <option value="no_plan">{t("operations.audienceNoPlan")}</option>
            <option value="plan">{t("operations.audiencePlan")}</option>
          </Select>
          {filter === "plan" && (
            <Select label={t("operations.plan")} value={planId} onChange={(e) => setPlanId(e.target.value)}>
              <option value="">{t("operations.choosePlan")}</option>
              {(plans.data ?? []).map((plan) => <option key={plan.id} value={plan.id}>{plan.name}</option>)}
            </Select>
          )}
        </div>
        <Button disabled={!message.trim() || (filter === "plan" && !planId)} loading={broadcast.isPending} onClick={() => broadcast.mutate()}>
          {t("operations.queueBroadcast")}
        </Button>
      </Card>

      <Card className="space-y-4">
        <div className="flex items-center gap-2">
          <Satellite className="h-5 w-5 text-violet-500" />
          <h3 className="font-semibold">{t("operations.workersTitle")}</h3>
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <Input label={t("operations.workerLabel")} value={worker.label} onChange={(e) => setWorker((v) => ({ ...v, label: e.target.value }))} />
          <Input label="App ID" type="number" dir="ltr" value={worker.app_id} onChange={(e) => setWorker((v) => ({ ...v, app_id: e.target.value }))} />
          <Input label="App Hash" dir="ltr" type="password" value={worker.app_hash} onChange={(e) => setWorker((v) => ({ ...v, app_hash: e.target.value }))} />
          <Input label={t("operations.phone")} dir="ltr" value={worker.phone} onChange={(e) => setWorker((v) => ({ ...v, phone: e.target.value }))} />
        </div>
        <Button disabled={!worker.app_id || !worker.app_hash || !worker.phone} loading={createWorker.isPending} onClick={() => createWorker.mutate()}>
          {t("operations.createWorker")}
        </Button>
        {createdLicense && (
          <div className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
            <p className="font-medium">{t("operations.licenseOnce")}</p>
            <code className="mt-1 block break-all" dir="ltr">{createdLicense}</code>
          </div>
        )}
        <div className="divide-y divide-slate-100 dark:divide-white/10">
          {(workers.data ?? []).map((item) => (
            <div key={item.id} className="flex flex-wrap items-center justify-between gap-3 py-3">
              <div>
                <p className="font-medium">{item.label || item.worker_id}</p>
                <p className="text-xs text-slate-400" dir="ltr">{item.phone} · {item.worker_id}</p>
                <p className="text-xs text-slate-400">{item.is_online ? t("status.online") : t("status.offline")} · {item.last_status || "—"}</p>
              </div>
              <div className="flex gap-1">
                <button className="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10" title={t("operations.toggleWorker")} onClick={() => toggleWorker.mutate({ id: item.id, is_active: !item.is_active })}>
                  <Power className={item.is_active ? "h-4 w-4 text-emerald-500" : "h-4 w-4 text-slate-400"} />
                </button>
                <button className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20" title={t("common.delete")} onClick={() => confirm(t("operations.deleteWorkerConfirm")) && deleteWorker.mutate(item.id)}>
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
          {!workers.isLoading && (workers.data?.length ?? 0) === 0 && <p className="py-4 text-sm text-slate-400">{t("operations.noWorkers")}</p>}
        </div>
      </Card>

      <Card className="space-y-4">
        <div className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-violet-500" />
          <h3 className="font-semibold">{t("operations.auditTitle")}</h3>
        </div>
        <div className="max-h-[32rem] divide-y divide-slate-100 overflow-auto dark:divide-white/10">
          {(audits.data ?? []).map((log) => (
            <div key={log.ID} className="grid gap-1 py-3 text-sm sm:grid-cols-[10rem_10rem_1fr]">
              <span className="text-xs text-slate-400">{formatDate(log.CreatedAt)}</span>
              <code className="text-xs" dir="ltr">{log.Action}</code>
              <span>{log.Description || `${log.TargetType}:${log.TargetID}`}</span>
            </div>
          ))}
          {!audits.isLoading && (audits.data?.length ?? 0) === 0 && <p className="py-4 text-sm text-slate-400">{t("operations.noAudit")}</p>}
        </div>
      </Card>
    </div>
  );
}
