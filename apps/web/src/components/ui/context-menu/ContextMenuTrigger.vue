<script setup lang="ts">
import type { ContextMenuTriggerProps } from "reka-ui"
import { ContextMenuTrigger, useForwardProps } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<ContextMenuTriggerProps>()

const forwardedProps = asForwardedProps(useForwardProps(props))
</script>

<template>
  <ContextMenuTrigger
    data-slot="context-menu-trigger"
    v-bind="forwardedProps"
  >
    <slot />
  </ContextMenuTrigger>
</template>
