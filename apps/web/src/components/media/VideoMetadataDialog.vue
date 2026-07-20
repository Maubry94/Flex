<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { LoaderCircle, Plus } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import type { MediaFile } from '@/lib/api/media'
import { updateMedia } from '@/lib/api/media'
import { getCollections, getMediaCollections, setMediaCollections } from '@/lib/api/collections'
import type { Tag } from '@/lib/api/tags'
import { createTag, getMediaTags, getTags, setMediaTags } from '@/lib/api/tags'
import { asForwardedProps } from '@/lib/utils'

const props = defineProps<{ item: MediaFile }>()
const emit = defineEmits<{ saved: [item: MediaFile] }>()
const open = defineModel<boolean>('open', { required: true })
const queryClient = useQueryClient()
const isSaving = ref(false)
const isCreatingTag = ref(false)
const serverError = ref('')
const tagError = ref('')
const newTagName = ref('')
const newTagColor = ref('#7c3aed')
const selectedTagIDs = ref<string[]>([])
const selectedCollectionIDs = ref<string[]>([])
const collectionsQuery = useQuery({ queryKey: ['collections'], queryFn: ({ signal }) => getCollections(signal), enabled: computed(() => open.value) })
const mediaCollectionsQuery = useQuery({ queryKey: computed(() => ['media-collections', props.item.id]), queryFn: ({ signal }) => getMediaCollections(props.item.id, signal), enabled: computed(() => open.value) })

const tagsQuery = useQuery({
  queryKey: ['tags'],
  queryFn: ({ signal }) => getTags(signal),
  enabled: computed(() => open.value),
})
const mediaTagsQuery = useQuery({
  queryKey: computed(() => ['media-tags', props.item.id]),
  queryFn: ({ signal }) => getMediaTags(props.item.id, signal),
  enabled: computed(() => open.value),
})

const form = useForm({
  validationSchema: toTypedSchema(z.object({
    title: z.string().trim().min(1, 'Le titre est requis.').max(200, 'Le titre est trop long.'),
    description: z.string().trim().max(5_000, 'La description est trop longue.'),
    recordedAt: z.string().regex(/^$|^\d{4}-\d{2}-\d{2}$/u, "La date n'est pas valide."),
  })),
})

watch(open, (isOpen) => {
  if (!isOpen) return
  serverError.value = ''
  tagError.value = ''
  newTagName.value = ''
  selectedTagIDs.value = (mediaTagsQuery.data.value ?? []).map((tag) => tag.id)
  selectedCollectionIDs.value = (mediaCollectionsQuery.data.value ?? []).map((item) => item.id)
  form.resetForm({ values: {
    title: props.item.title,
    description: props.item.description,
    recordedAt: props.item.recordedAt?.slice(0, 10) ?? '',
  } })
})

watch(() => mediaTagsQuery.data.value, (tags) => {
  if (open.value && tags) selectedTagIDs.value = tags.map((tag) => tag.id)
}, { immediate: true })
watch(() => mediaCollectionsQuery.data.value, (items) => { if (open.value && items) selectedCollectionIDs.value = items.map((item) => item.id) }, { immediate: true })

function toggleTag(tagID: string): void {
  if (!selectedTagIDs.value.includes(tagID) && selectedTagIDs.value.length >= 20) {
    tagError.value = 'Une vidéo peut contenir au maximum 20 tags.'
    return
  }
  tagError.value = ''
  selectedTagIDs.value = selectedTagIDs.value.includes(tagID)
    ? selectedTagIDs.value.filter((id) => id !== tagID)
    : [...selectedTagIDs.value, tagID]
}

async function handleCreateTag(notify = true): Promise<boolean> {
  const name = newTagName.value.trim()
  if (!name) return true
  if (isCreatingTag.value) return false
  const existing = tagsQuery.data.value?.find((tag) => tag.name.localeCompare(name, 'fr', { sensitivity: 'base' }) === 0)
  if (existing) {
    if (!selectedTagIDs.value.includes(existing.id)) toggleTag(existing.id)
    newTagName.value = ''
    return true
  }
  if (selectedTagIDs.value.length >= 20) {
    tagError.value = 'Une vidéo peut contenir au maximum 20 tags.'
    return false
  }
  isCreatingTag.value = true
  tagError.value = ''
  try {
    const created = await createTag({ name, color: newTagColor.value })
    queryClient.setQueryData<Tag[]>(['tags'], (items) => [...(items ?? []), created].sort((first, second) => first.name.localeCompare(second.name, 'fr')))
    selectedTagIDs.value = [...selectedTagIDs.value, created.id]
    newTagName.value = ''
    if (notify) toast.success('Tag créé')
    return true
  } catch (error) {
    tagError.value = error instanceof Error ? error.message : 'Impossible de créer le tag'
    return false
  } finally {
    isCreatingTag.value = false
  }
}

