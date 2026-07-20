<script setup lang="ts">
import type { ContextMenuGroupProps } from "reka-ui"
import { ContextMenuGroup } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<ContextMenuGroupProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <ContextMenuGroup
    data-slot="context-menu-group"
    v-bind="forwardedProps"
  >
    <slot />
  </ContextMenuGroup>
</template>
