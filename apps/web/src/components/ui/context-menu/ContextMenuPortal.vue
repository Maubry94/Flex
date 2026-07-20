<script setup lang="ts">
import type { ContextMenuPortalProps } from "reka-ui"
import { ContextMenuPortal } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<ContextMenuPortalProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <ContextMenuPortal
    data-slot="context-menu-portal"
    v-bind="forwardedProps"
  >
    <slot />
  </ContextMenuPortal>
</template>
