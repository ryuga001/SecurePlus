"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { DataTable } from "@/components/data-table/data-table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";

import { apiErrorMessage } from "@/lib/api-error";
import {
  useCreateEmailGroupMutation,
  useListGroupMembersQuery,
  useUpdateEmailGroupMutation,
  type EmailGroup,
} from "@/store/api/email-groups-api";
import {
  useListEmailUsersQuery,
  type EmailUser,
} from "@/store/api/email-users-api";

import { DrawerWrapper } from "../drawer/drawer";
import type {
  ColumnConfig,
  FilterConfig,
} from "../data-table/types";

export function GroupDrawer({
  open,
  onOpenChange,
  group,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group?: EmailGroup | null;
}) {
  const t = useTranslations("groups");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={
        group
          ? t("dialog.editTitle")
          : t("dialog.addTitle")
      }
      description={t("dialog.description")}
      width="2xl"
    >
      <GroupForm
        key={group?.id ?? "create"}
        group={group ?? null}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}

function GroupForm({
  group,
  onClose,
}: {
  group: EmailGroup | null;
  onClose: () => void;
}) {
  const t = useTranslations("groups");

  const [name, setName] = React.useState(group?.name ?? "");
  const [selectedIds, setSelectedIds] = React.useState<Set<number>>(
    new Set(),
  );
  const [selectedUsers, setSelectedUsers] = React.useState<
    Map<number, EmailUser>
  >(new Map());
  const [error, setError] = React.useState("");

  const [createGroup, { isLoading: creating }] =
    useCreateEmailGroupMutation();

  const [updateGroup, { isLoading: updating }] =
    useUpdateEmailGroupMutation();

  const pending = creating || updating;

  const handleSelectionChange = React.useCallback(
    (ids: Set<number>, users: Map<number, EmailUser>) => {
      setSelectedIds(new Set(ids));
      setSelectedUsers(new Map(users));
    },
    [],
  );

  async function onSubmit(
    event: React.FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    const trimmed = name.trim();

    if (!trimmed) {
      setError(t("dialog.nameRequired"));
      return;
    }

    setError("");

    const member_ids = Array.from(selectedIds);

    try {
      if (group) {
        const updated = await updateGroup({
          id: group.id,
          name: trimmed,
          member_ids,
        }).unwrap();

        setName(updated.name);
        toast.success(t("dialog.updated"));
        onClose();
      } else {
        const created = await createGroup({
          name: trimmed,
          member_ids,
        }).unwrap();

        setName(created.name);
        toast.success(t("dialog.created"));
        onClose();
      }
    } catch (caught) {
      setError(
        apiErrorMessage(caught, t("dialog.saveFailed")),
      );
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="flex flex-col gap-4"
    >
      <FormError message={error} />

      <Field
        id="group-name"
        label={t("dialog.name")}
      >
        <Input
          id="group-name"
          required
          autoFocus
          placeholder={t("dialog.namePlaceholder")}
          value={name}
          disabled={pending}
          onChange={(event) => {
            setName(event.target.value);
            setError("");
          }}
        />
      </Field>

      <Members
        groupId={group?.id ?? null}
        selectedIds={selectedIds}
        selectedUsers={selectedUsers}
        onSelectionChange={handleSelectionChange}
      />

      <div className="flex justify-end gap-2 border-t pt-4">
        <Button
          type="button"
          variant="outline"
          onClick={onClose}
          disabled={pending}
        >
          {t("dialog.cancel")}
        </Button>

        <Button
          type="submit"
          disabled={pending}
        >
          {pending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : null}

          {group
            ? t("dialog.saveChanges")
            : t("dialog.create")}
        </Button>
      </div>
    </form>
  );
}

function Members({
  groupId,
  selectedIds,
  selectedUsers,
  onSelectionChange,
}: {
  groupId: number | null;
  selectedIds: Set<number>;
  selectedUsers: Map<number, EmailUser>;
  onSelectionChange: (
    ids: Set<number>,
    users: Map<number, EmailUser>,
  ) => void;
}) {
  const t = useTranslations("groups");

  const [onlySelected, setOnlySelected] = React.useState(false);

  const { data: members } = useListGroupMembersQuery(
    {
      groupId: groupId as number,
      page: 1,
      pageSize: 100,
    },
    {
      skip: groupId === null,
    },
  );

  const hydrated = React.useRef(false);

  React.useEffect(() => {
    if (groupId === null || !members?.items || hydrated.current) {
      return;
    }

    hydrated.current = true;

    const ids = new Set<number>();
    const users = new Map<number, EmailUser>();

    for (const user of members.items) {
      ids.add(user.id);
      users.set(user.id, user);
    }

    onSelectionChange(ids, users);
  }, [groupId, members?.items, onSelectionChange]);

  const filters = React.useMemo<FilterConfig[]>(
    () => [
      {
        key: "search",
        label: t("dialog.searchUsers"),
        type: "text",
        placeholder: t("dialog.searchUsers"),
      },
    ],
    [t],
  );

  const columns = React.useMemo<ColumnConfig<EmailUser>[]>(
    () => [
      {
        key: "select",
        label: "",
        sortable: false,
        render: (_value, user) => (
          <Checkbox
            checked={selectedIds.has(user.id)}
            onCheckedChange={(checked) => {
              const ids = new Set(selectedIds);
              const users = new Map(selectedUsers);

              if (checked === true) {
                ids.add(user.id);
                users.set(user.id, user);
              } else {
                ids.delete(user.id);
                users.delete(user.id);
              }

              onSelectionChange(ids, users);
            }}
          />
        ),
      },
      {
        key: "email",
        label: t("dialog.name"),
        render: (_value, user) => {
          const name =
            `${user.first_name} ${user.last_name}`.trim();

          return (
            <div>
              <div className="font-medium">
                {name || user.email}
              </div>

              <div className="text-xs text-muted-foreground">
                {user.email}
              </div>
            </div>
          );
        },
      },
    ],
    [
      onSelectionChange,
      selectedIds,
      selectedUsers,
      t,
    ],
  );

  const selectedList = Array.from(
    selectedUsers.values(),
  ).filter((user) =>
    selectedIds.has(user.id),
  );

  if (onlySelected) {
    return (
      <section className="flex flex-col gap-3 border-t pt-5">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium">
            {t("dialog.members")}
          </h3>

          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {selectedIds.size} selected
            </span>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setOnlySelected(false)}
            >
              {t("dialog.showAll")}
            </Button>
          </div>
        </div>

        {selectedList.length === 0 ? (
          <div className="rounded-md border p-8 text-center text-sm text-muted-foreground">
            {t("dialog.noSelectedUsers")}
          </div>
        ) : (
          <div className="max-h-[360px] overflow-y-auto rounded-md border">
            {selectedList.map((user) => {
              const displayName =
                `${user.first_name} ${user.last_name}`.trim() ||
                user.email;

              return (
                <div
                  key={user.id}
                  className="flex items-center justify-between border-b px-3 py-2 last:border-b-0"
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">
                      {displayName}
                    </div>

                    <div className="truncate text-xs text-muted-foreground">
                      {user.email}
                    </div>
                  </div>

                  <Checkbox
                    checked
                    onCheckedChange={() => {
                      const ids = new Set(selectedIds);
                      const users = new Map(selectedUsers);

                      ids.delete(user.id);
                      users.delete(user.id);

                      onSelectionChange(ids, users);
                    }}
                  />
                </div>
              );
            })}
          </div>
        )}
      </section>
    );
  }

  return (
    <section className="flex min-h-0 flex-col gap-3 border-t pt-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-sm font-medium">
            {t("dialog.members")}
          </h3>

          <p className="mt-1 text-xs text-muted-foreground">
            {t("dialog.membersDescription")}
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <span className="rounded-md bg-muted px-2 py-1 text-xs font-medium">
            {selectedIds.size} selected
          </span>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setOnlySelected(true)}
            disabled={selectedIds.size === 0}
          >
            {t("dialog.onlySelected")}
          </Button>
        </div>
      </div>

      <DataTable
        query={useListEmailUsersQuery}
        columns={columns}
        filters={filters}
        getRowId={(row) => String(row.id)}
        defaultSort={{
          key: "email",
          direction: "asc",
        }}
        emptyMessage={t("dialog.noUsers")}
      />
    </section>
  );
}