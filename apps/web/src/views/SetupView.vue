<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { Film, LoaderCircle } from '@lucide/vue'
import { useForm } from 'vee-validate'
import { ref } from 'vue'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { setupAdministrator } from '@/lib/api/auth'
import { asForwardedProps } from '@/lib/utils'

const emit = defineEmits<{ authenticated: [] }>()
const errorMessage = ref('')
const form = useForm({ validationSchema: toTypedSchema(z.object({
  username: z.string().trim().min(3, 'Utilisez au moins 3 caractères.').max(64).regex(/^[A-Za-z0-9._-]+$/, 'Utilisez uniquement des lettres, chiffres, points, tirets ou underscores.'),
  password: z.string().min(12, 'Utilisez au moins 12 caractères.').max(256),
  confirmation: z.string(),
}).refine((values) => values.password === values.confirmation, { message: 'Les mots de passe ne correspondent pas.', path: ['confirmation'] })) })

const submit = form.handleSubmit(async (values) => {
  errorMessage.value = ''
  try {
    await setupAdministrator({ username: values.username, password: values.password })
    emit('authenticated')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Impossible de configurer Flex'
  }
})
</script>

<template>
  <main class="grid min-h-dvh place-items-center px-4 py-10">
    <Card class="w-full max-w-md border-white/10 bg-card/75 shadow-2xl shadow-black/20">
      <CardHeader class="justify-items-center text-center">
        <span class="mb-3 grid size-12 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/25"><Film /></span>
        <CardTitle class="text-2xl">Configurer Flex</CardTitle>
        <CardDescription>Créez le compte administrateur de votre serveur.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" novalidate @submit="submit">
          <FormField v-slot="{ componentField }" name="username"><FormItem><FormLabel>Nom d’utilisateur</FormLabel><FormControl><Input autocapitalize="none" autocomplete="username" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <FormField v-slot="{ componentField }" name="password"><FormItem><FormLabel>Mot de passe</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <FormField v-slot="{ componentField }" name="confirmation"><FormItem><FormLabel>Confirmer le mot de passe</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <p v-if="errorMessage" class="rounded-xl border border-red-400/15 bg-red-400/8 px-3 py-2.5 text-sm text-red-300">{{ errorMessage }}</p>
          <Button class="w-full" type="submit" :disabled="form.isSubmitting.value"><LoaderCircle v-if="form.isSubmitting.value" class="animate-spin" />Créer l’administrateur</Button>
        </form>
      </CardContent>
    </Card>
  </main>
</template>
