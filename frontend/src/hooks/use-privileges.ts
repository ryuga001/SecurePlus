"use client";

import * as React from "react";

import { hasPrivilege } from "@/lib/privileges";
import { useGetPrivilegesQuery } from "@/store/api/privileges-api";

const EMPTY: readonly string[] = [];

export function usePrivileges() {
  const { data, isLoading, isError, error, refetch } = useGetPrivilegesQuery();

  const privileges = data ?? EMPTY;

  const has = React.useCallback(
    (required: string) => hasPrivilege(privileges, required),
    [privileges],
  );

  return { privileges, has, loading: isLoading, isError, error, refetch };
}
