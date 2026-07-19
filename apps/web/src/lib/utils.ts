import type { ClassValue } from "clsx"
import { clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function asForwardedProps(props: object): Record<string, unknown> {
  return props as unknown as Record<string, unknown>
}
