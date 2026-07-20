<script setup lang="ts">
import type { DropdownMenuTriggerProps } from "reka-ui"
import { DropdownMenuTrigger, useForwardProps } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<DropdownMenuTriggerProps>()

const forwardedProps = asForwardedProps(useForwardProps(props))
</script>

<template>
  <DropdownMenuTrigger
    data-slot="dropdown-menu-trigger"
    v-bind="forwardedProps"
  >
    <slot />
  </DropdownMenuTrigger>
</template>
