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
        success: <CircleCheckIcon className="size-4.5 text-success-text" />,
        info: <InfoIcon className="size-4.5 text-info-text" />,
        warning: <TriangleAlertIcon className="size-4.5 text-warning-text" />,
        error: <CircleAlertIcon className="size-4.5 text-error-text" />,
        loading: <Loader2Icon className="size-4.5 animate-spin text-primary" />,
      }}
      style={
        {
          "--normal-bg": "var(--surface)",
          "--normal-text": "var(--text-primary)",
          "--normal-border": "var(--border)",
          "--border-radius": "12px",
          "--width": "22rem",
        } as React.CSSProperties
      }
      toastOptions={{
        classNames: {
          toast:
            "!rounded-lg !items-start !gap-3 !border !border-border !bg-surface !p-4 !text-foreground !shadow-e2",
          success: "!border-l-[3px] !border-l-success",
          error: "!border-l-[3px] !border-l-error",
          warning: "!border-l-[3px] !border-l-warning",
          info: "!border-l-[3px] !border-l-info",
          loading: "!border-l-[3px] !border-l-primary",
          icon: "!m-0 !mt-0.5 !size-4.5 !shrink-0",
          content: "!gap-1",
          title: "!text-sm !font-medium !leading-snug !text-foreground",
          description: "!text-sm !text-muted-foreground",
          actionButton: "!rounded-md !bg-primary !text-primary-foreground",
          cancelButton: "!rounded-md !bg-muted !text-muted-foreground",
          closeButton: "!rounded-md !border-border !bg-surface !text-muted-foreground",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
