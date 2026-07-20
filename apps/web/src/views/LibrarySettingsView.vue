<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { ArrowLeft, CircleAlert, LoaderCircle, RefreshCw, Trash2 } from '@lucide/vue'
import { computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { ApiError, deleteLibrary, getLibraries, updateLibrary } from '@/lib/api/libraries'
import { getScanStatus, scanLibrary } from '@/lib/api/media'
import { asForwardedProps } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const libraryID = computed(() => String(route.params.libraryId))

const libraryFormSchema = toTypedSchema(z.object({
  name: z.string().trim().min(1, 'Le nom est requis.').max(100, 'Le nom est trop long.'),
  path: z.string().trim().min(1, 'Le dossier est requis.').max(4_096, 'Le chemin est trop long.'),
}))
const libraryForm = useForm({ validationSchema: libraryFormSchema })

const librariesQuery = useQuery({ queryKey: ['libraries'], queryFn: ({ signal }) => getLibraries(signal) })
const scanStatusQuery = useQuery({
  queryKey: computed(() => ['scan-status', libraryID.value]),
  queryFn: ({ signal }) => getScanStatus(libraryID.value, signal),
  refetchInterval: 2_000,
})
const library = computed(() => librariesQuery.data.value?.find((item) => item.id === libraryID.value))
const latestScan = computed(() => scanStatusQuery.data.value?.result)
const isScanning = computed(() => ['pending', 'scanning'].includes(scanStatusQuery.data.value?.state ?? ''))

watch(library, (value) => {
  if (!value) return
  libraryForm.resetForm({ values: { name: value.name, path: value.path } })
}, { immediate: true })

const updateMutation = useMutation({
  mutationFn: (values: { name: string, path: string }) => updateLibrary(libraryID.value, values),
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['libraries'] })
    toast.success('Modifications enregistrées')
  },
  onError: (error) => {
    if (!(error instanceof ApiError)) {
      toast.error(error instanceof Error ? error.message : 'Impossible de modifier la bibliothèque')
      return
    }
    if (error.code === 'invalid_name') libraryForm.setFieldError('name', error.message)
    if (error.code === 'invalid_path' || error.code === 'path_conflict') libraryForm.setFieldError('path', error.message)
    if (!['invalid_name', 'invalid_path', 'path_conflict'].includes(error.code)) toast.error(error.message)
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
    toast.success('Bibliothèque supprimée')
    await router.push({ name: 'libraries' })
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de supprimer la bibliothèque'),
})

