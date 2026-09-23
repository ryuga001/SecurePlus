"use client";

import { Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { Field } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { emptyTargetValues, targetFields, type SourceType } from "@/lib/data-discovery";
import { cn } from "@/lib/utils";

export type TargetRow = { key: string; values: Record<string, string> };

export function newTargetRow(sourceType: SourceType): TargetRow {
  return { key: crypto.randomUUID(), values: emptyTargetValues(sourceType) };
}

export function TargetListEditor({
  sourceType,
  value,
  onChange,
  errors,
  disabled,
  max = 25,
  min = 1,
}: {
  sourceType: SourceType;
  value: TargetRow[];
  onChange: (next: TargetRow[]) => void;
  errors?: Record<number, Record<string, string>>;
  disabled?: boolean;
  max?: number;
  min?: number;
}) {
  const t = useTranslations("data-discovery");

  const fields = targetFields(sourceType);

  return (
    <div className="flex flex-col gap-3">
      {value.map((row, index) => (
        <div key={row.key} className="rounded-lg border p-4">
          <div className="mb-3 flex items-center justify-between gap-2">
            <span className="text-sm font-medium">
              {t("policies.dialog.targetIndex", { index: index + 1 })}
            </span>

            <Button
              type="button"
              size="icon"
              variant="ghost"
              disabled={disabled || value.length <= min}
              aria-label={t("policies.dialog.removeTarget")}
              onClick={() => onChange(value.filter((item) => item.key !== row.key))}
            >
              <Trash2 />
            </Button>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {fields.map((field) => {
              const fieldId = `target-${row.key}-${field.key}`;

              return (
                <Field
                  key={field.key}
                  id={fieldId}
                  label={t(`fields.${field.labelKey}`)}
                  error={errors?.[index]?.[field.key]}
                >
                  <Input
                    id={fieldId}
                    value={row.values[field.key] ?? ""}
                    placeholder={field.placeholder}
                    disabled={disabled}
                    spellCheck={false}
                    aria-invalid={errors?.[index]?.[field.key] ? true : undefined}
                    className={cn(field.mono && "font-mono")}
                    onChange={(event) =>
                      onChange(
                        value.map((item) =>
                          item.key === row.key
                            ? { ...item, values: { ...item.values, [field.key]: event.target.value } }
                            : item,
                        ),
                      )
                    }
                  />
                </Field>
              );
            })}
          </div>
        </div>
      ))}

      <Button
        type="button"
        variant="outline"
        className="self-start"
        disabled={disabled || value.length >= max}
        onClick={() => onChange([...value, newTargetRow(sourceType)])}
      >
        <Plus />
        {t("policies.dialog.addTarget")}
      </Button>
    </div>
  );
}
