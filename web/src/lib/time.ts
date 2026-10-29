const RTF = new Intl.RelativeTimeFormat("zh-CN", { numeric: "auto" });

const units: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 24 * 3600],
  ["month", 30 * 24 * 3600],
  ["day", 24 * 3600],
  ["hour", 3600],
  ["minute", 60],
];

// relativeTime renders "3 分钟前" style labels. Anything under a minute
// reads "刚刚", and dates beyond a year fall back to a short date.
export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diff = (then - Date.now()) / 1000;
  if (Math.abs(diff) < 60) return "刚刚";
  for (const [unit, seconds] of units) {
    if (Math.abs(diff) >= seconds) return RTF.format(Math.round(diff / seconds), unit);
  }
  return RTF.format(Math.round(diff), "second");
}

export function absoluteTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "" : d.toLocaleString("zh-CN");
}
