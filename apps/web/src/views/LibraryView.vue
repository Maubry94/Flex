<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { ArrowLeft, CheckCheck, CircleAlert, EyeOff, Heart, HeartOff, ListChecks, LoaderCircle, RefreshCw, Search, Settings, Video, X } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { toast } from 'vue-sonner'

import { Button } from '@/components/ui/button'
import { buttonVariants } from '@/components/ui/button/variants'
import { Input } from '@/components/ui/input'
import MediaCard from '@/components/media/MediaCard.vue'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getLibraries } from '@/lib/api/libraries'
import { getLibraryFolders } from '@/lib/api/folders'
import { getMedia, getScanStatus, scanLibrary, updateMedia } from '@/lib/api/media'
import { getCollections, getMediaCollections, setMediaCollections } from '@/lib/api/collections'
import { saveProgress } from '@/lib/api/progress'
import { getTagAssignments } from '@/lib/api/tags'

const route = useRoute()
const queryClient = useQueryClient()
const libraryID = computed(() => String(route.params.libraryId))
const search = ref('')
const filter = ref<'all' | 'favorite' | 'unwatched' | 'in-progress' | 'watched'>('all')
const tagFilter = ref('all')
const folderFilter = ref('all')
const sort = ref<'name' | 'recent' | 'recorded' | 'duration'>('name')
const selectionMode = ref(false)
const selectedIDs = ref<string[]>([])
const selectedCollectionID = ref('')
const isBulkUpdating = ref(false)
const scanWasAutomatic = ref(false)

const librariesQuery = useQuery({ queryKey: ['libraries'], queryFn: ({ signal }) => getLibraries(signal) })
const library = computed(() => librariesQuery.data.value?.find((item) => item.id === libraryID.value))
const mediaQuery = useQuery({
  queryKey: computed(() => ['media', libraryID.value]),
  queryFn: ({ signal }) => getMedia(libraryID.value, signal),
})
const assignmentsQuery = useQuery({ queryKey: ['tag-assignments'], queryFn: ({ signal }) => getTagAssignments(signal) })
const collectionsQuery = useQuery({ queryKey: ['collections'], queryFn: ({ signal }) => getCollections(signal) })
const foldersQuery = useQuery({ queryKey: computed(() => ['library-folders', libraryID.value]), queryFn: ({ signal }) => getLibraryFolders(libraryID.value, signal) })
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
    toast.success('Analyse terminée')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'L’analyse a échoué'),
})
const isScanning = computed(
  () => scanMutation.isPending.value || ['pending', 'scanning'].includes(scanStatusQuery.data.value?.state ?? ''),
)

watch(
  () => scanStatusQuery.data.value?.state,
  async (state, previousState) => {
    if ((state === 'pending' || state === 'scanning') && previousState !== 'pending' && previousState !== 'scanning') {
      scanWasAutomatic.value = !scanMutation.isPending.value
    }
    if ((state === 'completed' || state === 'failed') && (previousState === 'pending' || previousState === 'scanning')) {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['media', libraryID.value] }),
        queryClient.invalidateQueries({ queryKey: ['libraries'] }),
        queryClient.invalidateQueries({ queryKey: ['home'] }),
      ])
      if (scanWasAutomatic.value) {
        const result = scanStatusQuery.data.value?.result
        if (state === 'completed') {
          toast.success(result ? `Bibliothèque actualisée · ${String(result.indexed)} vidéo${result.indexed === 1 ? '' : 's'} ajoutée${result.indexed === 1 ? '' : 's'} ou modifiée${result.indexed === 1 ? '' : 's'}` : 'Bibliothèque actualisée')
        } else {
          toast.error('L’actualisation automatique a échoué')
        }
      }
      scanWasAutomatic.value = false
    }
  },
)

const filteredMedia = computed(() => {
  const query = search.value.trim().toLocaleLowerCase('fr')
  const items = (mediaQuery.data.value ?? []).filter((item) => {
    if (query && !item.title.toLocaleLowerCase('fr').includes(query) && !item.filename.toLocaleLowerCase('fr').includes(query)) return false
    if (tagFilter.value !== 'all' && !(assignmentsQuery.data.value ?? []).some((assignment) => assignment.mediaId === item.id && assignment.tag.id === tagFilter.value)) return false
    if (folderFilter.value !== 'all' && !(foldersQuery.data.value ?? []).some((assignment) => assignment.mediaId === item.id && (folderFilter.value === 'root' ? assignment.folder === '' : assignment.folder === folderFilter.value))) return false
    if (filter.value === 'favorite') return item.favorite
    if (filter.value === 'watched') return item.completed
    if (filter.value === 'in-progress') return item.progressMs > 0 && !item.completed
    if (filter.value === 'unwatched') return item.progressMs === 0 && !item.completed
    return true
  })
  return [...items].sort((first, second) => {
    if (sort.value === 'recent') return Date.parse(second.modifiedAt) - Date.parse(first.modifiedAt)
    if (sort.value === 'recorded') {
      if (!first.recordedAt) return second.recordedAt ? 1 : 0
      if (!second.recordedAt) return -1
      return Date.parse(second.recordedAt) - Date.parse(first.recordedAt)
    }
    if (sort.value === 'duration') return second.durationMs - first.durationMs
    return first.title.localeCompare(second.title, 'fr', { sensitivity: 'base' })
  })
})

