<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { CircleAlert, FolderOpen, LoaderCircle } from '@lucide/vue'

import MediaCard from '@/components/media/MediaCard.vue'
import { Button } from '@/components/ui/button'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { getHomeMedia } from '@/lib/api/media'

const homeQuery = useQuery({ queryKey: ['home'], queryFn: ({ signal }) => getHomeMedia(signal) })
</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(ellipse_at_top,rgba(124,58,237,0.10),transparent_68%)]" />
    <div class="relative mx-auto max-w-[1600px] px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Accueil</h1>
      <p class="mt-2 text-sm text-muted-foreground">Reprenez votre lecture ou découvrez les dernières vidéos ajoutées.</p>

      <div v-if="homeQuery.isPending.value" class="grid min-h-[520px] place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
      <Empty v-else-if="homeQuery.isError.value" class="mt-10 min-h-80 border border-red-400/15 bg-red-400/[0.025]">
        <EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Impossible de charger l'accueil</EmptyTitle><EmptyDescription>Une erreur est survenue pendant le chargement des vidéos.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" @click="homeQuery.refetch()">Réessayer</Button></EmptyContent>
      </Empty>
      <Empty v-else-if="homeQuery.data.value?.recentlyAdded.length === 0" class="mt-10 min-h-[420px] border border-white/10 bg-white/[0.015]">
        <EmptyHeader><EmptyMedia variant="icon"><FolderOpen /></EmptyMedia><EmptyTitle>Aucune vidéo disponible</EmptyTitle><EmptyDescription>Ajoutez une bibliothèque pour commencer à regarder vos vidéos.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" @click="$router.push({ name: 'libraries' })">Voir les bibliothèques</Button></EmptyContent>
      </Empty>
      <div v-else class="mt-10 space-y-12">
        <section v-if="homeQuery.data.value?.continueWatching.length">
          <div class="mb-5 flex items-center justify-between"><h2 class="text-xl font-semibold tracking-tight">Continuer à regarder</h2><span class="text-xs text-muted-foreground">{{ homeQuery.data.value.continueWatching.length }}</span></div>
          <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
            <MediaCard v-for="item in homeQuery.data.value.continueWatching" :key="item.id" :item="item" />
          </div>
        </section>
        <section>
          <div class="mb-5 flex items-center justify-between"><h2 class="text-xl font-semibold tracking-tight">Ajouts récents</h2><RouterLink :to="{ name: 'libraries' }" class="text-xs font-medium text-primary hover:text-primary/80">Voir les bibliothèques</RouterLink></div>
          <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
            <MediaCard v-for="item in homeQuery.data.value?.recentlyAdded" :key="item.id" :item="item" />
          </div>
        </section>
      </div>
    </div>
  </section>
</template>