const submit = form.handleSubmit(async (values) => {
  if (mediaTagsQuery.isPending.value || mediaCollectionsQuery.isPending.value) {
    serverError.value = 'Chargement des associations en cours.'
    return
  }
  if (mediaTagsQuery.isError.value || mediaCollectionsQuery.isError.value) {
    serverError.value = 'Impossible de charger les associations actuelles de la vidéo.'
    return
  }
  if (!(await handleCreateTag(false))) return
  isSaving.value = true
  serverError.value = ''
  try {
    const [updated, assignedTags, assignedCollections] = await Promise.all([
      updateMedia(props.item.id, {
        title: values.title,
        description: values.description,
        recordedAt: values.recordedAt || null,
        favorite: props.item.favorite,
      }),
      setMediaTags(props.item.id, selectedTagIDs.value),
      setMediaCollections(props.item.id, selectedCollectionIDs.value),
    ])
    queryClient.setQueryData(['media-tags', props.item.id], assignedTags)
    queryClient.setQueryData(['media-collections', props.item.id], assignedCollections)
    void queryClient.invalidateQueries({ queryKey: ['collections'] })
    void queryClient.invalidateQueries({ queryKey: ['collection-media'] })
    void queryClient.invalidateQueries({ queryKey: ['media-tags', props.item.id] })
    void queryClient.invalidateQueries({ queryKey: ['tag-assignments'] })
    emit('saved', updated)
    open.value = false
    toast.success('Informations enregistrées')
  } catch (error) {
    serverError.value = error instanceof Error ? error.message : 'Impossible de modifier la vidéo'
  } finally {
    isSaving.value = false
  }
})
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto border-white/10 bg-background sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>Modifier les informations</DialogTitle>
        <DialogDescription>Ces changements sont enregistrés dans Flex et ne renomment pas le fichier.</DialogDescription>
      </DialogHeader>
      <form class="space-y-5" novalidate @submit="submit">
        <FormField v-slot="{ componentField }" name="title">
          <FormItem><FormLabel>Titre</FormLabel><FormControl><Input v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="description">
          <FormItem><FormLabel>Description</FormLabel><FormControl><Textarea class="min-h-28 resize-y" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="recordedAt">
          <FormItem><FormLabel>Date d'enregistrement</FormLabel><FormControl><Input type="date" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
        </FormField>
        <div class="space-y-3">
          <div>
            <p class="text-sm font-medium">Tags</p>
            <p class="mt-1 text-xs text-muted-foreground">Sélectionnez jusqu’à 20 tags pour organiser cette vidéo.</p>
          </div>
          <div v-if="tagsQuery.isPending.value || mediaTagsQuery.isPending.value" class="flex h-10 items-center"><LoaderCircle class="size-4 animate-spin text-muted-foreground" /></div>
          <p v-else-if="tagsQuery.isError.value || mediaTagsQuery.isError.value" class="text-sm text-red-300">Impossible de charger les tags.</p>
          <div v-else-if="tagsQuery.data.value?.length" class="flex flex-wrap gap-2">
            <button v-for="tag in tagsQuery.data.value" :key="tag.id" type="button" :aria-pressed="selectedTagIDs.includes(tag.id)" @click="toggleTag(tag.id)">
              <Badge
                variant="outline"
                :class="selectedTagIDs.includes(tag.id) ? 'text-foreground' : 'opacity-55'"
                :style="{ borderColor: tag.color, backgroundColor: selectedTagIDs.includes(tag.id) ? `${tag.color}26` : 'transparent' }"
              >{{ tag.name }}</Badge>
            </button>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="newTagName" maxlength="50" placeholder="Nouveau tag" @keydown.enter.prevent="handleCreateTag" />
            <Input v-model="newTagColor" type="color" class="size-9 shrink-0 cursor-pointer p-1" aria-label="Couleur du tag" />
            <Button type="button" size="icon" variant="secondary" :disabled="!newTagName.trim() || isCreatingTag" aria-label="Créer le tag" @click="handleCreateTag()">
              <LoaderCircle v-if="isCreatingTag" class="animate-spin" /><Plus v-else />
            </Button>
          </div>
          <p v-if="tagError" class="text-sm text-red-300">{{ tagError }}</p>
        </div>
        <div v-if="collectionsQuery.data.value?.length" class="space-y-3">
          <div><p class="text-sm font-medium">Collections</p><p class="mt-1 text-xs text-muted-foreground">Ajoutez cette vidéo à une ou plusieurs collections.</p></div>
          <div class="flex flex-wrap gap-2"><button v-for="item in collectionsQuery.data.value" :key="item.id" type="button" @click="selectedCollectionIDs = selectedCollectionIDs.includes(item.id) ? selectedCollectionIDs.filter((id) => id !== item.id) : [...selectedCollectionIDs, item.id]"><Badge variant="outline" :class="selectedCollectionIDs.includes(item.id) ? 'border-primary bg-primary/15' : 'opacity-55'">{{ item.name }}</Badge></button></div>
        </div>
        <p v-if="serverError" class="text-sm text-red-300">{{ serverError }}</p>
        <div class="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button class="w-full sm:w-auto" type="button" variant="ghost" :disabled="isSaving" @click="open = false">Annuler</Button>
          <Button class="w-full sm:w-auto" type="submit" :disabled="isSaving || mediaTagsQuery.isPending.value || mediaCollectionsQuery.isPending.value"><LoaderCircle v-if="isSaving" class="animate-spin" />Enregistrer</Button>
        </div>
      </form>
    </DialogContent>
  </Dialog>
</template>