const availableTags = computed(() => {
  const mediaIDs = new Set((mediaQuery.data.value ?? []).map((item) => item.id))
  const tags = new Map<string, { id: string, name: string }>()
  for (const assignment of assignmentsQuery.data.value ?? []) {
    if (mediaIDs.has(assignment.mediaId)) tags.set(assignment.tag.id, assignment.tag)
  }
  return [...tags.values()].sort((first, second) => first.name.localeCompare(second.name, 'fr'))
})
const availableFolders = computed(() => [...new Set((foldersQuery.data.value ?? []).map((item) => item.folder))].sort((first, second) => first.localeCompare(second, 'fr')))

function toggleSelection(id: string, selected: boolean): void {
  selectedIDs.value = selected ? [...selectedIDs.value, id] : selectedIDs.value.filter((item) => item !== id)
}

function closeSelection(): void {
  selectionMode.value = false
  selectedIDs.value = []
  selectedCollectionID.value = ''
}

async function runBulk(action: 'favorite' | 'unfavorite' | 'watched' | 'unwatched' | 'collection'): Promise<void> {
  const items = (mediaQuery.data.value ?? []).filter((item) => selectedIDs.value.includes(item.id))
  if (!items.length || isBulkUpdating.value) return
  isBulkUpdating.value = true
  try {
    if (action === 'favorite' || action === 'unfavorite') {
      await Promise.all(items.map((item) => updateMedia(item.id, { title: item.title, description: item.description, recordedAt: item.recordedAt?.slice(0, 10) ?? null, favorite: action === 'favorite' })))
    } else if (action === 'watched' || action === 'unwatched') {
      await Promise.all(items.map((item) => saveProgress(item.id, { positionMs: action === 'watched' ? item.durationMs : 0, durationMs: item.durationMs })))
    } else if (selectedCollectionID.value) {
      await Promise.all(items.map(async (item) => {
        const current = await getMediaCollections(item.id)
        const ids = new Set(current.map((collection) => collection.id)); ids.add(selectedCollectionID.value)
        await setMediaCollections(item.id, [...ids])
      }))
    }
    await Promise.all([queryClient.invalidateQueries({ queryKey: ['media'] }), queryClient.invalidateQueries({ queryKey: ['home'] }), queryClient.invalidateQueries({ queryKey: ['favorites'] }), queryClient.invalidateQueries({ queryKey: ['collections'] }), queryClient.invalidateQueries({ queryKey: ['collection-media'] })])
    toast.success(`${String(items.length)} vidéo${items.length > 1 ? 's' : ''} mise${items.length > 1 ? 's' : ''} à jour`)
    closeSelection()
  } catch (error) { toast.error(error instanceof Error ? error.message : 'Impossible de modifier les vidéos') } finally { isBulkUpdating.value = false }
}

