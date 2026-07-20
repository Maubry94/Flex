<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { Check, CircleCheck, Film, Heart, Pencil, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'
import { toast } from 'vue-sonner'

import VideoMetadataDialog from '@/components/media/VideoMetadataDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuSeparator, ContextMenuTrigger } from '@/components/ui/context-menu'
import { Progress } from '@/components/ui/progress'
import type { MediaFile } from '@/lib/api/media'
import { thumbnailURL, updateMedia } from '@/lib/api/media'
import { saveProgress } from '@/lib/api/progress'
import { getTagAssignments } from '@/lib/api/tags'

const props = withDefaults(defineProps<{ item: MediaFile, selectable?: boolean, selected?: boolean, removableFromCollection?: boolean }>(), { selectable: false, selected: false, removableFromCollection: false })
const emit = defineEmits<{ select: [selected: boolean], removeFromCollection: [] }>()
const queryClient = useQueryClient()
const isMetadataDialogOpen = ref(false)
const assignmentsQuery = useQuery({ queryKey: ['tag-assignments'], queryFn: ({ signal }) => getTagAssignments(signal) })
const itemTags = computed(() => (assignmentsQuery.data.value ?? [])
  .filter((assignment) => assignment.mediaId === props.item.id)
  .map((assignment) => assignment.tag))

const favoriteMutation = useMutation({
  mutationFn: () => updateMedia(props.item.id, {
    title: props.item.title,
    description: props.item.description,
    recordedAt: props.item.recordedAt?.slice(0, 10) ?? null,
    favorite: !props.item.favorite,
  }),
  onSuccess: (item) => {
    applyUpdatedMedia(item)
    toast.success(item.favorite ? 'Ajoutée aux favoris' : 'Retirée des favoris')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier les favoris'),
})

const watchedMutation = useMutation({
  mutationFn: () => saveProgress(props.item.id, {
    positionMs: props.item.completed ? 0 : props.item.durationMs,
    durationMs: props.item.durationMs,
  }),
  onSuccess: (progress) => {
    queryClient.setQueryData<MediaFile>(['media-detail', props.item.id], (item) => item
      ? { ...item, progressMs: progress.positionMs, completed: progress.completed }
      : item)
    invalidateMediaQueries()
    toast.success(progress.completed ? 'Vidéo marquée comme vue' : 'Vidéo marquée comme non vue')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier le statut'),
})

function applyUpdatedMedia(item: MediaFile): void {
  queryClient.setQueryData(['media-detail', item.id], item)
  invalidateMediaQueries()
}

function invalidateMediaQueries(): void {
  void queryClient.invalidateQueries({ queryKey: ['media'] })
  void queryClient.invalidateQueries({ queryKey: ['home'] })
  void queryClient.invalidateQueries({ queryKey: ['favorites'] })
  void queryClient.invalidateQueries({ queryKey: ['global-search'] })
}

function formatDuration(durationMs: number): string {
	const totalSeconds = Math.floor(durationMs / 1_000)
	const hours = Math.floor(totalSeconds / 3_600)
	const minutes = Math.floor((totalSeconds % 3_600) / 60)
	const seconds = totalSeconds % 60
	const time = [minutes, seconds].map((value) => String(value).padStart(2, '0')).join(':')
	return hours > 0 ? `${String(hours).padStart(2, '0')}:${time}` : time
}

function progressPercent(positionMs: number, durationMs: number): number {
  if (durationMs <= 0) return 0
  return Math.min(100, Math.max(0, (positionMs / durationMs) * 100))
}

function hideBrokenImage(event: Event): void {
  if (event.currentTarget instanceof HTMLImageElement) event.currentTarget.hidden = true
}

function handleCardClick(event: MouseEvent): void {
  if (!props.selectable) return
  event.preventDefault()
  emit('select', !props.selected)
}
</script>

<template>
  <ContextMenu>
    <ContextMenuTrigger as-child>
      <RouterLink :to="{ name: 'video', params: { mediaId: item.id } }" class="group block" @click.capture="handleCardClick">
        <Card :class="['gap-0 overflow-hidden rounded-2xl bg-card/70 py-0 shadow-none transition group-hover:-translate-y-0.5', selected ? 'border-primary ring-2 ring-primary/25' : 'border-white/8 group-hover:border-white/15']">
          <div class="relative grid aspect-video place-items-center overflow-hidden bg-gradient-to-br from-zinc-800 to-zinc-950">
            <Film class="size-9 text-zinc-600" />
            <img :src="thumbnailURL(item.id)" :alt="`Miniature de ${item.title}`" class="absolute inset-0 size-full object-cover transition duration-300 group-hover:scale-[1.02]" loading="lazy" @error="hideBrokenImage" />
            <Badge v-if="item.completed" variant="secondary" class="absolute right-2 top-2 grid size-7 place-items-center rounded-full border-white/10 bg-black/75 p-0 text-white backdrop-blur" aria-label="Vue">
              <Check class="size-3.5 stroke-[3]" />
            </Badge>
            <span v-if="item.favorite" :class="['absolute top-2 grid size-7 place-items-center rounded-full bg-black/75 text-primary backdrop-blur', selectable ? 'left-11' : 'left-2']" aria-label="Favori"><Heart class="size-3.5 fill-current" /></span>
            <span v-if="selectable" class="absolute left-2 top-2 grid size-7 place-items-center rounded-full bg-black/75 backdrop-blur" @click.stop.prevent>
              <Checkbox :model-value="selected" :aria-label="selected ? 'Désélectionner la vidéo' : 'Sélectionner la vidéo'" @update:model-value="emit('select', $event === true)" />
            </span>
            <Progress v-if="item.progressMs > 0 && !item.completed" :model-value="progressPercent(item.progressMs, item.durationMs)" class="absolute inset-x-2 bottom-2 h-1 w-auto bg-black/50 [&_[data-slot=progress-indicator]]:bg-primary" />
          </div>
          <CardContent class="p-4">
            <h3 class="truncate text-sm font-semibold" :title="item.title">{{ item.title }}</h3>
            <p class="mt-2 text-xs text-muted-foreground">{{ item.width }}×{{ item.height }} · {{ formatDuration(item.durationMs) }}</p>
            <div v-if="itemTags.length" class="mt-3 flex min-w-0 gap-1.5 overflow-hidden">
              <Badge v-for="tag in itemTags.slice(0, 3)" :key="tag.id" variant="outline" class="min-w-0" :style="{ borderColor: tag.color, backgroundColor: `${tag.color}1f` }">
                <span class="truncate">{{ tag.name }}</span>
              </Badge>
              <Badge v-if="itemTags.length > 3" variant="secondary">+{{ itemTags.length - 3 }}</Badge>
            </div>
          </CardContent>
        </Card>
      </RouterLink>
    </ContextMenuTrigger>
    <ContextMenuContent class="w-56">
      <ContextMenuItem @select="isMetadataDialogOpen = true">
        <Pencil />
        Modifier les informations
      </ContextMenuItem>
      <ContextMenuSeparator />
      <ContextMenuItem :disabled="favoriteMutation.isPending.value" @select="favoriteMutation.mutate()">
        <Heart :class="item.favorite && 'fill-current text-primary'" />
        {{ item.favorite ? 'Retirer des favoris' : 'Ajouter aux favoris' }}
      </ContextMenuItem>
      <ContextMenuItem :disabled="watchedMutation.isPending.value" @select="watchedMutation.mutate()">
        <CircleCheck :class="item.completed && 'text-emerald-400'" />
        {{ item.completed ? 'Marquer comme non vue' : 'Marquer comme vue' }}
      </ContextMenuItem>
      <template v-if="removableFromCollection">
        <ContextMenuSeparator />
        <ContextMenuItem variant="destructive" @select="emit('removeFromCollection')">
          <Trash2 />
          Retirer de la collection
        </ContextMenuItem>
      </template>
    </ContextMenuContent>
  </ContextMenu>
  <VideoMetadataDialog v-model:open="isMetadataDialogOpen" :item="item" @saved="applyUpdatedMedia" />
</template>
