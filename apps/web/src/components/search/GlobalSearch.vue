<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { Film, LoaderCircle } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import { CommandDialog, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { searchMedia, thumbnailURL } from '@/lib/api/media'
import { mediaTitle } from '@/lib/media-title'

const open = defineModel<boolean>('open', { required: true })
const router = useRouter()
const input = ref('')
const query = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | undefined

const searchQuery = useQuery({
  queryKey: computed(() => ['global-search', query.value]),
  queryFn: ({ signal }) => searchMedia(query.value, signal),
  enabled: computed(() => open.value && query.value.length >= 2),
})

watch(input, (value) => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    query.value = value.trim()
  }, 250)
})

watch(open, (isOpen) => {
  if (!isOpen) {
    input.value = ''
    query.value = ''
  }
})

function handleShortcut(event: KeyboardEvent): void {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    open.value = !open.value
  }
}

function updateInput(value: string | number | undefined): void {
  input.value = String(value ?? '')
}

async function selectMedia(mediaID: string): Promise<void> {
  open.value = false
  await router.push({ name: 'video', params: { mediaId: mediaID } })
}

function formatDuration(durationMs: number): string {
  const totalSeconds = Math.floor(durationMs / 1_000)
  const hours = Math.floor(totalSeconds / 3_600)
  const minutes = Math.floor((totalSeconds % 3_600) / 60)
  const seconds = totalSeconds % 60
  const time = [minutes, seconds].map((value) => String(value).padStart(2, '0')).join(':')
  return hours > 0 ? `${String(hours).padStart(2, '0')}:${time}` : time
}

onMounted(() => {
  window.addEventListener('keydown', handleShortcut)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleShortcut)
  if (debounceTimer) clearTimeout(debounceTimer)
})
</script>

<template>
  <CommandDialog v-model:open="open" title="Rechercher" description="Recherchez une vidéo dans toutes vos bibliothèques">
    <CommandInput :model-value="input" placeholder="Rechercher une vidéo…" @update:model-value="updateInput" />
    <CommandList class="max-h-[min(60dvh,32rem)]">
      <div v-if="input.trim().length < 2" class="px-6 py-10 text-center text-sm text-muted-foreground">Saisissez au moins deux caractères.</div>
      <div v-else-if="searchQuery.isPending.value" class="grid min-h-28 place-items-center"><LoaderCircle class="size-5 animate-spin text-primary" /></div>
      <div v-else-if="searchQuery.isError.value" class="px-6 py-10 text-center text-sm text-red-300">La recherche a échoué.</div>
      <div v-else-if="searchQuery.data.value?.length === 0" class="px-6 py-10 text-center text-sm text-muted-foreground">Aucune vidéo trouvée.</div>
      <CommandGroup v-else heading="Vidéos" class="p-2">
        <CommandItem v-for="item in searchQuery.data.value" :key="item.id" :value="item.id" class="gap-3 rounded-xl p-2" @select="selectMedia(item.id)">
          <div class="relative grid aspect-video w-20 shrink-0 place-items-center overflow-hidden rounded-lg bg-muted">
            <Film class="size-5 text-muted-foreground" />
            <img :src="thumbnailURL(item.id)" alt="" class="absolute inset-0 size-full object-cover" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate font-medium">{{ mediaTitle(item.filename) }}</p>
            <p class="mt-1 truncate text-xs text-muted-foreground">{{ item.libraryName }} · {{ formatDuration(item.durationMs) }}</p>
          </div>
        </CommandItem>
      </CommandGroup>
    </CommandList>
  </CommandDialog>
</template>
