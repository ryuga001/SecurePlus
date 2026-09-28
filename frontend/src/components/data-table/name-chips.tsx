import { cn } from "@/lib/utils";

export type NameChip = {
  key: string | number;
  label: string;
  title?: string;
};

export function NameChips({
  items,
  total,
  className,
}: {
  items: NameChip[];
  total: number;
  className?: string;
}) {
  const remaining = Math.max(total - items.length, 0);

  if (items.length === 0 && total === 0) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }

  return (
    <div className={cn("flex min-w-0 flex-wrap items-center gap-1.5", className)}>
      {items.map((item) => (
        <span
          key={item.key}
          title={item.title ?? item.label}
          className="max-w-40 truncate rounded-md bg-muted px-2 py-1 text-xs font-medium"
        >
          {item.label}
        </span>
      ))}

      {remaining > 0 ? (
        <span className="rounded-md border px-2 py-1 text-xs font-medium">+{remaining}</span>
      ) : null}
    </div>
  );
}
