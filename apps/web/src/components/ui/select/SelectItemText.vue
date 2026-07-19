<script setup lang="ts">
import type { SelectItemTextProps } from "reka-ui"
import { SelectItemText } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<SelectItemTextProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <SelectItemText
    data-slot="select-item-text"
    v-bind="forwardedProps"
  >
    <slot />
  </SelectItemText>
</template>
