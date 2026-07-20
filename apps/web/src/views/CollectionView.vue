<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { ArrowLeft, Layers3, LoaderCircle, Pencil, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'

import MediaCard from '@/components/media/MediaCard.vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { deleteCollection, getCollectionMedia, getCollections, removeMediaFromCollection, updateCollection } from '@/lib/api/collections'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const id = computed(() => String(route.params.collectionId))
const isEditOpen = ref(false)
const isDeleteOpen = ref(false)
const name = ref('')

const collectionsQuery = useQuery({ queryKey: ['collections'], queryFn: ({ signal }) => getCollections(signal) })
const collection = computed(() => collectionsQuery.data.value?.find((item) => item.id === id.value))
const mediaQuery = useQuery({ queryKey: computed(() => ['collection-media', id.value]), queryFn: ({ signal }) => getCollectionMedia(id.value, signal) })

const updateMutation = useMutation({
  mutationFn: () => updateCollection(id.value, name.value),
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['collections'] })
    isEditOpen.value = false
    toast.success('Collection modifiée')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier la collection'),
})

const deleteMutation = useMutation({
  mutationFn: () => deleteCollection(id.value),
  onSuccess: async () => {
    queryClient.removeQueries({ queryKey: ['collection-media', id.value] })
    await queryClient.invalidateQueries({ queryKey: ['collections'] })
    toast.success('Collection supprimée')
    await router.push({ name: 'collections' })
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de supprimer la collection'),
})

const removeMutation = useMutation({
  mutationFn: (mediaId: string) => removeMediaFromCollection(id.value, mediaId),
  onSuccess: async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['collection-media', id.value] }),
      queryClient.invalidateQueries({ queryKey: ['collections'] }),
    ])
    toast.success('Vidéo retirée de la collection')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de retirer la vidéo'),
})

function openEditDialog(): void {
  name.value = collection.value?.name ?? ''
  isEditOpen.value = true
}
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <RouterLink :to="{ name: 'collections' }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground">
        <ArrowLeft class="size-4" />Collections
      </RouterLink>

      <div class="mt-6 flex flex-wrap items-center justify-between gap-4">
        <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">{{ collection?.name ?? 'Collection' }}</h1>
        <div v-if="collection" class="flex items-center gap-2">
          <Button variant="outline" @click="openEditDialog"><Pencil />Modifier</Button>
          <Button variant="destructive" @click="isDeleteOpen = true"><Trash2 />Supprimer</Button>
        </div>
      </div>

      <div v-if="mediaQuery.isPending.value" class="grid min-h-96 place-items-center"><LoaderCircle class="animate-spin text-primary" /></div>
      <Empty v-else-if="mediaQuery.data.value?.length === 0" class="mt-8 min-h-80 border border-white/10">
        <EmptyHeader><EmptyMedia variant="icon"><Layers3 /></EmptyMedia><EmptyTitle>Collection vide</EmptyTitle><EmptyDescription>Ajoutez des vidéos depuis leur fenêtre de modification.</EmptyDescription></EmptyHeader>
      </Empty>
      <div v-else class="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
        <MediaCard
          v-for="item in mediaQuery.data.value"
          :key="item.id"
          :item="item"
          removable-from-collection
          @remove-from-collection="removeMutation.mutate(item.id)"
        />
      </div>
    </div>

    <Dialog v-model:open="isEditOpen">
      <DialogContent class="border-white/10 bg-background sm:max-w-md">
        <DialogHeader><DialogTitle>Modifier la collection</DialogTitle><DialogDescription>Choisissez un nouveau nom pour cette collection.</DialogDescription></DialogHeader>
        <form class="space-y-5" @submit.prevent="updateMutation.mutate()">
          <Input v-model="name" maxlength="100" autofocus placeholder="Nom de la collection" />
          <DialogFooter><Button type="button" variant="ghost" @click="isEditOpen = false">Annuler</Button><Button type="submit" :disabled="!name.trim() || updateMutation.isPending.value"><LoaderCircle v-if="updateMutation.isPending.value" class="animate-spin" />Enregistrer</Button></DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="isDeleteOpen">
      <DialogContent class="border-white/10 bg-background sm:max-w-md">
        <DialogHeader><DialogTitle>Supprimer cette collection ?</DialogTitle><DialogDescription>Les vidéos ne seront pas supprimées de votre bibliothèque.</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="ghost" @click="isDeleteOpen = false">Annuler</Button><Button variant="destructive" :disabled="deleteMutation.isPending.value" @click="deleteMutation.mutate()"><LoaderCircle v-if="deleteMutation.isPending.value" class="animate-spin" />Supprimer</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </section>
</template>
