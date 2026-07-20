<script setup lang="ts">
import type { ContextMenuSubEmits, ContextMenuSubProps } from "reka-ui"
import {
  ContextMenuSub,
  useForwardPropsEmits,
} from "reka-ui"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<ContextMenuSubProps>()
const emits = defineEmits<ContextMenuSubEmits>()

const forwarded = asForwardedProps(useForwardPropsEmits(props, emits))
</script>

<template>
  <ContextMenuSub
    data-slot="context-menu-sub"
    v-bind="forwarded"
  >
    <slot />
  </ContextMenuSub>
</template>
