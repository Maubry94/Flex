import { cva, type VariantProps } from 'class-variance-authority'

export const buttonVariants = cva(
  'inline-flex shrink-0 items-center justify-center gap-2 rounded-full text-sm font-semibold transition-all outline-none focus-visible:ring-2 focus-visible:ring-primary/60 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground shadow-lg shadow-primary/20 hover:bg-primary/90',
        destructive: 'bg-destructive text-white shadow-lg shadow-destructive/15 hover:bg-destructive/90 focus-visible:ring-destructive/40',
        outline: 'border border-white/10 bg-transparent text-foreground hover:bg-white/8',
        secondary: 'bg-white/8 text-foreground ring-1 ring-white/10 hover:bg-white/12',
        ghost: 'text-muted-foreground hover:bg-white/8 hover:text-foreground',
      },
      size: {
        default: 'h-10 px-5',
        sm: 'h-8 px-3 text-xs',
        lg: 'h-12 px-6 text-base',
        icon: 'size-10 p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export type ButtonVariants = VariantProps<typeof buttonVariants>
