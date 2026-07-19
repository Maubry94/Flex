<script setup lang="ts">
import type { SelectGroupProps } from "reka-ui"
import { SelectGroup } from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<SelectGroupProps>()
const forwardedProps = asForwardedProps(props)
</script>

<template>
  <SelectGroup
    data-slot="select-group"
    v-bind="forwardedProps"
  >
    <slot />
  </SelectGroup>
</template>
