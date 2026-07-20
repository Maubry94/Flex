<script setup lang="ts">
import type { DropdownMenuGroupProps } from "reka-ui"
import { DropdownMenuGroup } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<DropdownMenuGroupProps>()
const forwarded = asForwardedProps(props)
</script>

<template>
  <DropdownMenuGroup
    data-slot="dropdown-menu-group"
    v-bind="forwarded"
  >
    <slot />
  </DropdownMenuGroup>
</template>
