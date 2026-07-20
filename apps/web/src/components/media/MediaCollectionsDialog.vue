<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { CircleAlert, Layers3, LoaderCircle } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { toast } from 'vue-sonner'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { getCollections, getMediaCollections, setMediaCollections } from '@/lib/api/collections'

const props = defineProps<{ mediaId: string }>()
const open = defineModel<boolean>('open', { required: true })
const queryClient = useQueryClient()
const selectedIDs = ref<string[]>([])

const collectionsQuery = useQuery({
  queryKey: ['collections'],
  queryFn: ({ signal }) => getCollections(signal),
  enabled: computed(() => open.value),
})
const assignedQuery = useQuery({
  queryKey: computed(() => ['media-collections', props.mediaId]),
  queryFn: ({ signal }) => getMediaCollections(props.mediaId, signal),
  enabled: computed(() => open.value),
})

watch(() => assignedQuery.data.value, (items) => {
  if (open.value && items) selectedIDs.value = items.map((item) => item.id)
}, { immediate: true })

watch(open, (isOpen) => {
  if (isOpen) selectedIDs.value = (assignedQuery.data.value ?? []).map((item) => item.id)
})

function toggle(collectionID: string): void {
  selectedIDs.value = selectedIDs.value.includes(collectionID)
    ? selectedIDs.value.filter((id) => id !== collectionID)
    : [...selectedIDs.value, collectionID]
}

const saveMutation = useMutation({
  mutationFn: () => setMediaCollections(props.mediaId, selectedIDs.value),
  onSuccess: (items) => {
    queryClient.setQueryData(['media-collections', props.mediaId], items)
    void queryClient.invalidateQueries({ queryKey: ['collections'] })
    void queryClient.invalidateQueries({ queryKey: ['collection-media'] })
    open.value = false
    toast.success('Collections mises à jour')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier les collections'),
})
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="border-white/10 bg-background sm:max-w-md">
      <DialogHeader><DialogTitle>Gérer les collections</DialogTitle><DialogDescription>Choisissez les collections contenant cette vidéo.</DialogDescription></DialogHeader>
      <div v-if="collectionsQuery.isPending.value || assignedQuery.isPending.value" class="grid min-h-40 place-items-center"><LoaderCircle class="size-5 animate-spin text-primary" /></div>
      <Empty v-else-if="collectionsQuery.isError.value || assignedQuery.isError.value" class="min-h-48 border border-red-400/15"><EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Impossible de charger les collections</EmptyTitle><EmptyDescription>Fermez cette fenêtre puis réessayez.</EmptyDescription></EmptyHeader></Empty>
      <Empty v-else-if="collectionsQuery.data.value?.length === 0" class="min-h-48 border border-white/8"><EmptyHeader><EmptyMedia variant="icon"><Layers3 /></EmptyMedia><EmptyTitle>Aucune collection</EmptyTitle><EmptyDescription>Créez d’abord une collection depuis la page Collections.</EmptyDescription></EmptyHeader></Empty>
      <div v-else class="max-h-72 space-y-1 overflow-y-auto">
        <button v-for="collection in collectionsQuery.data.value" :key="collection.id" type="button" class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition hover:bg-white/5" @click="toggle(collection.id)">
          <Checkbox :model-value="selectedIDs.includes(collection.id)" tabindex="-1" />
          <span class="min-w-0 flex-1 truncate">{{ collection.name }}</span>
          <span class="text-xs text-muted-foreground">{{ collection.mediaCount }}</span>
        </button>
      </div>
      <DialogFooter><Button variant="ghost" :disabled="saveMutation.isPending.value" @click="open = false">Annuler</Button><Button :disabled="collectionsQuery.isPending.value || assignedQuery.isPending.value || collectionsQuery.isError.value || assignedQuery.isError.value || saveMutation.isPending.value" @click="saveMutation.mutate()"><LoaderCircle v-if="saveMutation.isPending.value" class="animate-spin" />Enregistrer</Button></DialogFooter>
    </DialogContent>
  </Dialog>
</template>
