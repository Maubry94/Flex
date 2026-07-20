<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { ArrowLeft, CircleCheck, Film, Heart, LoaderCircle, Pause, Pencil, Play, RotateCcw, RotateCw } from '@lucide/vue'
import type HlsInstance from 'hls.js'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { toast } from 'vue-sonner'

import VideoMetadataDialog from '@/components/media/VideoMetadataDialog.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { getMediaByID, getPlayback, thumbnailURL, updateMedia } from '@/lib/api/media'
import type { MediaFile, PlaybackInfo } from '@/lib/api/media'
import { getProgress, saveProgress } from '@/lib/api/progress'

const route = useRoute()
const queryClient = useQueryClient()
const mediaID = computed(() => String(route.params.mediaId))
const playbackError = ref(false)
const isPreparing = ref(true)
const videoElement = ref<HTMLVideoElement>()
const playerContainer = ref<HTMLElement>()
const resumeApplied = ref(false)
const isPlaying = ref(false)
const showTouchControls = ref(true)
const isMetadataDialogOpen = ref(false)
const hasManualProgressOverride = ref(false)
let lastPeriodicSave = 0
let hls: HlsInstance | undefined
let touchControlsTimer: ReturnType<typeof setTimeout> | undefined

const mediaQuery = useQuery({
  queryKey: computed(() => ['media-detail', mediaID.value]),
  queryFn: ({ signal }) => getMediaByID(mediaID.value, signal),
})
const playbackQuery = useQuery({
  queryKey: computed(() => ['playback', mediaID.value]),
  queryFn: ({ signal }) => getPlayback(mediaID.value, signal),
})
const progressQuery = useQuery({
  queryKey: computed(() => ['progress', mediaID.value]),
  queryFn: ({ signal }) => getProgress(mediaID.value, signal),
})
const favoriteMutation = useMutation({
  mutationFn: async () => {
    const item = mediaQuery.data.value
    if (!item) throw new Error('Vidéo indisponible')
    return updateMedia(item.id, {
      title: item.title,
      description: item.description,
      recordedAt: item.recordedAt?.slice(0, 10) ?? null,
      favorite: !item.favorite,
    })
  },
  onSuccess: (item) => {
    applyUpdatedMedia(item)
    toast.success(item.favorite ? 'Ajoutée aux favoris' : 'Retirée des favoris')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier les favoris'),
})
const watchedMutation = useMutation({
  mutationFn: async () => {
    const item = mediaQuery.data.value
    if (!item) throw new Error('Vidéo indisponible')
    return saveProgress(item.id, {
      positionMs: item.completed ? 0 : item.durationMs,
      durationMs: item.durationMs,
    })
  },
  onSuccess: (progress) => {
    hasManualProgressOverride.value = true
    queryClient.setQueryData(['progress', mediaID.value], progress)
    queryClient.setQueryData<MediaFile>(['media-detail', mediaID.value], (item) => item
      ? { ...item, progressMs: progress.positionMs, completed: progress.completed }
      : item)
    void queryClient.invalidateQueries({ queryKey: ['media'] })
    void queryClient.invalidateQueries({ queryKey: ['home'] })
    toast.success(progress.completed ? 'Vidéo marquée comme vue' : 'Vidéo marquée comme non vue')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier le statut'),
})

function applyUpdatedMedia(item: MediaFile): void {
  queryClient.setQueryData(['media-detail', mediaID.value], item)
  void queryClient.invalidateQueries({ queryKey: ['media'] })
  void queryClient.invalidateQueries({ queryKey: ['home'] })
  void queryClient.invalidateQueries({ queryKey: ['favorites'] })
  void queryClient.invalidateQueries({ queryKey: ['global-search'] })
}

watch(
  [videoElement, () => playbackQuery.data.value],
  async ([video, playback]) => {
    if (!video || !playback) return
		await preparePlayback(video, playback)
  },
  { flush: 'post' },
)

watch(mediaID, () => {
	resumeApplied.value = false
	hasManualProgressOverride.value = false
	lastPeriodicSave = 0
})

onBeforeUnmount(() => {
	hls?.destroy()
	if (touchControlsTimer) clearTimeout(touchControlsTimer)
})

watch([videoElement, () => progressQuery.data.value], applyResume, { flush: 'post' })

function applyResume(): void {
  const video = videoElement.value
  const progress = progressQuery.data.value
  if (!video || !progress || resumeApplied.value || video.readyState < 1) return
  resumeApplied.value = true
  if (!progress.completed && progress.positionMs >= 5_000 && progress.positionMs < progress.durationMs - 5_000) {
    video.currentTime = progress.positionMs / 1000
  }
}

function persistProgress(refreshLibrary: boolean): void {
  const video = videoElement.value
  const item = mediaQuery.data.value
  if (hasManualProgressOverride.value || !video || !item || !Number.isFinite(video.duration) || video.duration <= 0) return
  void saveProgress(item.id, {
    positionMs: Math.round(video.currentTime * 1000),
    durationMs: Math.round(video.duration * 1000),
  }).then((progress) => {
    queryClient.setQueryData(['progress', item.id], progress)
    if (refreshLibrary) void queryClient.invalidateQueries({ queryKey: ['media'] })
  }).catch(() => undefined)
}

function handleTimeUpdate(): void {
  const now = Date.now()
  if (now - lastPeriodicSave < 5_000) return
  lastPeriodicSave = now
  persistProgress(false)
}

function handleLoadedMetadata(): void {
  applyResume()
}

async function preparePlayback(video: HTMLVideoElement, playback: PlaybackInfo): Promise<void> {
	hls?.destroy()
	hls = undefined
	video.removeAttribute('src')
	video.load()
	playbackError.value = false
	isPreparing.value = true
	if (playback.mode === 'direct') {
		video.src = playback.url
		return
	}

	const { default: Hls } = await import('hls.js')
	if (video !== videoElement.value || playback !== playbackQuery.data.value) return
	if (Hls.isSupported()) {
		hls = new Hls({ manifestLoadingTimeOut: 120_000 })
		hls.on(Hls.Events.ERROR, (_event, data) => {
			if (data.fatal) {
				playbackError.value = true
				isPreparing.value = false
			}
		})
		hls.loadSource(playback.url)
		hls.attachMedia(video)
		return
	}
	if (video.canPlayType('application/vnd.apple.mpegurl')) {
		video.src = playback.url
		return
	}
	playbackError.value = true
	isPreparing.value = false
}

async function retryPlayback(): Promise<void> {
	const result = await playbackQuery.refetch()
	const video = videoElement.value
	if (video && result.data) await preparePlayback(video, result.data)
}

function seekBy(seconds: number): void {
	const video = videoElement.value
	if (!video || !Number.isFinite(video.duration)) return
	video.currentTime = Math.min(Math.max(video.currentTime + seconds, 0), video.duration)
	revealTouchControls()
}

function togglePlayback(): void {
	const video = videoElement.value
	if (!video) return
	if (video.paused) void video.play()
	else video.pause()
}

function revealTouchControls(): void {
	showTouchControls.value = true
	if (touchControlsTimer) clearTimeout(touchControlsTimer)
	if (isPlaying.value) {
		touchControlsTimer = setTimeout(() => {
			showTouchControls.value = false
		}, 2_500)
	}
}

function handlePlay(): void {
	hasManualProgressOverride.value = false
	isPlaying.value = true
	isPreparing.value = false
	revealTouchControls()
}

function handlePause(): void {
	isPlaying.value = false
	showTouchControls.value = true
	persistProgress(true)
}

function handlePlaybackError(): void {
	playbackError.value = true
	isPreparing.value = false
}

function handleEnded(): void {
	isPlaying.value = false
	showTouchControls.value = true
	persistProgress(true)
}

function handlePlayerKeydown(event: KeyboardEvent): void {
	if (!window.matchMedia('(pointer: fine)').matches) return
	const video = videoElement.value
	if (!video) return
	const key = event.key.toLowerCase()
	if (![' ', 'k', 'arrowleft', 'arrowright', 'm', 'f'].includes(key)) return
	event.preventDefault()
	event.stopPropagation()
	if (key === ' ' || key === 'k') {
		togglePlayback()
	} else if (key === 'arrowleft') {
		seekBy(-10)
	} else if (key === 'arrowright') {
		seekBy(10)
	} else if (key === 'm') {
		video.muted = !video.muted
	} else if (key === 'f') {
		if (document.fullscreenElement) void document.exitFullscreen()
		else void playerContainer.value?.requestFullscreen()
	}
}

onBeforeUnmount(() => {
  persistProgress(true)
})

function formatDuration(durationMs: number): string {
  const totalSeconds = Math.floor(durationMs / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return [hours, minutes, seconds]
    .filter((_, index) => hours > 0 || index > 0)
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}

function formatSize(sizeBytes: number): string {
  return new Intl.NumberFormat('fr-FR', { style: 'unit', unit: 'megabyte', maximumFractionDigits: 1 }).format(sizeBytes / 1_000_000)
}

function formatRecordedDate(value: string): string {
  return new Intl.DateTimeFormat('fr-FR', { dateStyle: 'long', timeZone: 'UTC' }).format(new Date(value))
}
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div v-if="mediaQuery.isPending.value" class="grid min-h-[70dvh] place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
    <div v-else-if="mediaQuery.data.value" class="mx-auto max-w-[1500px] px-4 py-6 sm:px-5 sm:py-8 lg:px-10 lg:py-10">
      <RouterLink :to="{ name: 'library', params: { libraryId: mediaQuery.data.value.libraryId } }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground">
        <ArrowLeft class="size-4" />
        Retour à la bibliothèque
      </RouterLink>

      <div ref="playerContainer" class="relative mt-6 overflow-hidden rounded-2xl border border-white/10 bg-black shadow-2xl shadow-black/30 outline-none focus-visible:ring-2 focus-visible:ring-primary/70" tabindex="0" @keydown.capture="handlePlayerKeydown" @pointerdown="revealTouchControls">
        <video
          ref="videoElement"
          class="aspect-video w-full bg-black"
          controls
          playsinline
          preload="metadata"
          :poster="thumbnailURL(mediaQuery.data.value.id)"
          @error="handlePlaybackError"
          @loadedmetadata="handleLoadedMetadata"
          @canplay="isPreparing = false"
          @waiting="isPreparing = true"
          @playing="handlePlay"
          @timeupdate="handleTimeUpdate"
          @pause="handlePause"
          @ended="handleEnded"
        />
        <div v-show="showTouchControls && !playbackError" class="absolute inset-x-0 bottom-12 top-0 hidden items-center justify-center gap-7 bg-black/15 [@media(pointer:coarse)]:flex">
          <button type="button" class="grid size-12 place-items-center rounded-full bg-black/65 text-white backdrop-blur-sm active:scale-95" aria-label="Reculer de 10 secondes" @click.stop="seekBy(-10)">
            <RotateCcw class="size-5" />
            <span class="absolute mt-0.5 text-[8px] font-bold">10</span>
          </button>
          <button type="button" class="grid size-16 place-items-center rounded-full bg-white/90 text-black shadow-xl active:scale-95" :aria-label="isPlaying ? 'Mettre en pause' : 'Lire'" @click.stop="togglePlayback">
            <Pause v-if="isPlaying" class="size-7 fill-current" />
            <Play v-else class="ml-1 size-7 fill-current" />
          </button>
          <button type="button" class="grid size-12 place-items-center rounded-full bg-black/65 text-white backdrop-blur-sm active:scale-95" aria-label="Avancer de 10 secondes" @click.stop="seekBy(10)">
            <RotateCw class="size-5" />
            <span class="absolute mt-0.5 text-[8px] font-bold">10</span>
          </button>
        </div>
        <div v-if="isPreparing && !playbackError" class="pointer-events-none absolute inset-0 grid place-items-center bg-black/35">
          <div class="rounded-2xl bg-black/65 px-5 py-4 text-center backdrop-blur-sm">
            <LoaderCircle class="mx-auto size-6 animate-spin text-primary" />
            <p class="mt-2 text-xs text-white/75">{{ playbackQuery.data.value?.mode === 'hls' ? 'Conversion en cours…' : 'Préparation…' }}</p>
          </div>
        </div>
      </div>

      <div v-if="playbackError" class="mt-4 flex flex-col items-start gap-3 rounded-xl border border-amber-400/15 bg-amber-400/8 px-4 py-3 text-sm text-amber-200 sm:flex-row sm:items-center sm:justify-between">
        <span>La lecture a échoué. Vérifiez la conversion de cette vidéo dans les journaux du serveur.</span>
        <button type="button" class="shrink-0 font-medium text-amber-100 underline underline-offset-4" @click="retryPlayback">Réessayer</button>
      </div>

      <div class="mt-8 grid gap-8 lg:grid-cols-[minmax(0,1fr)_auto]">
        <div class="min-w-0">
          <div class="flex flex-col items-start gap-3 sm:flex-row sm:justify-between sm:gap-4">
            <h1 class="break-words text-2xl font-bold tracking-tight sm:min-w-0 sm:truncate sm:text-3xl" :title="mediaQuery.data.value.title">{{ mediaQuery.data.value.title }}</h1>
            <div class="flex max-w-full shrink-0 flex-wrap gap-2">
              <Button variant="ghost" size="icon" :aria-label="mediaQuery.data.value.completed ? 'Marquer comme non vue' : 'Marquer comme vue'" :disabled="watchedMutation.isPending.value" @click="watchedMutation.mutate()">
                <CircleCheck :class="mediaQuery.data.value.completed && 'fill-emerald-400/15 text-emerald-400'" />
              </Button>
              <Button variant="ghost" size="icon" :aria-label="mediaQuery.data.value.favorite ? 'Retirer des favoris' : 'Ajouter aux favoris'" :disabled="favoriteMutation.isPending.value" @click="favoriteMutation.mutate()">
                <Heart :class="mediaQuery.data.value.favorite && 'fill-primary text-primary'" />
              </Button>
              <Button class="max-sm:flex-1" variant="secondary" @click="isMetadataDialogOpen = true"><Pencil />Modifier</Button>
            </div>
          </div>
          <p v-if="mediaQuery.data.value.description" class="mt-3 max-w-3xl whitespace-pre-line text-sm leading-6 text-muted-foreground">{{ mediaQuery.data.value.description }}</p>
          <p v-else class="mt-3 text-sm text-muted-foreground">Vidéo personnelle</p>
          <p v-if="mediaQuery.data.value.recordedAt" class="mt-2 text-xs text-muted-foreground">Enregistrée le {{ formatRecordedDate(mediaQuery.data.value.recordedAt) }}</p>
          <span v-if="playbackQuery.data.value" class="mt-4 inline-flex rounded-full border border-white/8 bg-white/5 px-2.5 py-1 text-[11px] font-medium text-muted-foreground">
            {{ playbackQuery.data.value.mode === 'direct' ? 'Lecture directe' : 'Conversion HLS' }}
          </span>
        </div>
        <Card class="gap-0 rounded-2xl border-white/8 bg-card/60 py-0 shadow-none">
          <CardContent class="p-5">
            <dl class="grid grid-cols-2 gap-x-8 gap-y-4 text-sm sm:grid-cols-3">
              <div><dt class="text-xs text-muted-foreground">Durée</dt><dd class="mt-1 font-medium">{{ formatDuration(mediaQuery.data.value.durationMs) }}</dd></div>
              <div><dt class="text-xs text-muted-foreground">Résolution</dt><dd class="mt-1 font-medium">{{ mediaQuery.data.value.width }}×{{ mediaQuery.data.value.height }}</dd></div>
              <div><dt class="text-xs text-muted-foreground">Taille</dt><dd class="mt-1 font-medium">{{ formatSize(mediaQuery.data.value.sizeBytes) }}</dd></div>
              <div><dt class="text-xs text-muted-foreground">Vidéo</dt><dd class="mt-1 font-medium uppercase">{{ mediaQuery.data.value.videoCodec }}</dd></div>
              <div><dt class="text-xs text-muted-foreground">Audio</dt><dd class="mt-1 font-medium uppercase">{{ mediaQuery.data.value.audioCodec || '—' }}</dd></div>
              <div><dt class="text-xs text-muted-foreground">Conteneur</dt><dd class="mt-1 max-w-28 truncate font-medium uppercase" :title="mediaQuery.data.value.container">{{ mediaQuery.data.value.container.split(',')[0] }}</dd></div>
            </dl>
          </CardContent>
        </Card>
      </div>
      <VideoMetadataDialog v-model:open="isMetadataDialogOpen" :item="mediaQuery.data.value" @saved="applyUpdatedMedia" />
    </div>
    <Empty v-else class="mx-auto min-h-[70dvh] max-w-[1500px] border-0">
      <EmptyHeader><EmptyMedia variant="icon"><Film /></EmptyMedia><EmptyTitle>Vidéo introuvable</EmptyTitle><EmptyDescription>Cette vidéo n’existe plus ou n’est plus accessible.</EmptyDescription></EmptyHeader>
    </Empty>
  </section>
</template>
