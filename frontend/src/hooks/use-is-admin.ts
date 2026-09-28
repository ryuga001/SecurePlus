"use client";

import { useAuth } from "@/components/auth-provider";

export const ADMIN_ROLE_TYPE = "admin";

export function useIsAdmin() {
  const { identity } = useAuth();
  const roleType = identity?.user.role_type;

  return roleType === undefined || roleType === ADMIN_ROLE_TYPE;
}
