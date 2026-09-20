"use client"

import { Radio as RadioPrimitive } from "@base-ui/react/radio"
import { RadioGroup as RadioGroupPrimitive } from "@base-ui/react/radio-group"
import * as React from "react"
import { cn } from "cn"

function RadioGroup({ className, ...props }: React.ComponentProps<typeof RadioGroupPrimitive>) {
  return <RadioGroupPrimitive data-slot="radio-group" className={cn("grid gap-2", className)} {...props} />
}

function RadioGroupItem({ className, ...props }: React.ComponentProps<typeof RadioPrimitive.Root>) {
  return (
    <RadioPrimitive.Root
      data-slot="radio-group-item"
      className={cn(
        "relative mt-0.5 inline-flex size-4 shrink-0 items-center justify-center rounded-full border border-input outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 after:absolute after:size-2 after:rounded-full after:bg-primary after:opacity-0 after:transition-opacity data-checked:border-primary data-checked:after:opacity-100 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30",
        className
      )}
      {...props}
    />
  )
}

export { RadioGroup, RadioGroupItem }