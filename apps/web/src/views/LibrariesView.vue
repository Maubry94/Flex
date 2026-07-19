<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { Film, Folder, FolderPlus, LoaderCircle, X } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useForm } from 'vee-validate'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { ApiError, createLibrary, getLibraries } from '@/lib/api/libraries'
import { asForwardedProps } from '@/lib/utils'

const queryClient = useQueryClient()
const isDialogOpen = ref(false)

const libraryFormSchema = toTypedSchema(z.object({
  name: z.string().trim().min(1, 'Le nom est requis.').max(100, 'Le nom est trop long.'),
  path: z.string().trim().min(1, 'Le dossier est requis.').max(4_096, 'Le chemin est trop long.'),
}))
const libraryForm = useForm({
  validationSchema: libraryFormSchema,
  initialValues: { name: 'Mes vidéos', path: '/media' },
})

const librariesQuery = useQuery({
  queryKey: ['libraries'],
  queryFn: ({ signal }) => getLibraries(signal),
})

const createMutation = useMutation({
  mutationFn: createLibrary,
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['libraries'] })
    isDialogOpen.value = false
    libraryForm.resetForm({ values: { name: 'Mes vidéos', path: '/media' } })
  },
  onError: (error) => {
    if (!(error instanceof ApiError)) return
    if (error.code === 'invalid_name') libraryForm.setFieldError('name', error.message)
    if (error.code === 'invalid_path' || error.code === 'path_conflict') libraryForm.setFieldError('path', error.message)
  },
})

const errorMessage = computed(() => {
  const error = createMutation.error.value
  if (error instanceof ApiError && ['invalid_name', 'invalid_path', 'path_conflict'].includes(error.code)) return undefined
  if (error instanceof Error) return error.message
  return undefined
})

function openDialog(): void {
  createMutation.reset()
  libraryForm.resetForm({ values: { name: 'Mes vidéos', path: '/media' } })
  isDialogOpen.value = true
}

function closeDialog(): void {
  if (!createMutation.isPending.value) isDialogOpen.value = false
}

const submit = libraryForm.handleSubmit((values) => {
  createMutation.mutate(values)
})
</script>

<template>
  <section class="relative min-h-[calc(100dvh-4rem)] overflow-hidden">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(ellipse_at_top,rgba(124,58,237,0.10),transparent_68%)]" />
    <div class="relative mx-auto max-w-[1600px] px-5 py-10 lg:px-10 lg:py-14">
      <div class="flex items-end justify-between gap-6">
        <div>
          <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Bibliothèques</h1>
          <p class="mt-2 text-sm text-muted-foreground">Choisissez une bibliothèque pour parcourir ses vidéos.</p>
        </div>
        <Button v-if="librariesQuery.data.value?.length" @click="openDialog">
          <FolderPlus />
          Ajouter
        </Button>
      </div>

      <div v-if="librariesQuery.isPending.value" class="grid min-h-[520px] place-items-center">
        <LoaderCircle class="size-7 animate-spin text-primary" aria-label="Chargement" />
      </div>
      <div v-else-if="librariesQuery.isError.value" class="mt-10 grid min-h-[420px] place-items-center rounded-3xl border border-red-400/15 bg-red-400/[0.025] px-6 text-center">
        <div>
          <p class="font-semibold">Impossible de charger les bibliothèques</p>
          <Button class="mt-5" variant="secondary" @click="librariesQuery.refetch()">Réessayer</Button>
        </div>
      </div>
      <div v-else-if="librariesQuery.data.value?.length === 0" class="mt-10 grid min-h-[520px] place-items-center rounded-3xl border border-dashed border-white/12 bg-white/[0.018] px-6 py-16">
        <div class="max-w-md text-center">
          <div class="relative mx-auto grid size-20 place-items-center">
            <div class="absolute inset-0 rounded-3xl bg-primary/15 blur-xl" />
            <div class="relative grid size-20 place-items-center rounded-3xl border border-white/10 bg-card shadow-2xl shadow-black/30">
              <Film class="size-9 text-primary" />
            </div>
          </div>
          <h2 class="mt-7 text-xl font-semibold tracking-tight">Votre bibliothèque est vide</h2>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">Ajoutez le dossier contenant vos vidéos pour commencer.</p>
          <Button class="mt-7" size="lg" @click="openDialog">
            <FolderPlus />
            Ajouter une bibliothèque
          </Button>
        </div>
      </div>
      <div v-else class="mt-10 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <RouterLink v-for="library in librariesQuery.data.value" :key="library.id" :to="{ name: 'library', params: { libraryId: library.id } }" class="group block">
          <Card class="gap-0 rounded-2xl border-white/8 bg-card/70 py-0 shadow-none transition group-hover:-translate-y-0.5 group-hover:border-primary/35 group-hover:bg-card">
            <CardContent class="flex items-center gap-4 p-5">
              <span class="grid size-12 shrink-0 place-items-center rounded-2xl bg-primary/12 text-primary">
                <Folder class="size-6" />
              </span>
              <div class="min-w-0">
                <h2 class="truncate font-semibold">{{ library.name }}</h2>
              </div>
            </CardContent>
          </Card>
        </RouterLink>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="isDialogOpen" class="fixed inset-0 z-[100] grid place-items-center bg-black/70 p-4 backdrop-blur-sm" @mousedown.self="closeDialog">
        <section class="w-full max-w-md rounded-3xl border border-white/10 bg-zinc-950 p-6 shadow-2xl" role="dialog" aria-modal="true" aria-labelledby="add-library-title">
          <div class="flex items-start justify-between gap-4">
            <div>
              <h2 id="add-library-title" class="text-lg font-semibold">Ajouter une bibliothèque</h2>
              <p class="mt-1 text-sm text-muted-foreground">Indiquez le dossier tel qu'il est monté dans Flex.</p>
            </div>
            <button class="grid size-9 place-items-center rounded-full text-muted-foreground transition hover:bg-white/8 hover:text-foreground" aria-label="Fermer" @click="closeDialog"><X class="size-4" /></button>
          </div>
          <form class="mt-6 space-y-5" novalidate @submit="submit">
            <FormField v-slot="{ componentField }" name="name">
              <FormItem>
                <FormLabel>Nom</FormLabel>
                <FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 shadow-none focus-visible:ring-primary/20" autocomplete="off" v-bind="asForwardedProps(componentField)" /></FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
            <FormField v-slot="{ componentField }" name="path">
              <FormItem>
                <FormLabel>Dossier</FormLabel>
                <FormControl><Input class="h-11 rounded-xl border-white/10 bg-white/5 px-3.5 font-mono shadow-none focus-visible:ring-primary/20" autocomplete="off" v-bind="asForwardedProps(componentField)" /></FormControl>
                <FormDescription>Le dossier racine par défaut est <code class="rounded bg-white/6 px-1.5 py-0.5 text-foreground">/media</code>.</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
            <p v-if="errorMessage" class="rounded-xl border border-red-400/15 bg-red-400/8 px-3 py-2.5 text-sm text-red-300">{{ errorMessage }}</p>
            <div class="flex justify-end gap-3 pt-1">
              <Button type="button" variant="ghost" :disabled="createMutation.isPending.value" @click="closeDialog">Annuler</Button>
              <Button type="submit" :disabled="createMutation.isPending.value">
                <LoaderCircle v-if="createMutation.isPending.value" class="animate-spin" />
                Ajouter
              </Button>
            </div>
          </form>
        </section>
      </div>
    </Teleport>
  </section>
</template>