const scanMutation = useMutation({
  mutationFn: () => scanLibrary(libraryID.value),
  onSuccess: async (result) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['libraries'] }),
      queryClient.invalidateQueries({ queryKey: ['scan-status', libraryID.value] }),
      queryClient.invalidateQueries({ queryKey: ['media', libraryID.value] }),
    ])
    toast.success(result.issues.length ? 'Analyse terminée, certains fichiers restent ignorés' : 'Analyse terminée sans erreur')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'L’analyse a échoué'),
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
    <div class="mx-auto max-w-4xl px-4 py-8 sm:px-5 sm:py-10 lg:px-10 lg:py-14">
      <RouterLink :to="{ name: 'library', params: { libraryId: libraryID } }" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground"><ArrowLeft class="size-4" />Retour à la bibliothèque</RouterLink>
      <div v-if="librariesQuery.isPending.value" class="grid min-h-96 place-items-center"><LoaderCircle class="size-7 animate-spin text-primary" /></div>
      <template v-else-if="library">
        <div class="mt-6"><h1 class="text-3xl font-bold tracking-tight">Paramètres</h1><p class="mt-2 text-sm text-muted-foreground">Gérez la bibliothèque {{ library.name }}.</p></div>

        <Card class="mt-10 gap-0 rounded-2xl border-white/8 bg-card/60 py-0 shadow-none">
          <form novalidate @submit="submitUpdate">
            <CardHeader class="p-5 pb-0 sm:p-6 sm:pb-0"><CardTitle>Informations générales</CardTitle></CardHeader>
            <CardContent class="p-5 sm:p-6">
              <div class="grid gap-5 sm:grid-cols-2">
                <FormField v-slot="{ componentField }" name="name">
                  <FormItem><FormLabel>Nom</FormLabel><FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 shadow-none focus-visible:ring-primary/20" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="path">
                  <FormItem><FormLabel>Dossier</FormLabel><FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 font-mono shadow-none focus-visible:ring-primary/20" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem>
                </FormField>
              </div>
              <p v-if="errorMessage" class="mt-4 rounded-xl border border-red-400/15 bg-red-400/8 px-3 py-2.5 text-sm text-red-300">{{ errorMessage }}</p>
              <div class="mt-6 flex justify-stretch sm:justify-end"><Button class="w-full sm:w-auto" type="submit" :disabled="updateMutation.isPending.value"><LoaderCircle v-if="updateMutation.isPending.value" class="animate-spin" />Enregistrer</Button></div>
            </CardContent>
          </form>
        </Card>

        <Card class="mt-6 gap-0 rounded-2xl border-white/8 bg-card/60 py-0 shadow-none">
          <CardHeader class="p-5 pb-0 sm:p-6 sm:pb-0"><CardTitle>Dernière analyse</CardTitle></CardHeader>
          <CardContent class="p-5 sm:p-6">
            <p v-if="library.lastScanAt" class="text-sm text-muted-foreground">{{ formatDate(library.lastScanAt) }}</p>
            <p v-else class="text-sm text-muted-foreground">Cette bibliothèque n'a pas encore été analysée depuis l'ajout du suivi.</p>
            <dl v-if="library.lastScanAt" class="mt-5 grid grid-cols-2 gap-4 text-sm sm:grid-cols-5">
            <div><dt class="text-xs text-muted-foreground">Détectées</dt><dd class="mt-1 text-lg font-semibold">{{ latestScan?.discovered ?? library.lastScanDiscovered }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Actualisées</dt><dd class="mt-1 text-lg font-semibold">{{ latestScan?.indexed ?? library.lastScanIndexed }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Inchangées</dt><dd class="mt-1 text-lg font-semibold">{{ latestScan?.unchanged ?? library.lastScanUnchanged }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Supprimées</dt><dd class="mt-1 text-lg font-semibold">{{ latestScan?.removed ?? 0 }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">Ignorées</dt><dd class="mt-1 text-lg font-semibold">{{ latestScan?.skipped ?? library.lastScanSkipped }}</dd></div>
            </dl>
          </CardContent>
        </Card>

        <Card v-if="latestScan?.issues.length" class="mt-6 gap-0 rounded-2xl border-amber-400/15 bg-amber-400/2.5 py-0 shadow-none">
          <CardHeader class="p-5 pb-0 sm:p-6 sm:pb-0"><CardTitle class="text-amber-200">Fichiers ignorés</CardTitle></CardHeader>
          <CardContent class="p-5 sm:p-6">
            <p class="text-sm text-muted-foreground">Ces fichiers n’ont pas pu être ajoutés pendant la dernière analyse.</p>
            <ul class="mt-4 divide-y divide-white/8 rounded-xl border border-white/8">
              <li v-for="issue in latestScan.issues" :key="`${issue.filename}-${issue.reason}`" class="px-4 py-3">
                <p class="break-all text-sm font-medium">{{ issue.filename }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ issue.reason }}</p>
              </li>
            </ul>
            <Button class="mt-5" variant="secondary" :disabled="isScanning || scanMutation.isPending.value" @click="scanMutation.mutate()"><RefreshCw :class="(isScanning || scanMutation.isPending.value) && 'animate-spin'" />Relancer l’analyse</Button>
          </CardContent>
        </Card>

        <Card class="mt-6 gap-0 rounded-2xl border-red-400/15 bg-red-400/2.5 py-0 shadow-none">
          <CardHeader class="p-5 pb-0 sm:p-6 sm:pb-0"><CardTitle class="text-red-300">Zone dangereuse</CardTitle></CardHeader>
          <CardContent class="p-5 sm:p-6">
            <p class="break-words text-sm leading-6 text-muted-foreground">Retire cette bibliothèque et son index de Flex. Les fichiers présents dans {{ library.path }} ne seront jamais supprimés.</p>
            <Button class="mt-5 w-full sm:w-auto" variant="secondary" :disabled="deleteMutation.isPending.value" @click="confirmDelete"><Trash2 />Supprimer la bibliothèque</Button>
          </CardContent>
        </Card>
      </template>
      <Empty v-else class="mt-10 min-h-96 border border-white/10">
        <EmptyHeader><EmptyMedia variant="icon"><CircleAlert /></EmptyMedia><EmptyTitle>Bibliothèque introuvable</EmptyTitle><EmptyDescription>Cette bibliothèque n’existe plus ou n’est plus accessible.</EmptyDescription></EmptyHeader>
      </Empty>
    </div>
  </section>
</template>
