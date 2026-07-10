import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { useAuthStore } from "./auth-store";
import { logRequest } from "./request-log-store";

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

// در dev سرور Vite را (نه apimanager را) مستقیم صدا می‌زنیم: مسیر نسبی /api/v1 از همین
// مبدأ (localhost:5173) شروع می‌شود و vite.config.ts آن را پشت‌صحنه به apimanager پروکسی
// می‌کند — چون apimanager هیچ CORS middleware ای ندارد و فراخوانی مستقیمِ یک پورت دیگر از
// مرورگر بلاک می‌شود. در build نهایی (production) از آدرس واقعی apimanager استفاده می‌شود؛
// آن‌جا یا باید فرانت و apimanager هم‌مبدأ سرو شوند، یا CORS به apimanager اضافه شود.
const API_PREFIX = import.meta.env.DEV ? "/api/v1" : `${BASE_URL}/api/v1`;

declare module "axios" {
  interface InternalAxiosRequestConfig {
    _startTime?: number;
  }
}

export const api = axios.create({
  baseURL: API_PREFIX,
  timeout: 15_000,
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  config._startTime = performance.now();
  return config;
});

/** برای صفحه‌ی «لاگ درخواست‌ها» — فقط داخل همین مرورگر، هیچ‌جا فرستاده نمی‌شود. */
function recordFromResponse(res: {
  config: InternalAxiosRequestConfig;
  status: number;
  data?: unknown;
}) {
  const start = res.config._startTime ?? performance.now();
  logRequest({
    method: (res.config.method ?? "get").toUpperCase(),
    url: res.config.url ?? "",
    status: res.status,
    ok: res.status >= 200 && res.status < 400,
    durationMs: Math.round(performance.now() - start),
    startedAt: Date.now(),
    requestBody: res.config.data,
    responseBody: res.data,
  });
}

function recordFromError(error: AxiosError) {
  const config = error.config as InternalAxiosRequestConfig | undefined;
  const start = config?._startTime ?? performance.now();
  logRequest({
    method: (config?.method ?? "get").toUpperCase(),
    url: config?.url ?? "",
    status: error.response?.status ?? null,
    ok: false,
    durationMs: Math.round(performance.now() - start),
    startedAt: Date.now(),
    requestBody: config?.data,
    responseBody: error.response?.data,
    errorMessage: error.message,
  });
}

let refreshPromise: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = useAuthStore.getState().refreshToken;
  if (!refreshToken) return null;
  try {
    const res = await axios.post(`${API_PREFIX}/auth/refresh`, {
      refresh_token: refreshToken,
    });
    const newToken: string | undefined = res.data?.data?.access_token;
    if (!newToken) return null;
    useAuthStore.getState().setAccessToken(newToken);
    return newToken;
  } catch {
    return null;
  }
}

api.interceptors.response.use(
  (res) => {
    recordFromResponse(res);
    return res;
  },
  async (error: AxiosError) => {
    recordFromError(error);
    const original = error.config as (InternalAxiosRequestConfig & { _retry?: boolean }) | undefined;

    if (error.response?.status === 401 && original && !original._retry) {
      original._retry = true;
      if (!refreshPromise) {
        refreshPromise = refreshAccessToken().finally(() => {
          refreshPromise = null;
        });
      }
      const newToken = await refreshPromise;
      if (newToken) {
        original.headers = original.headers ?? {};
        (original.headers as Record<string, string>).Authorization = `Bearer ${newToken}`;
        original._startTime = performance.now();
        return api(original);
      }
      useAuthStore.getState().logout();
    }
    return Promise.reject(error);
  }
);

export interface ApiEnvelope<T> {
  ok: boolean;
  data?: T;
  message?: string;
}

/** استخراج پیام خطای خوانا از پاسخ apimanager (که همیشه {ok:false, message} برمی‌گرداند). */
export function apiErrorMessage(err: unknown, fallback = "خطایی رخ داد. دوباره تلاش کنید."): string {
  if (axios.isAxiosError(err)) {
    const msg = (err.response?.data as ApiEnvelope<unknown> | undefined)?.message;
    if (msg) return msg;
    if (err.code === "ECONNABORTED") return "درخواست بیش از حد طول کشید.";
    if (!err.response) return "اتصال به سرور برقرار نشد.";
  }
  return fallback;
}

/** استخراج data از پاسخ استاندارد apimanager: {ok, data}. */
export function unwrap<T>(res: { data: ApiEnvelope<T> }): T {
  return res.data.data as T;
}
