<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { CircleAlert, Heart, LoaderCircle } from '@lucide/vue'

import MediaCard from '@/components/media/MediaCard.vue'
import { Button } from '@/components/ui/button'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { getFavorites } from '@/lib/api/media'

const favoritesQuery = useQuery({
  queryKey: ['favorites'],
  queryFn: ({ signal }) => getFavorites(signal),
})
</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(ellipse_at_top,rgba(124,58,237,0.10),transparent_68%)]" />
    <div class="relative mx-auto max-w-[1600px] px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <div class="flex flex-col items-start gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-6">
        <div>
          <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Favoris</h1>
          <p class="mt-2 text-sm text-muted-foreground">Retrouvez les vidéos que vous souhaitez garder à portée de main.</p>
        </div>
        <p v-if="favoritesQuery.data.value" class="text-xs text-muted-foreground">{{ favoritesQuery.data.value.length }} vidéo{{ favoritesQuery.data.value.length > 1 ? 's' : '' }}</p>
      </div>

      <div v-if="favoritesQuery.isPending.value" class="grid min-h-[520px] place-items-center">
        <LoaderCircle class="size-7 animate-spin text-primary" />
      </div>
      <Empty v-else-if="favoritesQuery.isError.value" class="mt-10 min-h-80 border border-red-400/15 bg-red-400/[0.025]">
        <EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Impossible de charger les favoris</EmptyTitle><EmptyDescription>Une erreur est survenue pendant le chargement.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" @click="favoritesQuery.refetch()">Réessayer</Button></EmptyContent>
      </Empty>
      <Empty v-else-if="favoritesQuery.data.value?.length === 0" class="mt-10 min-h-[420px] border border-white/10 bg-white/[0.015]">
        <EmptyHeader><EmptyMedia variant="icon"><Heart /></EmptyMedia><EmptyTitle>Aucune vidéo favorite</EmptyTitle><EmptyDescription>Ajoutez une vidéo aux favoris depuis sa fiche pour la retrouver ici.</EmptyDescription></EmptyHeader>
        <EmptyContent><Button variant="secondary" @click="$router.push({ name: 'libraries' })">Voir les bibliothèques</Button></EmptyContent>
      </Empty>
      <div v-else class="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
        <MediaCard v-for="item in favoritesQuery.data.value" :key="item.id" :item="item" />
      </div>
    </div>
  </section>
</template>
