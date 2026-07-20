import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { Trans, useTranslation } from "react-i18next";
import {
  Bot,
  ShieldCheck,
  Loader2,
  TriangleAlert,
  FlaskConical,
  ChevronDown,
  ServerCog,
  Radio,
} from "lucide-react";
import toast from "react-hot-toast";
import { api, apiErrorMessage, unwrap } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";
import { signTelegramAuth } from "@/lib/telegram-sign";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import type { TelegramAuthPayload } from "@/lib/types";

const BOT_USERNAME = import.meta.env.VITE_TELEGRAM_BOT_USERNAME as string | undefined;

const DEV_LOGIN_ENABLED =
  import.meta.env.DEV || import.meta.env.VITE_ENABLE_DEV_LOGIN === "true";

declare global {
  interface Window {
    onTelegramAuth?: (user: TelegramAuthPayload) => void;
  }
}

interface LoginResult {
  access_token: string;
  refresh_token: string;
  user_id: string;
  role: string;
}

export default function Login() {
  const { t } = useTranslation();
  const widgetRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.setSession);
  const [loading, setLoading] = useState(false);

  async function completeLogin(payload: TelegramAuthPayload) {
    setLoading(true);
    try {
      const res = await api.post("/auth/telegram", payload);
      const data = unwrap<LoginResult>(res);
      setSession({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        user: { id: data.user_id, role: data.role },
      });
      toast.success(t("auth.welcomeToast"));
      navigate("/app", { replace: true });
    } catch (err) {
      toast.error(apiErrorMessage(err, t("auth.loginFailedToast")));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!BOT_USERNAME || !widgetRef.current) return;

    window.onTelegramAuth = (payload) => {
      void completeLogin(payload);
    };

    const script = document.createElement("script");
    script.src = "https://telegram.org/js/telegram-widget.js?22";
    script.async = true;
    script.setAttribute("data-telegram-login", BOT_USERNAME);
    script.setAttribute("data-size", "large");
    script.setAttribute("data-radius", "12");
    script.setAttribute("data-onauth", "onTelegramAuth(user)");
    script.setAttribute("data-request-access", "write");
    widgetRef.current.innerHTML = "";
    widgetRef.current.appendChild(script);

    return () => {
      window.onTelegramAuth = undefined;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const features = [
    { icon: Bot, key: "auth.featureBotHosting" },
    { icon: ShieldCheck, key: "auth.featureSecureAuth" },
    { icon: Radio, key: "auth.featureRealtime" },
  ];

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* پنل برندینگ */}
      <div className="relative hidden overflow-hidden bg-brand-gradient p-10 text-white lg:flex lg:flex-col lg:justify-between">
        <div
          className="pointer-events-none absolute inset-0 opacity-20"
          style={{
            backgroundImage:
              "radial-gradient(circle at 20% 20%, white 1px, transparent 1px), radial-gradient(circle at 60% 70%, white 1px, transparent 1px)",
            backgroundSize: "48px 48px",
          }}
        />
        <div className="relative flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-white/15 backdrop-blur">
            <Bot className="h-5 w-5" />
          </div>
          <span className="text-lg font-bold">CreatorBot</span>
        </div>

        <div className="relative space-y-6">
          <h2 className="max-w-sm text-2xl font-bold leading-relaxed">{t("auth.brandTagline")}</h2>
          <ul className="space-y-3.5">
            {features.map(({ icon: Icon, key }) => (
              <li key={key} className="flex items-center gap-3 text-sm text-white/90">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white/10">
                  <Icon className="h-4 w-4" />
                </span>
                {t(key)}
              </li>
            ))}
          </ul>
        </div>

        <div className="relative flex items-center gap-2 text-xs text-white/60">
          <ServerCog className="h-3.5 w-3.5" />
          apimanager · CreatorBotV3
        </div>
      </div>

      {/* کارت لاگین */}
      <div className="flex flex-col bg-slate-50 dark:bg-transparent">
        <div className="flex justify-end p-4">
          <LanguageSwitcher />
        </div>
        <div className="flex flex-1 items-center justify-center p-4">
          <div className="glass-card w-full max-w-sm p-8 text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-gradient text-white shadow-glow lg:hidden">
              <Bot className="h-6 w-6" />
            </div>
            <h1 className="text-lg font-bold">{t("auth.title")}</h1>
            <p className="mt-1.5 text-sm text-slate-500 dark:text-slate-400">{t("auth.subtitle")}</p>

            <div className="mt-6 flex min-h-[52px] items-center justify-center">
              {loading ? (
                <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
              ) : BOT_USERNAME ? (
                <div ref={widgetRef} />
              ) : (
                <div className="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-start text-xs text-amber-800 dark:bg-amber-900/20 dark:text-amber-400">
                  <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <span>
                    <Trans
                      i18nKey="auth.missingUsernameWarning"
                      components={{
                        code: (
                          <code className="rounded bg-amber-100 px-1 dark:bg-amber-900/40" />
                        ),
                      }}
                    />
                  </span>
                </div>
              )}
            </div>

            <div className="mt-6 flex items-center justify-center gap-1.5 text-xs text-slate-400">
              <ShieldCheck className="h-3.5 w-3.5" />
              {t("auth.secureLogin")}
            </div>

            {DEV_LOGIN_ENABLED && <DevLogin loading={loading} onSubmitPayload={completeLogin} />}
          </div>
        </div>
      </div>
    </div>
  );
}

