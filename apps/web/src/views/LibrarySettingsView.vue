<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { ArrowLeft, LoaderCircle, Trash2 } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { useRoute, useRouter } from 'vue-router'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { ApiError, deleteLibrary, getLibraries, updateLibrary } from '@/lib/api/libraries'
import { asForwardedProps } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const libraryID = computed(() => String(route.params.libraryId))
const successMessage = ref('')

const libraryFormSchema = toTypedSchema(z.object({
  name: z.string().trim().min(1, 'Le nom est requis.').max(100, 'Le nom est trop long.'),
  path: z.string().trim().min(1, 'Le dossier est requis.').max(4_096, 'Le chemin est trop long.'),
}))
const libraryForm = useForm({ validationSchema: libraryFormSchema })

const librariesQuery = useQuery({ queryKey: ['libraries'], queryFn: ({ signal }) => getLibraries(signal) })
const library = computed(() => librariesQuery.data.value?.find((item) => item.id === libraryID.value))

watch(library, (value) => {
  if (!value) return
  libraryForm.resetForm({ values: { name: value.name, path: value.path } })
}, { immediate: true })

const updateMutation = useMutation({
  mutationFn: (values: { name: string, path: string }) => updateLibrary(libraryID.value, values),
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['libraries'] })
    successMessage.value = 'Modifications enregistrées.'
  },
  onError: (error) => {
    if (!(error instanceof ApiError)) return
    if (error.code === 'invalid_name') libraryForm.setFieldError('name', error.message)
    if (error.code === 'invalid_path' || error.code === 'path_conflict') libraryForm.setFieldError('path', error.message)
  },
})

const submitUpdate = libraryForm.handleSubmit((values) => {
  updateMutation.mutate(values)
})

const deleteMutation = useMutation({
  mutationFn: () => deleteLibrary(libraryID.value),
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['libraries'] })
    await queryClient.invalidateQueries({ queryKey: ['home'] })
    await router.push({ name: 'libraries' })
  },
})

const errorMessage = computed(() => {
  const error = updateMutation.error.value ?? deleteMutation.error.value
  if (error instanceof ApiError && ['invalid_name', 'invalid_path', 'path_conflict'].includes(error.code)) return undefined
  if (error instanceof Error) return error.message
  return undefined
})

function confirmDelete(): void {
  if (!library.value) return
  const confirmed = window.confirm(`Supprimer la bibliothèque « ${library.value.name} » de Flex ? Les fichiers vidéo ne seront pas supprimés.`)
  if (confirmed) deleteMutation.mutate()
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('fr-FR', { dateStyle: 'long', timeStyle: 'short' }).format(new Date(value))
}
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div class="mx-auto max-w-4xl px-5 py-10 lg:px-10 lg:py-14">
      <RouterLink :to="{ name: 'library', params: { libraryId: libraryID } }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground"><ArrowLeft class="size-4" />Retour à la bibliothèque</RouterLink>
      <div v-if="librariesQuery.isPending.value" class="grid min-h-96 place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
      <template v-else-if="library">
        <div class="mt-6"><h1 class="text-3xl font-bold tracking-tight">Paramètres</h1><p class="mt-2 text-sm text-muted-foreground">Gérez la bibliothèque {{ library.name }}.</p></div>

        <Card class="mt-10 gap-0 rounded-2xl border-white/8 bg-card/60 py-0 shadow-none">
          <form novalidate @submit="submitUpdate">
            <CardHeader class="p-6 pb-0"><CardTitle>Informations générales</CardTitle></CardHeader>
            <CardContent class="p-6">
              <div class="grid gap-5 sm:grid-cols-2">
                <FormField v-slot="{ componentField }" name="name">
                  <FormItem><FormLabel>Nom</FormLabel><FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 shadow-none focus-visible:ring-primary/20" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="path">
                  <FormItem><FormLabel>Dossier</FormLabel><FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 font-mono shadow-none focus-visible:ring-primary/20" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
                </FormField>
              </div>
              <p v-if="errorMessage" class="mt-4 rounded-xl border border-red-400/15 bg-red-400/8 px-3 py-2.5 text-sm text-red-300">{{ errorMessage }}</p>
              <p v-if="successMessage" class="mt-4 text-sm text-emerald-400">{{ successMessage }}</p>
              <div class="mt-6 flex justify-end"><Button type="submit" :disabled="updateMutation.isPending.value"><LoaderCircle v-if="updateMutation.isPending.value" class="animate-spin" />Enregistrer</Button></div>
            </CardContent>
          </form>
        </Card>

        <Card class="mt-6 gap-0 rounded-2xl border-white/8 bg-card/60 py-0 shadow-none">
          <CardHeader class="p-6 pb-0"><CardTitle>Dernière analyse</CardTitle></CardHeader>
          <CardContent class="p-6">
            <p v-if="library.lastScanAt" class="text-sm text-muted-foreground">{{ formatDate(library.lastScanAt) }}</p>
            <p v-else class="text-sm text-muted-foreground">Cette bibliothèque n'a pas encore été analysée depuis l'ajout du suivi.</p>
            <dl v-if="library.lastScanAt" class="mt-5 grid grid-cols-3 gap-4 text-sm">
            <div><dt class="text-xs text-muted-foreground">Détectées</dt><dd class="mt-1 text-lg font-semibold">{{ library.lastScanDiscovered }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Indexées</dt><dd class="mt-1 text-lg font-semibold">{{ library.lastScanIndexed }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Ignorées</dt><dd class="mt-1 text-lg font-semibold">{{ library.lastScanSkipped }}</dd></div>
            </dl>
          </CardContent>
        </Card>

        <Card class="mt-6 gap-0 rounded-2xl border-red-400/15 bg-red-400/[0.025] py-0 shadow-none">
          <CardHeader class="p-6 pb-0"><CardTitle class="text-red-300">Zone dangereuse</CardTitle></CardHeader>
          <CardContent class="p-6">
            <p class="text-sm leading-6 text-muted-foreground">Retire cette bibliothèque et son index de Flex. Les fichiers présents dans {{ library.path }} ne seront jamais supprimés.</p>
            <Button class="mt-5" variant="secondary" :disabled="deleteMutation.isPending.value" @click="confirmDelete"><Trash2 />Supprimer la bibliothèque</Button>
          </CardContent>
        </Card>
      </template>
      <div v-else class="grid min-h-96 place-items-center text-muted-foreground">Bibliothèque introuvable.</div>
    </div>
  </section>
</template>
