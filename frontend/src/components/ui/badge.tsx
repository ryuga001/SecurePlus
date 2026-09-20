import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

const badgeVariants = cva(
  "group/badge inline-flex h-6 w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-sm border whitespace-nowrap transition-all focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&>svg]:pointer-events-none [&>svg]:size-3! [&>svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "border-primary bg-primary text-primary-foreground [a]:hover:bg-primary-hover",
        secondary: "border-border bg-secondary text-secondary-foreground [a]:hover:bg-surface-container-high",
        destructive:
          "border-error-border bg-error-container text-error-text [a]:hover:bg-error-container/80",
        outline: "border-border text-foreground [a]:hover:bg-muted",
        ghost: "border-transparent bg-surface-container text-text-secondary hover:bg-surface-container-high",
        link: "border-transparent text-primary underline-offset-4 hover:underline",
        success: "border-success-border bg-success-container text-success-text [a]:hover:bg-success-container/80",
        warning: "border-warning-border bg-warning-container text-warning-text [a]:hover:bg-warning-container/80",
        error: "border-error-border bg-error-container text-error-text [a]:hover:bg-error-container/80",
        info: "border-info-border bg-info-container text-info-text [a]:hover:bg-info-container/80",
        neutral: "border-border bg-surface-container text-neutral-text [a]:hover:bg-surface-container-high",
      },
      size: {
        default: "h-6 rounded-sm px-2 text-xs [&>svg]:size-3!",
        sm: "h-5.5 gap-0.5 rounded-sm px-1.5 text-[0.6875rem] [&>svg]:size-2.5!",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Badge({
  className,
  variant = "default",
  size = "default",
  render,
  ...props
}: useRender.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ variant, size }), className),
      },
      props
    ),
    render,
    state: {
      slot: "badge",
      variant,
    },
  })
}

export { Badge, badgeVariants }