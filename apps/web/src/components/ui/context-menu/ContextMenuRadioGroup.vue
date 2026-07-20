<script setup lang="ts">
import type { ContextMenuRadioGroupEmits, ContextMenuRadioGroupProps } from "reka-ui"
import {
  ContextMenuRadioGroup,
  useForwardPropsEmits,
} from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<ContextMenuRadioGroupProps>()
const emits = defineEmits<ContextMenuRadioGroupEmits>()

const forwarded = asForwardedProps(useForwardPropsEmits(props, emits))
</script>

<template>
  <ContextMenuRadioGroup
    data-slot="context-menu-radio-group"
    v-bind="forwarded"
  >
    <slot />
  </ContextMenuRadioGroup>
</template>
