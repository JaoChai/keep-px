import * as React from "react"
import { cn } from "@/lib/utils"
import { buttonVariants, type ButtonProps } from "./button-variants"

export function Button({
  className,
  variant,
  size,
  ref,
  ...props
}: ButtonProps & { ref?: React.Ref<HTMLButtonElement> }) {
  return (
    <button
      type="button"
      ref={ref}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export type { ButtonProps }
