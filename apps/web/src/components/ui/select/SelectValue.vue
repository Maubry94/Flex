<script setup lang="ts">
import type { SelectValueProps } from "reka-ui"
import { SelectValue } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<SelectValueProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <SelectValue
    data-slot="select-value"
    v-bind="forwardedProps"
  >
    <slot />
  </SelectValue>
</template>
