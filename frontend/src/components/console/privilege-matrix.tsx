"use client";

import { ChevronRight, Search } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { usePrivilegeLabels } from "@/hooks/use-privilege-labels";
import { groupPrivileges } from "@/lib/privilege-catalog";
import { cn } from "@/lib/utils";

export function PrivilegeMatrix({
  id,
  available,
  value,
  onChange,
  readOnly = false,
  disabled = false,
  invalid = false,
}: {
  id: string;
  available: readonly string[];
  value: readonly string[];
  onChange: (next: string[]) => void;
  readOnly?: boolean;
  disabled?: boolean;
  invalid?: boolean;
}) {
  const t = useTranslations("console.privileges");
  const labels = usePrivilegeLabels();
  const [search, setSearch] = React.useState("");
  const [expanded, setExpanded] = React.useState<Set<string>>(() => new Set());

  const selected = React.useMemo(() => new Set(value), [value]);
  const locked = readOnly || disabled;
  const searching = search.trim() !== "";

  const allGroups = React.useMemo(() => groupPrivileges(available), [available]);

  const groups = React.useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return allGroups;

    return allGroups
      .map((group) => ({
        ...group,
        entries: labels.group(group.key).toLowerCase().includes(term)
          ? group.entries
          : group.entries.filter((entry) => labels.privilege(entry.name).toLowerCase().includes(term)),
      }))
      .filter((group) => group.entries.length > 0);
  }, [allGroups, labels, search]);

  const everything = available.length > 0 && available.every((name) => selected.has(name));
  const anything = available.some((name) => selected.has(name));
  const allExpanded = allGroups.length > 0 && allGroups.every((group) => expanded.has(group.key));

  function apply(names: string[], checked: boolean) {
    const next = new Set(selected);

    for (const name of names) {
      if (checked) next.add(name);
      else next.delete(name);
    }

    onChange(available.filter((name) => next.has(name)));
  }

  function toggle(key: string) {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  function summary(names: string[], actions: string[]) {
    const picked = actions.filter((_, index) => selected.has(names[index]));

    if (picked.length === 0) return "";
    if (picked.length === actions.length) return t("allActions");

    return picked.map((action) => labels.action(action)).join(", ");
  }

  return (
    <div id={id} tabIndex={-1} aria-invalid={invalid || undefined} className="flex flex-col gap-3 outline-none">
      <div className="relative">
        <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-9"
          value={search}
          placeholder={t("search")}
          aria-label={t("search")}
          onChange={(event) => setSearch(event.target.value)}
        />
      </div>

      <div className={cn("overflow-hidden rounded-md border", invalid && "border-error ring-1 ring-error")}>
        <div className="flex items-center gap-2.5 border-b bg-surface-container-low px-3 py-2">
          <Checkbox
            checked={everything}
            indeterminate={anything && !everything}
            disabled={locked || available.length === 0}
            aria-label={t("allPrivileges")}
            onCheckedChange={(checked) => onChange(checked === true || (anything && !everything) ? [...available] : [])}
          />
          <span className="text-sm font-medium">{t("allPrivileges")}</span>

          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="ml-auto"
            disabled={searching || allGroups.length === 0}
            onClick={() => setExpanded(allExpanded ? new Set() : new Set(allGroups.map((group) => group.key)))}
          >
            {allExpanded ? t("collapseAll") : t("expandAll")}
          </Button>
        </div>

        {groups.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm text-muted-foreground">{t("noMatches")}</p>
        ) : (
          <ul className="divide-y">
            {groups.map((group) => {
              const names = group.entries.map((entry) => entry.name);
              const actions = group.entries.map((entry) => entry.action);
              const picked = names.filter((name) => selected.has(name)).length;
              const all = picked === names.length;
              const some = picked > 0 && !all;
              const open = searching || expanded.has(group.key);
              const groupLabel = labels.group(group.key);
              const panel = `${id}-${group.key.replaceAll(".", "-")}`;

              return (
                <li key={group.key}>
                  <div
                    className={cn(
                      "flex items-center gap-2 px-3 py-2 transition-colors hover:bg-surface-container-low",
                      picked > 0 && "bg-primary/5",
                    )}
                  >
                    <button
                      type="button"
                      aria-expanded={open}
                      aria-controls={panel}
                      aria-label={t("toggleGroup", { group: groupLabel })}
                      disabled={searching}
                      onClick={() => toggle(group.key)}
                      className="flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:pointer-events-none"
                    >
                      <ChevronRight className={cn("size-4 transition-transform", open && "rotate-90")} />
                    </button>

                    <Checkbox
                      checked={all}
                      indeterminate={some}
                      disabled={locked}
                      aria-label={t("selectGroup", { group: groupLabel })}
                      onCheckedChange={(checked) => apply(names, checked === true || some)}
                    />

                    <button
                      type="button"
                      tabIndex={-1}
                      disabled={searching}
                      onClick={() => toggle(group.key)}
                      className="flex min-w-0 flex-1 items-center justify-between gap-3 py-1 text-left disabled:cursor-default"
                    >
                      <span className="truncate text-sm font-medium">{groupLabel}</span>
                      <span className="truncate text-xs text-muted-foreground">{summary(names, actions)}</span>
                    </button>
                  </div>

                  {open ? (
                    <ul id={panel} className="border-t bg-surface-container-low/40 py-1">
                      {group.entries.map((entry) => (
                        <li key={entry.name}>
                          <label
                            title={entry.name}
                            className="flex cursor-pointer items-center gap-2.5 py-1.5 pr-3 pl-17 text-sm text-text-secondary transition-colors hover:bg-surface-container-low hover:text-foreground"
                          >
                            <Checkbox
                              checked={selected.has(entry.name)}
                              disabled={locked}
                              aria-label={labels.privilege(entry.name)}
                              onCheckedChange={(checked) => apply([entry.name], checked === true)}
                            />
                            {labels.action(entry.action)}
                          </label>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
