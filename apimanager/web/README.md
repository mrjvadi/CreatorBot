# apimanager-web

فرانت‌اند وب برای سرویس `apimanager` — دو بخش دارد:

- **پنل کاربری** (/app): داشبورد، مدیریت instanceهای ربات شامل ساخت، کنترل، لاگ، تنظیمات و محتوا، به‌همراه پلن‌ها و پرداخت‌ها.
- **پنل مدیریت** (/admin): آمار و مدیریت instanceها، سرورها، قالب‌ها، imageها، پلن‌ها، پرداخت‌ها، کدهای تخفیف، کاربران و لاگ درخواست مرورگر. فقط برای نقش‌های admin/owner.

ورود با **Telegram Login Widget** انجام می‌شود و مستقیم به endpoint های موجود در
`internal/handler/handler.go` وصل می‌شود (بدون هیچ تغییری در بک‌اند).

### ورود بدون ثبت دامنه در BotFather (`/setdomain`)

ویجت رسمی تلگرام فقط روی دامنه‌ای کار می‌کند که با دستور `/setdomain` در BotFather برای همان ربات ثبت
شده باشد — یعنی روی `localhost` یا هر دامنه‌ی تستی دیگر اصلاً رندر نمی‌شود. برای این‌که مجبور نباشید هر
بار قبل از تست، دامنه را در BotFather عوض کنید، صفحه‌ی لاگین یک بخش «ورود آزمایشی» هم دارد (در پایین
صفحه، جمع‌شونده) که همان امضای HMAC مورد نیاز `POST /api/v1/auth/telegram` را **در خودِ مرورگر** با
Web Crypto API می‌سازد؛ فقط کافی است توکن ربات را همان‌جا وارد کنید. توکن هیچ‌وقت به هیچ سروری فرستاده
نمی‌شود، فقط برای امضای محلی استفاده می‌شود — دقیقاً همان الگوریتمی که خودِ `verifyTelegramAuth` سمت
apimanager انتظار دارد (`src/lib/telegram-sign.ts`).

- در `npm run dev` این بخش همیشه نمایش داده می‌شود.
- در `npm run build` فقط اگر `VITE_ENABLE_DEV_LOGIN=true` در `.env` باشد نمایش داده می‌شود — در
  production واقعی این را روشن نکنید و به‌جایش دامنه‌ی واقعی را در BotFather ثبت کنید.

## اجرا

```bash
npm install
cp .env.example .env   # و مقادیر را پر کن
npm run dev
```

## متغیرهای محیطی (`.env`)

| متغیر | توضیح |
|---|---|
| `VITE_API_BASE_URL` | آدرس apimanager، مثلاً `http://localhost:8080` (بدون اسلش انتهایی) |
| `VITE_TELEGRAM_BOT_USERNAME` | یوزرنیم ربات تلگرام platform؛ باید همان bot باشد که `BOT_TOKEN` در apimanager به آن اشاره می‌کند. دامنه‌ی این سایت باید در BotFather با `/setdomain` ثبت شود، وگرنه ویجت لاگین کار نمی‌کند. |

## ساخت نسخه‌ی production

```bash
npm run build      # خروجی در dist/
npm run preview    # پیش‌نمایش محلی همان build
```

می‌توانید پوشه‌ی `dist/` را پشت هر وب‌سروری (nginx, Caddy, یا حتی خودِ gin با `static.Serve`) سرو کنید.
apimanager هیچ static file ای سرو نمی‌کند، پس این فرانت باید جدا دیپلوی شود (روی همان دامنه یا ساب‌دامنه‌ای
که در `/setdomain` ثبت کرده‌اید، وگرنه CORS باید در apimanager اضافه شود — فعلاً هندلری برای CORS وجود ندارد).

## نکات فنی

- **استک**: Vite + React 18 + TypeScript، Tailwind CSS، React Router، TanStack Query، Zustand
  (state مدیریت auth/theme/sidebar)، react-hook-form، axios، recharts (نمودارها)، react-i18next (چندزبانه).
- **Auth**: بعد از ورود، `access_token` (کوتاه‌مدت) و `refresh_token` در store نگه‌داری می‌شوند. یک
  axios interceptor روی خطای 401 خودکار `refresh_token` را صدا می‌زند و درخواست را دوباره می‌فرستد؛ اگر
  refresh هم شکست بخورد، کاربر logout و به `/login` هدایت می‌شود.
- **نقش‌ها**: مسیرهای زیر `/admin` فقط برای `role: admin|owner` باز است (چک سمت کلاینت؛ چک واقعی همان
  `middleware.RequireRole` سمت بک‌اند است که همیشه معتبر است).
