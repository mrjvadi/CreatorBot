import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useForm } from "react-hook-form";
import toast from "react-hot-toast";
import axios from "axios";
import { CreditCard, Check, Wallet, Copy, RefreshCw } from "lucide-react";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusBadge } from "@/components/ui/Badge";
import { EmptyState, ErrorState } from "@/components/ui/EmptyState";
import { formatDate } from "@/lib/format";
import type { Me, Plan, Subscription } from "@/lib/types";

interface WalletBalance {
  telegram_id: number;
  ton_balance: number;
  credit: number;
  total: number;
  frozen: number;
  ton_address: string;
}

interface TopupInvoice {
  code: string;
  master_address: string;
  amount_ton: number;
  expires_at: number;
}

/**
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۵ («پرداختی‌ها کامل نشده»): تا این‌جا صفحه‌ی پلن‌ها فقط نمایشی بود —
 * هیچ دکمه‌ی خریدی نبود، یعنی جدول Payment همیشه خالی می‌ماند. natspayclient (که خودِ apimanager
 * از قبل برایِ اعتبار دستیِ ادمین استفاده می‌کرد) کامل پیاده‌سازی شده بود — DeductForService/
 * CreateInvoice/InvoiceStatus/Balance — فقط هیچ endpoint کاربرمحوری صدایشان نمی‌زد. حالا خریدِ
 * واقعی + شارژِ کیف‌پول (واریز مستقیم TON با کد comment، طبق معماری واقعیِ botpay) وصل شده.
 */
export default function Plans() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [topupOpen, setTopupOpen] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["plans"],
    queryFn: async () => unwrap<Plan[]>(await api.get("/plans")),
  });

  const { data: me } = useQuery({
    queryKey: ["me"],
    queryFn: async () => unwrap<Me>(await api.get("/me")),
  });
  const subscription = me?.subscription as Subscription | null | undefined;

  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ["wallet-balance"],
    queryFn: async () => unwrap<WalletBalance>(await api.get("/wallet/balance")),
    retry: false,
  });

  const buyMutation = useMutation({
    mutationFn: (planId: string) => api.post(`/plans/${planId}/buy`),
    onSuccess: () => {
      toast.success(t("plans.buySuccess"));
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["wallet-balance"] });
    },
    onError: (err) => {
      if (axios.isAxiosError(err) && err.response?.status === 402) {
        toast.error(t("plans.buyInsufficientBalance"));
        setTopupOpen(true);
        return;
      }
      toast.error(apiErrorMessage(err, t("plans.buyFailed")));
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold">{t("plans.title")}</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t("plans.subtitle")}</p>
        </div>

        <Card className="flex items-center gap-3 py-3">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-50 text-violet-600 dark:bg-violet-500/15 dark:text-violet-300">
            <Wallet className="h-4.5 w-4.5" />
          </span>
          <div>
            <p className="text-xs text-slate-400">{t("plans.walletBalanceLabel")}</p>
            {walletLoading ? (
              <Skeleton className="mt-1 h-4 w-16" />
            ) : (
              <p className="font-bold tabular-nums" dir="ltr">
                {(wallet?.total ?? 0).toLocaleString()} TON
              </p>
            )}
          </div>
          <Button size="sm" variant="secondary" onClick={() => setTopupOpen(true)}>
            {t("plans.topupButton")}
          </Button>
        </Card>
      </div>

      {isLoading && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      )}

      {error && <ErrorState message={apiErrorMessage(error, t("plans.loadFailed"))} onRetry={refetch} />}

      {!isLoading && !error && (!data || data.length === 0) && (
        <EmptyState icon={<CreditCard className="h-8 w-8" />} title={t("plans.emptyTitle")} />
      )}

      {!isLoading && !error && data && data.length > 0 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((plan) => {
            const isCurrent = subscription?.is_active && subscription.plan_id === plan.id;
            return (
              <Card key={plan.id} className="flex flex-col gap-4 transition-shadow hover:shadow-popover">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-50 text-violet-600 dark:bg-violet-500/15 dark:text-violet-300">
                    <CreditCard className="h-5 w-5" />
                  </div>
                  <div>
                    <h3 className="font-semibold">{plan.name ?? t("plans.unnamed", { id: plan.id })}</h3>
                    <p className="text-lg font-bold text-violet-600 dark:text-violet-400" dir="ltr">
                      {plan.is_free ? t("plans.freeLabel") : `${plan.price} TON`}
                    </p>
                  </div>
                </div>

                <dl className="space-y-2 border-t border-slate-100 pt-3 text-sm text-slate-600 dark:border-white/10 dark:text-slate-300">
                  <div className="flex items-start justify-between gap-2">
                    <span className="flex items-center gap-1.5 text-slate-400">
                      <Check className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                      {t("plans.durationLabel")}
                    </span>
                    <span className="font-medium">
                      {plan.duration_day > 0 ? t("plans.durationDays", { count: plan.duration_day }) : t("plans.durationForever")}
                    </span>
                  </div>
                  <div className="flex items-start justify-between gap-2">
                    <span className="flex items-center gap-1.5 text-slate-400">
                      <Check className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                      {t("plans.maxBotsLabel")}
                    </span>
                    <span className="font-medium">{plan.max_bots}</span>
                  </div>
                </dl>

                {isCurrent ? (
                  <span className="mt-auto inline-flex items-center justify-center gap-1.5 rounded-lg bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-300">
                    <Check className="h-4 w-4" />
                    {t("plans.currentPlanLabel")}
                  </span>
                ) : (
                  <Button
                    className="mt-auto w-full justify-center"
                    loading={buyMutation.isPending && buyMutation.variables === plan.id}
                    onClick={() => buyMutation.mutate(plan.id)}
                  >
                    {plan.is_free ? t("plans.activateFreeButton") : t("plans.buyButton")}
                  </Button>
                )}
              </Card>
            );
          })}
        </div>
      )}

      <TopupModal open={topupOpen} onClose={() => setTopupOpen(false)} />
    </div>
  );
}

function TopupModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const [invoice, setInvoice] = useState<TopupInvoice | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<{ amount_ton: number }>();

  const createMutation = useMutation({
    mutationFn: (amountTON: number) => api.post("/wallet/topup", { amount_ton: amountTON }),
    onSuccess: (res) => setInvoice(unwrap<TopupInvoice>(res)),
    onError: (err) => toast.error(apiErrorMessage(err, t("plans.topupFailed"))),
  });

  const statusQuery = useQuery({
    queryKey: ["topup-status", invoice?.code],
    queryFn: async () =>
      unwrap<{ status: string; amount_ton: number; paid_ton: number }>(
        await api.get(`/wallet/topup/${invoice!.code}/status`)
      ),
    enabled: false,
  });

  function copy(text: string) {
    navigator.clipboard?.writeText(text);
    toast.success(t("common.copied"));
  }

  function handleClose() {
    setInvoice(null);
    reset();
    onClose();
  }

  return (
    <Modal open={open} onClose={handleClose} title={t("plans.topupTitle")}>
      {!invoice ? (
        <form
          className="space-y-4"
          onSubmit={handleSubmit((values) => createMutation.mutate(Number(values.amount_ton)))}
        >
          <Input
            label={t("plans.topupAmountLabel")}
            type="number"
            step="any"
            dir="ltr"
            {...register("amount_ton", { required: true, valueAsNumber: true, min: 0.01 })}
            error={errors.amount_ton?.message}
          />
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={handleClose}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={isSubmitting || createMutation.isPending}>
              {t("plans.topupCreateButton")}
            </Button>
          </div>
        </form>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300">{t("plans.topupInstructions")}</p>

          <div className="space-y-2 rounded-xl border border-slate-100 p-3 text-sm dark:border-white/10">
            <div className="flex items-center justify-between gap-2">
              <span className="text-slate-400">{t("plans.topupAddressLabel")}</span>
              <button
                onClick={() => copy(invoice.master_address)}
                className="flex items-center gap-1 font-mono text-xs"
                dir="ltr"
              >
                <Copy className="h-3 w-3" />
                {invoice.master_address}
              </button>
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-slate-400">{t("plans.topupCodeLabel")}</span>
              <button onClick={() => copy(invoice.code)} className="flex items-center gap-1 font-mono text-xs" dir="ltr">
                <Copy className="h-3 w-3" />
                {invoice.code}
              </button>
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-slate-400">{t("plans.topupAmountLabel")}</span>
              <span className="font-medium tabular-nums" dir="ltr">
                {invoice.amount_ton} TON
              </span>
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-slate-400">{t("plans.topupExpiresLabel")}</span>
              <span className="font-medium">{formatDate(new Date(invoice.expires_at * 1000).toISOString())}</span>
            </div>
          </div>

          <div className="flex items-center justify-between gap-2">
            <Button type="button" variant="secondary" size="sm" loading={statusQuery.isFetching} onClick={() => statusQuery.refetch()}>
              <RefreshCw className="h-3.5 w-3.5" />
              {t("plans.topupCheckStatusButton")}
            </Button>
            {statusQuery.data && <StatusBadge status={statusQuery.data.status} size="sm" />}
          </div>

          <div className="flex justify-end pt-2">
            <Button type="button" variant="secondary" onClick={handleClose}>
              {t("common.close")}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}
