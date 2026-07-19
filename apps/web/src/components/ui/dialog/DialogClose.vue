<script setup lang="ts">
import type { DialogCloseProps } from "reka-ui"
import { DialogClose } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<DialogCloseProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <DialogClose
    data-slot="dialog-close"
    v-bind="forwardedProps"
  >
    <slot />
  </DialogClose>
</template>
