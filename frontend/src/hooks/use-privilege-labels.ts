"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { humanize, messageKey, parsePrivilege } from "@/lib/privilege-catalog";

export function usePrivilegeLabels() {
  const t = useTranslations("console.privileges");

  return React.useMemo(() => {
    const group = (key: string) => {
      const path = `groups.${messageKey(key)}`;
      return t.has(path) ? t(path) : humanize(key);
    };

    const action = (key: string) => {
      const path = `actions.${messageKey(key)}`;
      return t.has(path) ? t(path) : humanize(key);
    };

    const privilege = (name: string) => {
      const entry = parsePrivilege(name);
      return `${group(entry.group)} · ${action(entry.action)}`;
    };

    return { group, action, privilege };
  }, [t]);
}
