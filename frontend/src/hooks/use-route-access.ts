"use client";

import * as React from "react";

import { useIsAdmin } from "@/hooks/use-is-admin";
import { usePrivileges } from "@/hooks/use-privileges";
import type { RouteAccess } from "@/lib/routes";

export function useRouteAccess(): RouteAccess {
  const isAdmin = useIsAdmin();
  const { privileges, loading, isError } = usePrivileges();

  return React.useMemo(
    () => ({ isAdmin, granted: loading || isError ? null : privileges }),
    [isAdmin, loading, isError, privileges],
  );
}
