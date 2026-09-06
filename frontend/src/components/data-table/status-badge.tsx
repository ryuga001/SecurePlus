import { cn } from "@/lib/utils";

const tones: Record<string, string> = {
  active: "border-emerald-600/30 bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400",
  enabled: "border-emerald-600/30 bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400",
  success: "border-emerald-600/30 bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400",
  pending: "border-amber-500/30 bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-400",
  warning: "border-amber-500/30 bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-400",
  invited: "border-primary/30 bg-primary/8 text-primary",
  info: "border-primary/30 bg-primary/8 text-primary",
  disabled: "border-border bg-muted text-muted-foreground",
  inactive: "border-border bg-muted text-muted-foreground",
  blocked: "border-destructive/30 bg-destructive/8 text-destructive",
  failed: "border-destructive/30 bg-destructive/8 text-destructive",
  error: "border-destructive/30 bg-destructive/8 text-destructive",
};

export function StatusBadge({ status }: { status: string }) {
  const tone = tones[status?.toLowerCase?.()] ?? "border-border bg-muted text-muted-foreground";

  return (
    <span
      className={cn(
        "inline-flex items-center border px-2 py-0.5 text-xs font-medium capitalize",
        tone
      )}
    >
      {status}
    </span>
  );
}
