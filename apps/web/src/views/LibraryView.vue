<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { ArrowLeft, LoaderCircle, RefreshCw, Search, Settings, Video } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import { Button } from '@/components/ui/button'
import { buttonVariants } from '@/components/ui/button/variants'
import { Input } from '@/components/ui/input'
import MediaCard from '@/components/media/MediaCard.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getLibraries } from '@/lib/api/libraries'
import { getMedia, getScanStatus, scanLibrary } from '@/lib/api/media'

const route = useRoute()
const queryClient = useQueryClient()
const libraryID = computed(() => String(route.params.libraryId))
const search = ref('')
const filter = ref<'all' | 'unwatched' | 'in-progress' | 'watched'>('all')
const sort = ref<'name' | 'recent' | 'duration'>('name')

const librariesQuery = useQuery({ queryKey: ['libraries'], queryFn: ({ signal }) => getLibraries(signal) })
const library = computed(() => librariesQuery.data.value?.find((item) => item.id === libraryID.value))
const mediaQuery = useQuery({
  queryKey: computed(() => ['media', libraryID.value]),
  queryFn: ({ signal }) => getMedia(libraryID.value, signal),
})
const scanStatusQuery = useQuery({
  queryKey: computed(() => ['scan-status', libraryID.value]),
  queryFn: ({ signal }) => getScanStatus(libraryID.value, signal),
  refetchInterval: 2_000,
})
const scanMutation = useMutation({
  mutationFn: () => scanLibrary(libraryID.value),
  onSuccess: async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['media', libraryID.value] }),
      queryClient.invalidateQueries({ queryKey: ['libraries'] }),
      queryClient.invalidateQueries({ queryKey: ['scan-status', libraryID.value] }),
    ])
  },
})
const isScanning = computed(
  () => scanMutation.isPending.value || scanStatusQuery.data.value?.state === 'scanning',
)

watch(
  () => scanStatusQuery.data.value?.state,
  async (state, previousState) => {
    if (state === 'idle' && previousState === 'scanning') {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['media', libraryID.value] }),
        queryClient.invalidateQueries({ queryKey: ['libraries'] }),
        queryClient.invalidateQueries({ queryKey: ['home'] }),
      ])
    }
  },
)

const filteredMedia = computed(() => {
  const query = search.value.trim().toLocaleLowerCase('fr')
  const items = (mediaQuery.data.value ?? []).filter((item) => {
    if (query && !item.filename.toLocaleLowerCase('fr').includes(query)) return false
    if (filter.value === 'watched') return item.completed
    if (filter.value === 'in-progress') return item.progressMs > 0 && !item.completed
    if (filter.value === 'unwatched') return item.progressMs === 0 && !item.completed
    return true
  })
  return [...items].sort((first, second) => {
    if (sort.value === 'recent') return Date.parse(second.modifiedAt) - Date.parse(first.modifiedAt)
    if (sort.value === 'duration') return second.durationMs - first.durationMs
    return first.filename.localeCompare(second.filename, 'fr', { sensitivity: 'base' })
  })
})

</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(ellipse_at_top,rgba(124,58,237,0.10),transparent_68%)]" />
    <div class="relative mx-auto max-w-[1600px] px-5 py-10 lg:px-10 lg:py-14">
      <RouterLink :to="{ name: 'libraries' }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground">
        <ArrowLeft class="size-4" />
        Bibliothèques
      </RouterLink>
      <div class="mt-6 flex items-end justify-between gap-6">
        <div class="min-w-0">
          <h1 class="truncate text-3xl font-bold tracking-tight sm:text-4xl">{{ library?.name ?? 'Bibliothèque' }}</h1>
        </div>
        <div class="flex gap-2">
          <Button variant="secondary" :disabled="isScanning" @click="scanMutation.mutate()">
            <RefreshCw :class="isScanning && 'animate-spin'" />
            {{ isScanning ? 'Analyse…' : 'Analyser' }}
          </Button>
          <RouterLink :to="{ name: 'library-settings', params: { libraryId: libraryID } }" :class="buttonVariants({ variant: 'ghost', size: 'icon' })" aria-label="Paramètres de la bibliothèque">
            <Settings />
          </RouterLink>
        </div>
      </div>

      <div class="mt-10 flex items-center justify-between">
        <h2 class="text-xl font-semibold tracking-tight">Vidéos</h2>
        <p v-if="mediaQuery.data.value" class="text-xs text-muted-foreground">{{ mediaQuery.data.value.length }} vidéo{{ mediaQuery.data.value.length > 1 ? 's' : '' }}</p>
      </div>

      <div class="mt-5 flex flex-col gap-3 sm:flex-row">
        <label class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="search" type="search" placeholder="Rechercher une vidéo…" class="h-11 rounded-xl border-white/10 bg-white/5 pl-10 pr-4 shadow-none placeholder:text-muted-foreground/60 focus-visible:ring-primary/15" />
        </label>
        <Select v-model="filter">
          <SelectTrigger class="h-11 min-w-36 rounded-xl border-white/10 bg-white/5 shadow-none" aria-label="Filtrer les vidéos">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Toutes</SelectItem>
            <SelectItem value="unwatched">Non vues</SelectItem>
            <SelectItem value="in-progress">En cours</SelectItem>
            <SelectItem value="watched">Vues</SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="sort">
          <SelectTrigger class="h-11 min-w-40 rounded-xl border-white/10 bg-white/5 shadow-none" aria-label="Trier les vidéos">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="name">Nom</SelectItem>
            <SelectItem value="recent">Plus récentes</SelectItem>
            <SelectItem value="duration">Durée</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="mediaQuery.isPending.value" class="grid min-h-72 place-items-center"><LoaderCircle class="size-6 animate-spin text-primary" /></div>
      <div v-else-if="mediaQuery.isError.value" class="mt-5 grid min-h-64 place-items-center rounded-2xl border border-red-400/15 bg-red-400/[0.025] text-center">
        <div><p class="text-sm font-medium">Impossible de charger les vidéos</p><Button class="mt-4" variant="secondary" @click="mediaQuery.refetch()">Réessayer</Button></div>
      </div>
      <div v-else-if="mediaQuery.data.value?.length === 0" class="mt-5 grid min-h-64 place-items-center rounded-2xl border border-dashed border-white/10 bg-white/[0.015] px-6 text-center">
        <div>
          <Video class="mx-auto size-7 text-muted-foreground" />
          <p class="mt-3 text-sm font-medium">Aucune vidéo indexée</p>
          <p class="mt-1 text-xs text-muted-foreground">Les nouvelles vidéos apparaîtront automatiquement dans cette bibliothèque.</p>
          <Button class="mt-5" variant="secondary" :disabled="isScanning" @click="scanMutation.mutate()"><RefreshCw :class="isScanning && 'animate-spin'" />{{ isScanning ? 'Analyse en cours…' : 'Analyser maintenant' }}</Button>
        </div>
      </div>
      <div v-else-if="filteredMedia.length" class="mt-5 grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
        <MediaCard v-for="item in filteredMedia" :key="item.id" :item="item" />
      </div>
      <div v-else class="mt-5 grid min-h-52 place-items-center rounded-2xl border border-dashed border-white/10 text-center">
        <div><Search class="mx-auto size-6 text-muted-foreground" /><p class="mt-3 text-sm font-medium">Aucune vidéo ne correspond</p><button class="mt-2 text-xs text-primary" @click="search = ''; filter = 'all'">Réinitialiser les filtres</button></div>
      </div>
    </div>
  </section>
</template>
