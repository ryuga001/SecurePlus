"use client";

import { CircleAlert } from "lucide-react";
import { useTranslations } from "next-intl";

import { LoadingRegion } from "@/components/loading/skeletons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { useListFileTypesQuery } from "@/store/api/policies-api";

export function FileTypePicker({
  value,
  onChange,
  disabled,
}: {
  value: string[];
  onChange: (next: string[]) => void;
  disabled?: boolean;
}) {
  const t = useTranslations("data-discovery.policies.dialog");
  const table = useTranslations("table");

  const { data, isLoading, isError, refetch } = useListFileTypesQuery();

  const fileTypes = data?.items ?? [];
  const selected = new Set(value);

  if (isLoading) {
    return (
      <LoadingRegion label={t("oneMoment")} className="overflow-hidden rounded-lg border">
        <div className="flex items-center justify-between gap-2 border-b px-4 py-3">
          <Skeleton className="h-5 w-24" />
          <div className="flex gap-2">
            <Skeleton className="h-8 w-20" />
            <Skeleton className="h-8 w-16" />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2 p-3 sm:grid-cols-3">
          {Array.from({ length: 6 }, (_, index) => (
            <div key={index} className="flex items-center gap-2 rounded-md border p-2">
              <Skeleton className="h-5 w-9 shrink-0 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                <Skeleton className="h-3.5 w-12" />
                <Skeleton className="h-3 w-20" />
              </div>
            </div>
          ))}
        </div>
      </LoadingRegion>
    );
  }

  if (isError) {
    return (
      <div className="flex min-h-40 flex-col items-center justify-center gap-2 text-sm">
        <CircleAlert className="size-5 text-error" />
        <p className="text-muted-foreground">{t("loadFileTypesFailed")}</p>
        <Button type="button" size="sm" variant="outline" onClick={() => refetch()}>
          {table("tryAgain")}
        </Button>
      </div>
    );
  }

  if (fileTypes.length === 0) {
    return (
      <div className="flex min-h-40 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
        {t("noFileTypes")}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
        <Badge variant="secondary">{t("selectedCount", { count: value.length })}</Badge>

        <div className="flex gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={disabled || value.length === fileTypes.length}
            onClick={() => onChange(fileTypes.map((item) => item.extension))}
          >
            {t("selectAll")}
          </Button>

          <Button
            type="button"
            size="sm"
            variant="ghost"
            disabled={disabled || value.length === 0}
            onClick={() => onChange([])}
          >
            {t("clearSelection")}
          </Button>
        </div>
      </div>

      <div className="grid max-h-64 grid-cols-2 gap-2 overflow-y-auto p-3 sm:grid-cols-3">
        {fileTypes.map((fileType) => {
          const checked = selected.has(fileType.extension);

          return (
            <label
              key={fileType.id}
              className={cn(
                "flex cursor-pointer items-center gap-2 rounded-md border p-2 transition hover:bg-muted/50",
                checked && "border-primary/60 bg-primary/5",
              )}
            >
              <Switch
                checked={checked}
                disabled={disabled}
                onCheckedChange={(next) =>
                  onChange(
                    next
                      ? Array.from(new Set([...value, fileType.extension]))
                      : value.filter((item) => item !== fileType.extension),
                  )
                }
              />

              <div className="min-w-0">
                <div className="truncate text-sm font-medium">.{fileType.extension}</div>
                <div className="truncate text-xs text-muted-foreground">{fileType.label}</div>
              </div>
            </label>
          );
        })}
      </div>
    </div>
  );
}
