"use client";

import {
  CircleAlertIcon,
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  TriangleAlertIcon,
} from "lucide-react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      className="toaster group"
      icons={{
        success: <CircleCheckIcon className="size-4.5 text-emerald-600" />,
        info: <InfoIcon className="size-4.5 text-primary" />,
        warning: <TriangleAlertIcon className="size-4.5 text-amber-600" />,
        error: <CircleAlertIcon className="size-4.5 text-destructive" />,
        loading: <Loader2Icon className="size-4.5 animate-spin text-primary" />,
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "0px",
          "--width": "22rem",
        } as React.CSSProperties
      }
      toastOptions={{
        classNames: {
          toast:
            "!rounded-none !items-start !gap-3 !border !border-border !border-l-[3px] !border-l-primary !bg-popover !p-4 !text-popover-foreground !shadow-xl !shadow-primary/10",
          success: "!border-l-emerald-600 !bg-emerald-50/70 dark:!bg-emerald-950/30",
          error: "!border-l-destructive !bg-destructive/6",
          warning: "!border-l-amber-500 !bg-amber-50/70 dark:!bg-amber-950/30",
          info: "!border-l-primary !bg-primary/6",
          loading: "!border-l-primary",
          icon: "!m-0 !mt-0.5 !size-4.5 !shrink-0",
          content: "!gap-1",
          title: "!text-sm !font-medium !leading-snug !text-foreground",
          description: "!text-sm !text-muted-foreground",
          actionButton: "!rounded-none !bg-primary !text-primary-foreground",
          cancelButton: "!rounded-none !bg-muted !text-muted-foreground",
          closeButton: "!rounded-none !border-border !bg-popover !text-muted-foreground",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
