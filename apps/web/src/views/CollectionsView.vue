<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { Layers3, LoaderCircle, Plus } from '@lucide/vue'
import { ref } from 'vue'
import { toast } from 'vue-sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { createCollection, getCollections } from '@/lib/api/collections'

const queryClient = useQueryClient()
const open = ref(false)
const name = ref('')
const collectionsQuery = useQuery({ queryKey: ['collections'], queryFn: ({ signal }) => getCollections(signal) })
const createMutation = useMutation({
  mutationFn: () => createCollection(name.value),
  onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['collections'] }); open.value = false; name.value = ''; toast.success('Collection créée') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de créer la collection'),
})
</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="relative mx-auto max-w-[1600px] px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <div class="flex items-end justify-between gap-5"><div><h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Collections</h1><p class="mt-2 text-sm text-muted-foreground">Regroupez librement vos vidéos personnelles.</p></div><Button v-if="collectionsQuery.data.value?.length" @click="open = true"><Plus />Ajouter</Button></div>
      <div v-if="collectionsQuery.isPending.value" class="grid min-h-96 place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
      <Empty v-else-if="collectionsQuery.data.value?.length === 0" class="mt-10 min-h-105 border border-white/10"><EmptyHeader><EmptyMedia variant="icon"><Layers3 /></EmptyMedia><EmptyTitle>Aucune collection</EmptyTitle><EmptyDescription>Créez une collection pour regrouper des vidéos.</EmptyDescription></EmptyHeader><EmptyContent><Button @click="open = true"><Plus />Créer une collection</Button></EmptyContent></Empty>
      <div v-else class="mt-10 grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><RouterLink v-for="item in collectionsQuery.data.value" :key="item.id" :to="{ name: 'collection', params: { collectionId: item.id } }"><Card class="gap-0 rounded-2xl border-white/8 bg-card/70 py-0 transition hover:border-primary/35"><CardContent class="flex items-center gap-4 p-5"><span class="grid size-12 place-items-center rounded-2xl bg-primary/12 text-primary"><Layers3 /></span><div><h2 class="font-semibold">{{ item.name }}</h2><p class="mt-1 text-xs text-muted-foreground">{{ item.mediaCount }} vidéo{{ item.mediaCount > 1 ? 's' : '' }}</p></div></CardContent></Card></RouterLink></div>
    </div>
    <Dialog v-model:open="open"><DialogContent class="border-white/10 bg-background sm:max-w-md"><DialogHeader><DialogTitle>Créer une collection</DialogTitle><DialogDescription>Vous pourrez ensuite y ajouter des vidéos.</DialogDescription></DialogHeader><form class="space-y-5" @submit.prevent="createMutation.mutate()"><Input v-model="name" maxlength="100" autofocus placeholder="Nom de la collection" /><div class="flex justify-end gap-3"><Button type="button" variant="ghost" @click="open = false">Annuler</Button><Button type="submit" :disabled="!name.trim() || createMutation.isPending.value"><LoaderCircle v-if="createMutation.isPending.value" class="animate-spin" />Créer</Button></div></form></DialogContent></Dialog>
  </section>
</template>
