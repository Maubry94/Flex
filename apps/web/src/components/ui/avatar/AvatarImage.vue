<script setup lang="ts">
import type { AvatarImageProps } from "reka-ui"
import { AvatarImage, useForwardProps } from "reka-ui"
import { reactiveOmit } from "@vueuse/core"
import { asForwardedProps } from "@/lib/utils"

const props = defineProps<AvatarImageProps>()
const delegatedProps = reactiveOmit(props, "src")
const forwardedProps = asForwardedProps(useForwardProps(delegatedProps))
</script>

<template>
  <AvatarImage
    data-slot="avatar-image"
    :src="props.src"
    v-bind="forwardedProps"
    class="aspect-square size-full"
  >
    <slot />
  </AvatarImage>
</template>
