<script setup lang="ts">
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { ArrowLeft, Film, LoaderCircle, Pause, Play, RotateCcw, RotateCw } from '@lucide/vue'
import type HlsInstance from 'hls.js'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import { getMediaByID, getPlayback, thumbnailURL } from '@/lib/api/media'
import type { PlaybackInfo } from '@/lib/api/media'
import { getProgress, saveProgress } from '@/lib/api/progress'
import { mediaTitle } from '@/lib/media-title'
import { Card, CardContent } from '@/components/ui/card'

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
  if (!video || !Number.isFinite(video.duration) || video.duration <= 0) return
  void saveProgress(mediaID.value, {
    positionMs: Math.round(video.currentTime * 1000),
    durationMs: Math.round(video.duration * 1000),
  }).then((progress) => {
    queryClient.setQueryData(['progress', mediaID.value], progress)
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
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div v-if="mediaQuery.isPending.value" class="grid min-h-[70dvh] place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
    <div v-else-if="mediaQuery.data.value" class="mx-auto max-w-[1500px] px-5 py-8 lg:px-10 lg:py-10">
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
          <h1 class="truncate text-2xl font-bold tracking-tight sm:text-3xl" :title="mediaTitle(mediaQuery.data.value.filename)">{{ mediaTitle(mediaQuery.data.value.filename) }}</h1>
          <p class="mt-3 text-sm text-muted-foreground">Vidéo personnelle</p>
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
    </div>
    <div v-else class="grid min-h-[70dvh] place-items-center text-center"><div><Film class="mx-auto size-8 text-muted-foreground" /><p class="mt-3 font-medium">Vidéo introuvable</p></div></div>
  </section>
</template>
