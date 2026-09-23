"use client";

import { Combobox as ComboboxPrimitive } from "@base-ui/react/combobox";
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react";

import { cn } from "@/lib/utils";

function Combobox<Value>(props: ComboboxPrimitive.Root.Props<Value>) {
  return <ComboboxPrimitive.Root data-slot="combobox" {...props} />;
}

function ComboboxInput({ className, ...props }: ComboboxPrimitive.Input.Props) {
  return (
    <ComboboxPrimitive.Input
      data-slot="combobox-input"
      className={cn(
        "h-10 w-full min-w-0 rounded-md border border-input bg-background px-3 py-2 pr-9 text-sm text-foreground transition-colors outline-none placeholder:text-text-disabled focus-visible:border-primary focus-visible:shadow-[0_0_0_2px_var(--primary-container)] disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted disabled:opacity-50 aria-invalid:border-error",
        className,
      )}
      {...props}
    />
  );
}

function ComboboxTrigger({ className, ...props }: ComboboxPrimitive.Trigger.Props) {
  return (
    <ComboboxPrimitive.Trigger
      data-slot="combobox-trigger"
      className={cn(
        "absolute top-1/2 right-2 -translate-y-1/2 rounded-sm p-1 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40",
        className,
      )}
      {...props}
    >
      <ChevronsUpDownIcon className="size-4" />
    </ComboboxPrimitive.Trigger>
  );
}

function ComboboxContent({
  align = "start",
  alignOffset = 0,
  side = "bottom",
  sideOffset = 4,
  className,
  ...props
}: ComboboxPrimitive.Popup.Props &
  Pick<ComboboxPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset">) {
  return (
    <ComboboxPrimitive.Portal>
      <ComboboxPrimitive.Positioner
        className="isolate z-50 outline-none"
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
      >
        <ComboboxPrimitive.Popup
          data-slot="combobox-content"
          className={cn(
            "z-50 max-h-(--available-height) w-(--anchor-width) min-w-56 origin-(--transform-origin) overflow-x-hidden overflow-y-auto rounded-md bg-popover p-1 text-popover-foreground shadow-e2 ring-1 ring-foreground/10 duration-100 outline-none data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0",
            className,
          )}
          {...props}
        />
      </ComboboxPrimitive.Positioner>
    </ComboboxPrimitive.Portal>
  );
}

function ComboboxList({ className, ...props }: ComboboxPrimitive.List.Props) {
  return (
    <ComboboxPrimitive.List
      data-slot="combobox-list"
      className={cn("flex flex-col", className)}
      {...props}
    />
  );
}

function ComboboxItem({ className, children, ...props }: ComboboxPrimitive.Item.Props) {
  return (
    <ComboboxPrimitive.Item
      data-slot="combobox-item"
      className={cn(
        "relative flex cursor-pointer items-start gap-2 rounded-sm py-2 pr-2 pl-8 text-sm outline-none select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-highlighted:bg-accent data-highlighted:text-accent-foreground",
        className,
      )}
      {...props}
    >
      <ComboboxPrimitive.ItemIndicator className="absolute left-2 top-2.5">
        <CheckIcon className="size-4" />
      </ComboboxPrimitive.ItemIndicator>
      {children}
    </ComboboxPrimitive.Item>
  );
}

function ComboboxEmpty({ className, ...props }: ComboboxPrimitive.Empty.Props) {
  return (
    <ComboboxPrimitive.Empty
      data-slot="combobox-empty"
      className={cn("px-3 py-6 text-center text-sm text-muted-foreground", className)}
      {...props}
    />
  );
}

function ComboboxStatus({ className, ...props }: ComboboxPrimitive.Status.Props) {
  return (
    <ComboboxPrimitive.Status
      data-slot="combobox-status"
      className={cn("px-3 py-2 text-xs text-muted-foreground", className)}
      {...props}
    />
  );
}

export {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxStatus,
  ComboboxTrigger,
};
