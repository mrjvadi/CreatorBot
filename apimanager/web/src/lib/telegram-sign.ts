/**
 * پیاده‌سازی مرورگری همان الگوریتمی که apimanager برای تأیید امضای Telegram Login Widget
 * استفاده می‌کند (رجوع به verifyTelegramAuth در internal/handler/handler.go):
 *
 *   data-check-string = تمام فیلدهای غیرخالی (به‌جز hash)، مرتب‌شده الفبایی، به‌صورت "key=value"
 *                        و جداشده با "\n"
 *   secret_key        = SHA256(bot_token)
 *   hash              = HMAC-SHA256(secret_key, data-check-string) به‌صورت hex
 *
 * فقط برای «ورود آزمایشی/توسعه» استفاده می‌شود، جایی که دامنه هنوز در BotFather با /setdomain
 * ثبت نشده و ویجت رسمی تلگرام قابل استفاده نیست. توکن ربات هرگز به سرور فرستاده نمی‌شود؛ فقط در
 * مرورگر خودِ کاربر برای امضا کردن محلی به کار می‌رود (دقیقاً همان کاری که خودِ Telegram سمت سرورش
 * انجام می‌دهد).
 */

async function sha256(input: string): Promise<ArrayBuffer> {
  return crypto.subtle.digest("SHA-256", new TextEncoder().encode(input));
}

async function hmacSha256Hex(keyBytes: ArrayBuffer, message: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    keyBytes,
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const signature = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(message));
  return Array.from(new Uint8Array(signature))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export async function signTelegramAuth(
  botToken: string,
  fields: Record<string, string | number | undefined>
): Promise<string> {
  const entries = Object.entries(fields).filter(
    ([, v]) => v !== undefined && v !== null && String(v) !== ""
  ) as [string, string | number][];
  entries.sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));

  const dataCheckString = entries.map(([k, v]) => `${k}=${v}`).join("\n");
  const secretKey = await sha256(botToken);
  return hmacSha256Hex(secretKey, dataCheckString);
}
