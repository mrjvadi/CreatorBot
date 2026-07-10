import { HTMLAttributes } from "react";
import { ArrowDown, ArrowUp } from "lucide-react";
import clsx from "clsx";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={clsx("glass-card p-5", className)} {...props} />;
}

export function StatCard({
  label,
  value,
  icon,
  hint,
  delta,
  accent = "brand",
}: {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  hint?: string;
  /** تغییر نسبت به آخرین poll (داده‌ی واقعی، نه فرضی) — مثبت=سبز، منفی=قرمز، صفر=خنثی */
  delta?: number;
  accent?: "brand" | "success" | "warning" | "danger";
}) {
  const accentClasses: Record<string, string> = {
    brand: "bg-violet-50 text-violet-600 dark:bg-violet-500/15 dark:text-violet-300",
    success: "bg-success-50 text-success-600 dark:bg-success-500/15 dark:text-success-400",
    warning: "bg-warning-50 text-warning-600 dark:bg-warning-500/15 dark:text-warning-400",
    danger: "bg-danger-50 text-danger-600 dark:bg-danger-500/15 dark:text-danger-400",
  };

  return (
    <Card className="relative flex items-start justify-between overflow-hidden">
      <div
        className="pointer-events-none absolute -top-10 -end-10 h-24 w-24 rounded-full opacity-0 blur-2xl dark:opacity-100"
        style={{
          background:
            accent === "brand"
              ? "radial-gradient(circle, rgba(124,58,237,0.35), transparent 70%)"
              : undefined,
        }}
        aria-hidden="true"
      />
      <div className="relative">
        <p className="text-sm text-slate-500 dark:text-slate-400">{label}</p>
        <div className="mt-1.5 flex items-center gap-2">
          <p className="text-2xl font-bold tabular-nums">{value}</p>
          {typeof delta === "number" && delta !== 0 && (
            <span
              className={clsx(
                "flex items-center gap-0.5 text-xs font-medium",
                delta > 0 ? "text-success-600 dark:text-success-400" : "text-danger-600 dark:text-danger-400"
              )}
            >
              {delta > 0 ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />}
              {Math.abs(delta)}
            </span>
          )}
        </div>
        {hint && <p className="mt-1 text-xs text-slate-400">{hint}</p>}
      </div>
      {icon && <div className={clsx("relative rounded-xl p-2.5", accentClasses[accent])}>{icon}</div>}
    </Card>
  );
}