</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(ellipse_at_top,rgba(124,58,237,0.10),transparent_68%)]" />
    <div class="relative mx-auto max-w-[1600px] px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <RouterLink :to="{ name: 'libraries' }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground">
        <ArrowLeft class="size-4" />
        Bibliothèques
      </RouterLink>
      <div class="mt-6 flex flex-col items-start gap-4 sm:flex-row sm:items-end sm:justify-between sm:gap-6">
        <div class="min-w-0 max-w-full">
          <h1 class="break-words text-3xl font-bold tracking-tight sm:truncate sm:text-4xl">{{ library?.name ?? 'Bibliothèque' }}</h1>
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
        <div class="flex items-center gap-3"><p v-if="mediaQuery.data.value" class="text-xs text-muted-foreground">{{ mediaQuery.data.value.length }} vidéo{{ mediaQuery.data.value.length > 1 ? 's' : '' }}</p><Button v-if="mediaQuery.data.value?.length" size="sm" :variant="selectionMode ? 'secondary' : 'ghost'" @click="selectionMode ? closeSelection() : selectionMode = true"><X v-if="selectionMode" /><ListChecks v-else />{{ selectionMode ? 'Annuler' : 'Sélectionner' }}</Button></div>
      </div>

      <div v-if="selectionMode" class="mt-5 flex flex-col gap-3 rounded-2xl border border-primary/20 bg-primary/5 p-3 lg:flex-row lg:items-center">
        <p class="shrink-0 text-sm font-medium">{{ selectedIDs.length }} sélectionnée{{ selectedIDs.length > 1 ? 's' : '' }}</p>
        <div class="flex flex-wrap gap-2 lg:ml-auto"><Button size="sm" variant="secondary" :disabled="!selectedIDs.length || isBulkUpdating" @click="runBulk('favorite')"><Heart />Favoris</Button><Button size="sm" variant="secondary" :disabled="!selectedIDs.length || isBulkUpdating" @click="runBulk('unfavorite')"><HeartOff />Retirer</Button><Button size="sm" variant="secondary" :disabled="!selectedIDs.length || isBulkUpdating" @click="runBulk('watched')"><CheckCheck />Vues</Button><Button size="sm" variant="secondary" :disabled="!selectedIDs.length || isBulkUpdating" @click="runBulk('unwatched')"><EyeOff />Non vues</Button></div>
        <div v-if="collectionsQuery.data.value?.length" class="flex gap-2"><Select v-model="selectedCollectionID"><SelectTrigger class="h-9 min-w-40"><SelectValue placeholder="Collection" /></SelectTrigger><SelectContent><SelectItem v-for="item in collectionsQuery.data.value" :key="item.id" :value="item.id">{{ item.name }}</SelectItem></SelectContent></Select><Button size="sm" :disabled="!selectedIDs.length || !selectedCollectionID || isBulkUpdating" @click="runBulk('collection')">Ajouter</Button></div>
      </div>

      <div class="mt-5 flex flex-col gap-3 sm:flex-row">
        <label class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="search" type="search" placeholder="Rechercher une vidéo…" class="h-11 rounded-xl border-white/10 bg-white/5 pl-10 pr-4 shadow-none placeholder:text-muted-foreground/60 focus-visible:ring-primary/15" />
        </label>
        <Select v-model="filter">
          <SelectTrigger class="h-11 w-full rounded-xl border-white/10 bg-white/5 shadow-none sm:w-auto sm:min-w-36" aria-label="Filtrer les vidéos">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Toutes</SelectItem>
            <SelectItem value="favorite">Favoris</SelectItem>
            <SelectItem value="unwatched">Non vues</SelectItem>
            <SelectItem value="in-progress">En cours</SelectItem>
            <SelectItem value="watched">Vues</SelectItem>
          </SelectContent>
        </Select>
        <Select v-if="availableTags.length" v-model="tagFilter">
          <SelectTrigger class="h-11 w-full rounded-xl border-white/10 bg-white/5 shadow-none sm:w-auto sm:min-w-40" aria-label="Filtrer par tag">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Tous les tags</SelectItem>
            <SelectItem v-for="tag in availableTags" :key="tag.id" :value="tag.id">{{ tag.name }}</SelectItem>
          </SelectContent>
        </Select>
        <Select v-if="availableFolders.some((folder) => folder !== '')" v-model="folderFilter">
          <SelectTrigger class="h-11 w-full rounded-xl border-white/10 bg-white/5 shadow-none sm:w-auto sm:min-w-40" aria-label="Filtrer par dossier"><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="all">Tous les dossiers</SelectItem><SelectItem v-for="folder in availableFolders" :key="folder || 'root'" :value="folder || 'root'">{{ folder || 'Racine' }}</SelectItem></SelectContent>
        </Select>
        <Select v-model="sort">
          <SelectTrigger class="h-11 w-full rounded-xl border-white/10 bg-white/5 shadow-none sm:w-auto sm:min-w-40" aria-label="Trier les vidéos">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="name">Nom</SelectItem>
            <SelectItem value="recent">Plus récentes</SelectItem>
            <SelectItem value="recorded">Date d’enregistrement</SelectItem>
            <SelectItem value="duration">Durée</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="mediaQuery.isPending.value" class="grid min-h-72 place-items-center"><LoaderCircle class="size-6 animate-spin text-primary" /></div>
      <Empty v-else-if="mediaQuery.isError.value" class="mt-5 min-h-64 border border-red-400/15 bg-red-400/2.5">
        <EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Impossible de charger les vidéos</EmptyTitle><EmptyDescription>Une erreur est survenue pendant le chargement de cette bibliothèque.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" @click="mediaQuery.refetch()">Réessayer</Button></EmptyContent>
      </Empty>
      <Empty v-else-if="mediaQuery.data.value?.length === 0" class="mt-5 min-h-64 border border-white/10 bg-white/1.5">
        <EmptyHeader><EmptyMedia variant="icon"><Video /></EmptyMedia><EmptyTitle>Aucune vidéo indexée</EmptyTitle><EmptyDescription>Les nouvelles vidéos apparaîtront automatiquement dans cette bibliothèque.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" :disabled="isScanning" @click="scanMutation.mutate()"><RefreshCw :class="isScanning && 'animate-spin'" />{{ isScanning ? 'Analyse en cours…' : 'Analyser maintenant' }}</Button></EmptyContent>
      </Empty>
      <div v-else-if="filteredMedia.length" class="mt-5 grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
        <MediaCard v-for="item in filteredMedia" :key="item.id" :item="item" :selectable="selectionMode" :selected="selectedIDs.includes(item.id)" @select="toggleSelection(item.id, $event)" />
      </div>
      <Empty v-else class="mt-5 min-h-52 border border-white/10">
        <EmptyHeader><EmptyMedia variant="icon"><Search /></EmptyMedia><EmptyTitle>Aucune vidéo ne correspond</EmptyTitle><EmptyDescription>Modifiez votre recherche ou vos filtres.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="ghost" @click="search = ''; filter = 'all'; tagFilter = 'all'; folderFilter = 'all'">Réinitialiser les filtres</Button></EmptyContent>
      </Empty>
    </div>
  </section>
</template>
