"use client";

import { Loader2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { CheckboxList } from "@/components/ui/checkbox-list";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useAddGroupMemberMutation,
  useCreateEmailGroupMutation,
  useListGroupMembersQuery,
  useRemoveGroupMemberMutation,
  useUpdateEmailGroupMutation,
  type EmailGroup,
} from "@/store/api/email-groups-api";
import { useListEmailUsersQuery } from "@/store/api/email-users-api";

const ALL_USERS = { page: 1, pageSize: 100, filters: {} };

function Members({ groupId }: { groupId: number }) {
  const { data: members, isLoading: loadingMembers } = useListGroupMembersQuery(groupId);
  const { data: users, isLoading: loadingUsers } = useListEmailUsersQuery(ALL_USERS);

  const [addMember] = useAddGroupMemberMutation();
  const [removeMember] = useRemoveGroupMemberMutation();

  const selected = (members?.items ?? []).map((member) => member.id);

  const options = (users?.items ?? []).map((user) => ({
    id: user.id,
    label: `${user.first_name} ${user.last_name}`.trim() || user.email,
    hint: user.email,
  }));

  async function onToggle(emailUserId: number, checked: boolean) {
    try {
      if (checked) {
        await addMember({ groupId, emailUserId }).unwrap();
      } else {
        await removeMember({ groupId, emailUserId }).unwrap();
      }
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not update membership"));
    }
  }

  return (
    <div className="flex flex-col gap-3 border-t pt-5">
      <div>
        <h3 className="text-sm font-medium">Members</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          Ticking a name adds them to this group straight away.
        </p>
      </div>

      {loadingMembers || loadingUsers ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading users...
        </div>
      ) : (
        <CheckboxList
          options={options}
          selected={selected}
          onToggle={onToggle}
          searchPlaceholder="Search users"
          emptyMessage="Add email users first"
        />
      )}
    </div>
  );
}

function GroupForm({ group, onClose }: { group: EmailGroup | null; onClose: () => void }) {
  const [name, setName] = React.useState(group?.name ?? "");
  const [groupId, setGroupId] = React.useState<number | null>(group?.id ?? null);
  const [error, setError] = React.useState("");

  const [createGroup, { isLoading: creating }] = useCreateEmailGroupMutation();
  const [updateGroup, { isLoading: updating }] = useUpdateEmailGroupMutation();

  const pending = creating || updating;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmed = name.trim();
    if (!trimmed) {
      setError("Enter a group name.");
      return;
    }

    setError("");

    try {
      if (groupId === null) {
        const created = await createGroup({ name: trimmed }).unwrap();
        setGroupId(created.id);
        setName(created.name);
        toast.success("Group created");
      } else {
        const updated = await updateGroup({ id: groupId, name: trimmed }).unwrap();
        setName(updated.name);
        toast.success("Group updated");
      }
    } catch (caught) {
      setError(apiErrorMessage(caught, "Could not save this group"));
    }
  }

  return (
    <>
      <DialogTitle>{group ? "Edit group" : "Add group"}</DialogTitle>
      <DialogDescription>
        A group collects email users so a policy can target them together.
      </DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-5">
        <FormError message={error} />

        <Field id="group-name" label="Group name">
          <Input
            id="group-name"
            required
            autoFocus
            placeholder="Finance"
            value={name}
            disabled={pending}
            onChange={(event) => {
              setName(event.target.value);
              setError("");
            }}
          />
        </Field>

        {groupId === null ? (
          <p className="border-t pt-5 text-xs text-muted-foreground">
            Save the group to start adding members.
          </p>
        ) : (
          <Members groupId={groupId} />
        )}

        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" variant="outline" onClick={onClose}>
            {groupId === null ? "Cancel" : "Done"}
          </Button>
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : null}
            {groupId === null ? "Create group" : "Save changes"}
          </Button>
        </div>
      </form>
    </>
  );
}

export function GroupDialog({
  open,
  onOpenChange,
  group,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group?: EmailGroup | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-4rem)] max-w-lg overflow-y-auto">
        <GroupForm group={group ?? null} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
