/** پالت ثابت برای نمودارها (recharts کلاس تیلویند نمی‌پذیرد، باید هگز مستقیم بدهیم). */
export const CHART_COLORS = {
  running: "#10b981",
  active: "#10b981",
  pending: "#f59e0b",
  stopped: "#94a3b8",
  error: "#ef4444",
  failed: "#ef4444",
  online: "#10b981",
  offline: "#94a3b8",
  brand: "#3182f6",
  grid: "#e2e8f0",
  gridDark: "#1e293b",
} as const;

export function colorForStatus(status: string): string {
  const key = status?.toLowerCase?.() as keyof typeof CHART_COLORS;
  return CHART_COLORS[key] ?? "#94a3b8";
}
