<script setup lang="ts">
import type { DialogTriggerProps } from "reka-ui"
import { DialogTrigger } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<DialogTriggerProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <DialogTrigger
    data-slot="sheet-trigger"
    v-bind="forwardedProps"
  >
    <slot />
  </DialogTrigger>
</template>