interface DevLoginForm {
  bot_token: string;
  telegram_id: string;
  first_name: string;
  last_name?: string;
  username?: string;
}

function DevLogin({
  loading,
  onSubmitPayload,
}: {
  loading: boolean;
  onSubmitPayload: (payload: TelegramAuthPayload) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<DevLoginForm>();

  async function onSubmit(values: DevLoginForm) {
    try {
      const authDate = Math.floor(Date.now() / 1000);
      const telegramID = Number(values.telegram_id.trim());
      const fields = {
        id: telegramID,
        first_name: values.first_name,
        last_name: values.last_name,
        username: values.username,
        auth_date: authDate,
      };
      const hash = await signTelegramAuth(values.bot_token.trim(), fields);
      await onSubmitPayload({
        id: telegramID,
        first_name: values.first_name,
        last_name: values.last_name,
        username: values.username,
        auth_date: authDate,
        hash,
      });
    } catch {
      toast.error(t("auth.signFailedToast"));
    }
  }

  return (
    <div className="mt-6 border-t border-dashed border-slate-200 pt-4 text-start dark:border-slate-800">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
      >
        <span className="flex items-center gap-1.5">
          <FlaskConical className="h-3.5 w-3.5" />
          {t("auth.devLoginToggle")}
        </span>
        <ChevronDown className={`h-4 w-4 transition-transform ${open ? "rotate-180" : ""}`} />
      </button>

      {open && (
        <form className="mt-3 space-y-3" onSubmit={handleSubmit(onSubmit)}>
          <p className="rounded-lg bg-slate-50 p-2.5 text-xs leading-5 text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            {t("auth.devLoginHint")}
          </p>
          <Input
            label={t("auth.botTokenLabel")}
            type="password"
            dir="ltr"
            placeholder="123456:ABC-..."
            {...register("bot_token", { required: t("auth.botTokenRequired") })}
            error={errors.bot_token?.message}
          />
          <Input
            label={t("auth.telegramIdLabel")}
            dir="ltr"
            inputMode="numeric"
            placeholder="123456789"
            {...register("telegram_id", { required: t("auth.telegramIdRequired") })}
            error={errors.telegram_id?.message}
          />
          <Input
            label={t("auth.firstNameLabel")}
            {...register("first_name", { required: t("auth.firstNameRequired") })}
            error={errors.first_name?.message}
          />
          <Input label={t("auth.lastNameLabel")} {...register("last_name")} />
          <Input label={t("auth.usernameLabel")} dir="ltr" {...register("username")} />
          <Button
            type="submit"
            variant="secondary"
            className="w-full justify-center"
            loading={isSubmitting || loading}
          >
            {t("auth.devLoginSubmit")}
          </Button>
        </form>
      )}
    </div>
  );
}
