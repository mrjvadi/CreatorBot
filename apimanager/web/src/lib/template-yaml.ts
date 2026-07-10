/**
 * پارسر بسیار سبک و محدود — فقط برای فرمتِ خاصِ «مانیفست تمپلیت» که این‌جا مستند شده، نه یک
 * پارسرِ عمومیِ YAML. دلیل نبودن یک کتابخانه‌ی واقعی (مثل js-yaml): هیچ پکیج YAML ای در
 * node_modules نصب نبود و طبق دستور صریح کاربر، پکیج جدید بدون اجازه نصب نمی‌شود — پس این‌جا
 * فقط همان زیرمجموعه‌ی محدودی که برای این فرم لازم است دستی parse می‌شود.
 * بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «یک قالب بذار که YAML بدیم و خودش فیلدها را پر کند».
 *
 * فرمتِ پشتیبانی‌شده (نمونه):
 *
 * name: Uploader Bot v2
 * type: uploader
 * image_name: registry.example.com/uploader
 * image_tag: v2.3.1
 * description: توضیح ربات
 * is_free: false
 * config_schema:
 *   - key: CHANNEL_ID
 *     label: آیدی کانال
 *     type: string
 *     required: true
 *   - key: MODE
 *     label: حالت
 *     type: select
 *     options: [fast, safe]
 */

export interface ParsedConfigField {
  key: string;
  label: string;
  type: string;
  required?: boolean;
  default?: string;
  options?: string[];
}

export interface ParsedTemplateManifest {
  name?: string;
  type?: string;
  image_name?: string;
  image_tag?: string;
  description?: string;
  is_free?: boolean;
  config_schema?: ParsedConfigField[];
}

function stripQuotes(raw: string): string {
  const t = raw.trim();
  if ((t.startsWith('"') && t.endsWith('"')) || (t.startsWith("'") && t.endsWith("'"))) {
    return t.slice(1, -1);
  }
  return t;
}

function parseInlineArray(raw: string): string[] {
  const inner = raw.trim().replace(/^\[/, "").replace(/\]$/, "");
  if (!inner.trim()) return [];
  return inner
    .split(",")
    .map((s) => stripQuotes(s.trim()))
    .filter(Boolean);
}

function applyFieldKV(field: Partial<ParsedConfigField>, key: string, rawValue: string) {
  const value = rawValue.trim();
  switch (key) {
    case "key":
      field.key = stripQuotes(value);
      break;
    case "label":
      field.label = stripQuotes(value);
      break;
    case "type":
      field.type = stripQuotes(value);
      break;
    case "required":
      field.required = stripQuotes(value) === "true";
      break;
    case "default":
      field.default = stripQuotes(value);
      break;
    case "options":
      field.options = parseInlineArray(value);
      break;
  }
}

export function parseTemplateYaml(text: string): ParsedTemplateManifest {
  const lines = text.split(/\r?\n/);
  const result: ParsedTemplateManifest = {};
  const schema: ParsedConfigField[] = [];
  let inSchema = false;
  let currentField: Partial<ParsedConfigField> | null = null;

  const flushField = () => {
    if (currentField && currentField.key && currentField.label && currentField.type) {
      schema.push(currentField as ParsedConfigField);
    }
    currentField = null;
  };

  for (const rawLine of lines) {
    const line = rawLine.replace(/#.*$/, "");
    if (!line.trim()) continue;

    if (/^config_schema\s*:\s*$/.test(line.trim())) {
      inSchema = true;
      continue;
    }

    if (inSchema) {
      if (!/^\s/.test(line)) {
        // خط بدون indent یعنی از بخش config_schema خارج شدیم
        inSchema = false;
        flushField();
      } else {
        const listItem = line.match(/^\s*-\s*(.*)$/);
        if (listItem) {
          flushField();
          currentField = {};
          const rest = listItem[1];
          if (rest.trim()) {
            const kv = rest.match(/^(\w+)\s*:\s*(.*)$/);
            if (kv) applyFieldKV(currentField, kv[1], kv[2]);
          }
          continue;
        }
        const kv = line.match(/^\s+(\w+)\s*:\s*(.*)$/);
        if (kv && currentField) {
          applyFieldKV(currentField, kv[1], kv[2]);
        }
        continue;
      }
    }

    if (!inSchema) {
      const kv = line.match(/^(\w+)\s*:\s*(.*)$/);
      if (!kv) continue;
      const key = kv[1];
      const value = stripQuotes(kv[2]);
      switch (key) {
        case "name":
          result.name = value;
          break;
        case "type":
          result.type = value;
          break;
        case "image_name":
          result.image_name = value;
          break;
        case "image_tag":
          result.image_tag = value;
          break;
        case "description":
          result.description = value;
          break;
        case "is_free":
          result.is_free = value === "true";
          break;
      }
    }
  }
  flushField();
  if (schema.length > 0) result.config_schema = schema;
  return result;
}
