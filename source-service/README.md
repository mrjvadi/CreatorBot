# Source Service

این سرویس برای forward کردن محتوا از یک کانال منبع به کانال مقصد از طریق MTProto (Telegram UserBot API) طراحی شده است.

## وضعیت

⚠️ این سرویس stub است و پیاده‌سازی کامل نشده.

## پیاده‌سازی نیاز دارد

برای پیاده‌سازی کامل به کتابخانه [gotd/td](https://github.com/gotd/td) نیاز است:

```bash
go get github.com/gotd/td
```

## ENV های لازم

```env
TG_APP_ID=       # از my.telegram.org/apps
TG_APP_HASH=     # از my.telegram.org/apps  
TG_PHONE=        # شماره تلگرام
TG_SESSION_FILE=/app/sessions/session.json
TG_SOURCE_CHANNEL=  # ID کانال منبع
TG_DELIVERY_CHANNEL= # ID کانال مقصد
```

## توجه

استفاده از UserBot API ممکن است با شرایط خدمات تلگرام در تضاد باشد.
فقط برای archive شخصی استفاده کنید.
