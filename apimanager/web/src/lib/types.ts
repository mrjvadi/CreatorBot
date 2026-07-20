/**
 * این تایپ‌ها حالا مستقیماً از خودِ shared-core/models/models.go می‌آیند (۲۰۲۶-۰۷-۰۳ به‌روز شد
 * بعد از این‌که آن پوشه به این workspace وصل شد). قبلاً هیچ json tag ای روی این structها
 * نبود — یعنی apimanager بدون تغییر، این فیلدها را PascalCase (نام فیلد Go) برمی‌گرداند، نه
 * snake_case؛ در همان جلسه json tag های snake_case به models.go اضافه شد تا با همین تایپ‌ها
 * (و با الگوی دستیِ بقیه‌ی پاسخ‌های apimanager) هم‌خوان باشد.
 */

export interface User {
  id: string;
  telegram_id: number;
  username?: string;
  first_name?: string;
  role: "user" | "admin" | "owner" | string;
  balance?: number;
  is_blocked?: boolean;
  created_at?: string;
  [key: string]: unknown;
}

export interface Subscription {
  id: string;
  user_id: string;
  plan_id: string;
  started_at: string;
  expires_at?: string | null;
  is_active: boolean;
  bot_count: number;
}

export interface Me {
  user: User;
  subscription?: Subscription | null;
  bot_count: number;
}

export type InstanceStatus = "pending" | "running" | "stopped" | "error" | string;

export interface BotInstance {
  id: string;
  owner_id?: string;
  template_id?: string;
  server_id?: string;
  bot_id?: number;
  container_name?: string;
  status: InstanceStatus;
  created_at?: string;
  [key: string]: unknown;
}

export interface ServerContainerStatus {
  name: string;
  image: string;
  state: string; // running | exited | paused
  status: string; // "Up 2 hours"
}

export interface Server {
  id: string;
  name: string;
  ip: string;
  channel?: string;
  is_online?: boolean;
  last_seen?: string;
  online_since?: string | null;
  // ثانیه‌هایی که سرور پشت‌سرهم آنلاین بوده — فقط وقتی is_online=true مقدار دارد.
  online_seconds?: number | null;
  // این سه فیلد null هستند تا وقتی agentmanager آن‌ها را در heartbeat گزارش کند — یعنی
  // «گزارش نشده»، نه صفر واقعی. رجوع به توضیح در apimanager/shared-core برای جزئیات.
  cpu_percent?: number | null;
  memory_used_mb?: number | null;
  memory_total_mb?: number | null;
  containers?: ServerContainerStatus[];
  // برچسبی مثل "free" — instance های ساخته‌شده از قالب رایگان ترجیحاً فقط روی سرورهایی با
  // این تگ دیپلوی می‌شوند (رجوع به apimanager: SelectLeastLoadedServer).
  tags?: string[];
  // سقف تعداد container مجاز روی این سرور (۰ = نامحدود).
  max_containers?: number;
  [key: string]: unknown;
}

export type TemplateFieldType = "string" | "number" | "boolean" | "select";

export interface TemplateConfigField {
  key: string;
  label: string;
  type: TemplateFieldType;
  required?: boolean;
  default?: string;
  options?: string[];
}

export interface BotTemplate {
  id: string;
  name: string;
  type: string;
  image_name: string;
  image_tag: string;
  is_active?: boolean;
  is_free?: boolean;
  config_schema?: TemplateConfigField[];
  [key: string]: unknown;
}

export interface PlanBotLimit {
  id: string;
  plan_id: string;
  bot_type: string;
  max_bots: number;
}

export interface Plan {
  id: string;
  name: string;
  duration_day: number;
  price: number;
  max_bots: number;
  is_free: boolean;
  is_active: boolean;
  limits?: PlanBotLimit[] | null;
  [key: string]: unknown;
}

// مقادیر واقعیِ models.PaymentStatus در Go (بررسی شد ۲۰۲۶-۰۷-۰۵): فقط همین سه‌تا — قبلاً
// این‌جا اشتباهاً "confirmed"/"expired" حدس زده شده بود که هیچ‌کدام واقعی نیستند.
export type PaymentStatus = "pending" | "done" | "failed" | string;

export interface Payment {
  id: string;
  user_id: string;
  plan_id?: string | null;
  amount: number;
  currency: string;
  status: PaymentStatus;
  tx_hash?: string;
  from_wallet?: string;
  payment_url?: string;
  invoice_id?: string;
  confirmed_at?: string | null;
  created_at?: string;
  [key: string]: unknown;
}

export interface PromoCode {
  id: string;
  code: string;
  amount_ton: number;
  max_uses: number;
  used_count: number;
  expires_at?: string | null;
  is_active: boolean;
  created_at?: string;
  [key: string]: unknown;
}

// این تایپ حالا با store.RegisteredImage واقعیِ image-registry هم‌خوان شده (۲۰۲۶-۰۷-۰۵،
// بعد از اینکه مشخص شد آن struct اصلاً json tag نداشت و همه‌چیز PascalCase برمی‌گشت —
// رجوع image-registry/internal/store/models.go و apimanager/internal/handler/image_registry.go).
// «فایل دارد یا نه» را از روی وجودِ file_sha256 تشخیص بده، نه یک فیلدِ has_file جدا
// (چون خودِ سرویس چنین فیلدی روی این endpoint برنمی‌گرداند — فقط GET /v1/check که
// apimanager مستقیم proxy نمی‌کند این را دارد).
export interface RegistryImage {
  id?: string;
  name: string;
  tag: string;
  service_type?: string;
  description?: string;
  is_active?: boolean;
  file_sha256?: string;
  file_size?: number;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

// ── uploader-bot content (پروکسیِ NATS، رجوع apimanager/internal/handler/uploader_proxy.go) ──

export interface UploaderFolder {
  id: string;
  name: string;
  parent_id: string;
  icon?: string;
  sort_order: number;
  is_active: boolean;
}

export interface UploaderCode {
  id: string;
  code: string;
  type: "once" | "limited" | "unlimited" | "expiry" | string;
  folder_id: string;
  caption?: string;
  is_album: boolean;
  max_use: number;
  used_count: number;
  expires_at?: string | null;
  created_at?: string;
  [key: string]: unknown;
}

export interface AdminStats {
  instances: {
    total: number;
    running: number;
    stopped: number;
    pending: number;
    error: number;
  };
  servers: { total: number; online: number };
  users: number;
}

export interface TelegramAuthPayload {
  id: number;
  telegram_id?: number;
  first_name?: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
}