- **دو زبان (فارسی/English)**: کل رابط با `react-i18next` چندزبانه است — سوییچر زبان در تاپ‌بار (و در
  صفحه‌ی لاگین). جهت صفحه (`dir`) به‌صورت خودکار بین rtl/ltr عوض می‌شود، انتخاب کاربر در `localStorage`
  می‌ماند. متن‌ها در `src/i18n/locales/fa.json` و `en.json` هستند؛ برای اضافه‌کردن زبان سوم، یک فایل JSON
  جدید با همان ساختار کلیدها بساز و در `src/i18n/index.ts` به `resources` و `SUPPORTED_LANGUAGES` اضافه
  کن. تمام کلاس‌های Tailwind از خاصیت‌های منطقی (`ps-`, `pe-`, `start-`, `end-`) استفاده می‌کنند تا با
  تغییر جهت نیازی به کلاس‌های جداگانه نباشد.
- **طراحی داده‌محور**: جدول‌ها (`components/ui/DataTable.tsx`) جستجو، مرتب‌سازی ستونی و صفحه‌بندی دارند؛
  در AdminStats یک نمودار دونات واقعی (recharts) وضعیت instance ها را نشان می‌دهد و KPI cardها دلتای
  واقعی نسبت به poll قبلی را نمایش می‌دهند (نه داده‌ی فرضی). صفحات سنگین‌تر (خصوصاً AdminStats به‌خاطر
  recharts) با `React.lazy` جدا بارگذاری می‌شوند تا bundle اولیه سبک بماند.
- **لاگ درخواست‌ها**: صفحه ادمین /admin/request-logs درخواست‌های API همین مرورگر را در حافظه نگه
  می‌دارد و به سرور ارسال نمی‌کند؛ زنگ اعلان پنل ادمین تعداد خطاهای همین session را نشان می‌دهد.
- **موبایل و دسترس‌پذیری**: پنل کاربر در موبایل نوار دسترسی سریع چهارقسمتی دارد و safe-area دستگاه‌های
  دارای home indicator را رعایت می‌کند. skip link، aria-current مسیر فعال، فوکوس واضح کیبورد و
  prefers-reduced-motion نیز در پوسته مشترک فعال‌اند.
- **پروکسی dev برای دور زدن CORS**: apimanager هیچ CORS middleware ای ندارد. در `npm run dev`،
  `vite.config.ts` مسیر `/api` را به آدرس واقعی apimanager (از `VITE_API_BASE_URL`) پروکسی می‌کند تا
  مرورگر آن را هم‌مبدأ ببیند. در build نهایی این پروکسی وجود ندارد — یا فرانت و apimanager باید هم‌مبدأ
  سرو شوند، یا CORS باید به apimanager اضافه شود.
- **RTL/LTR و فونت**: فونت Vazirmatn هم فارسی و هم لاتین را پوشش می‌دهد، پس نیازی به تعویض فونت بین
  زبان‌ها نیست.
- **ساخت instance جدید**: فرم ابتدا GET /service-types را می‌خواند و سپس قالب‌های فعال همان نوع را
  از GET /templates?type=... می‌گیرد؛ کاربر به‌جای واردکردن شناسه خام، نوع سرویس و نسخه را انتخاب
  می‌کند.
- **تایپ‌های API**: قرارداد پاسخ‌های مصرف‌شده از apimanager در src/lib/types.ts تعریف شده و باید همراه
  تغییر handlerها به‌روز بماند.

## ساختار پوشه‌ها

```
src/
  i18n/
    index.ts                init i18next + تنظیم خودکار dir/lang روی <html>
    locales/fa.json          en.json
  components/
    Layout/AppShell.tsx     شل مشترک: سایدبار دسکتاپ + ناوبری پایین موبایل + تاپ‌بار، زبان، تم و منوی کاربر
    LanguageSwitcher.tsx    سوییچر زبان
    ProtectedRoute.tsx      گارد لاگین و گارد نقش ادمین
    ui/                     Button, Card (+StatCard با delta), Badge, Modal, Input/Select,
                             Skeleton, EmptyState, DataTable (جستجو/مرتب‌سازی/صفحه‌بندی), ActionMenu
  lib/
    api.ts                  axios instance + auto-refresh token + لاگ درخواست‌ها
    auth-store.ts           zustand store برای session
    theme-store.ts          zustand store برای دارک‌مود
    sidebar-store.ts        zustand store برای جمع/باز بودن سایدبار
    request-log-store.ts    zustand store لاگ درخواست‌های همین مرورگر
    chart-colors.ts         پالت رنگ ثابت برای نمودارها
    types.ts                تایپ‌های TypeScript پاسخ‌های API
  pages/
    Login.tsx               split-screen: پنل برندینگ + کارت لاگین (+ ورود آزمایشی)
    Dashboard.tsx
    Instances.tsx           DataTable + فیلتر وضعیت + منوی عملیات
    Plans.tsx
    Payments.tsx
    RequestLogs.tsx         ابزار مشاهده درخواست‌های همین session برای ادمین
    admin/AdminStats.tsx    KPI + نمودار دونات recharts
    admin/AdminServers.tsx
    admin/AdminTemplates.tsx
```
