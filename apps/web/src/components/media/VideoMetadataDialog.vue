<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { LoaderCircle } from '@lucide/vue'
import { ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import type { MediaFile } from '@/lib/api/media'
import { updateMedia } from '@/lib/api/media'
import { asForwardedProps } from '@/lib/utils'

const props = defineProps<{ item: MediaFile }>()
const emit = defineEmits<{ saved: [item: MediaFile] }>()
const open = defineModel<boolean>('open', { required: true })
const isSaving = ref(false)
const serverError = ref('')

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
  form.resetForm({ values: {
    title: props.item.title,
    description: props.item.description,
    recordedAt: props.item.recordedAt?.slice(0, 10) ?? '',
  } })
})

const submit = form.handleSubmit(async (values) => {
  isSaving.value = true
  serverError.value = ''
  try {
    const updated = await updateMedia(props.item.id, {
      title: values.title,
      description: values.description,
      recordedAt: values.recordedAt || null,
      favorite: props.item.favorite,
    })
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
        <p v-if="serverError" class="text-sm text-red-300">{{ serverError }}</p>
        <div class="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button class="w-full sm:w-auto" type="button" variant="ghost" :disabled="isSaving" @click="open = false">Annuler</Button>
          <Button class="w-full sm:w-auto" type="submit" :disabled="isSaving"><LoaderCircle v-if="isSaving" class="animate-spin" />Enregistrer</Button>
        </div>
      </form>
    </DialogContent>
  </Dialog>
</template>
