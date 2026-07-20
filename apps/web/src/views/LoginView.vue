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
import { login } from '@/lib/api/auth'
import { asForwardedProps } from '@/lib/utils'

const emit = defineEmits<{ authenticated: [] }>()
const errorMessage = ref('')
const form = useForm({ validationSchema: toTypedSchema(z.object({ username: z.string().trim().min(1, 'Le nom d’utilisateur est requis.'), password: z.string().min(1, 'Le mot de passe est requis.') })) })
const submit = form.handleSubmit(async (values) => {
  errorMessage.value = ''
  try {
    await login(values.username, values.password)
    emit('authenticated')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Impossible de se connecter'
  }
})
</script>

<template>
  <main class="grid min-h-dvh place-items-center px-4 py-10">
    <Card class="w-full max-w-md border-white/10 bg-card/75 shadow-2xl shadow-black/20">
      <CardHeader class="justify-items-center text-center">
        <span class="mb-3 grid size-12 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/25"><Film /></span>
        <CardTitle class="text-2xl">Connexion à Flex</CardTitle>
        <CardDescription>Accédez à votre vidéothèque personnelle.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" novalidate @submit="submit">
          <FormField v-slot="{ componentField }" name="username"><FormItem><FormLabel>Nom d’utilisateur</FormLabel><FormControl><Input autofocus autocapitalize="none" autocomplete="username" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <FormField v-slot="{ componentField }" name="password"><FormItem><FormLabel>Mot de passe</FormLabel><FormControl><Input type="password" autocomplete="current-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <p v-if="errorMessage" class="rounded-xl border border-red-400/15 bg-red-400/8 px-3 py-2.5 text-sm text-red-300">{{ errorMessage }}</p>
          <Button class="w-full" type="submit" :disabled="form.isSubmitting.value"><LoaderCircle v-if="form.isSubmitting.value" class="animate-spin" />Se connecter</Button>
        </form>
      </CardContent>
    </Card>
  </main>
</template>
