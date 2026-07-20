<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { KeyRound, LoaderCircle } from '@lucide/vue'
import { useForm } from 'vee-validate'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { changePassword, getAuthStatus } from '@/lib/api/auth'
import { asForwardedProps } from '@/lib/utils'
import ProfileDetailsForm from '@/components/users/ProfileDetailsForm.vue'

const authQuery = useQuery({ queryKey: ['auth-status'], queryFn: ({ signal }) => getAuthStatus(signal) })
const form = useForm({
  validationSchema: toTypedSchema(z.object({
    currentPassword: z.string().min(1, 'Le mot de passe actuel est requis.'),
    newPassword: z.string().min(12, 'Utilisez au moins 12 caractères.').max(256),
    confirmation: z.string().min(1, 'Confirmez le nouveau mot de passe.'),
  }).refine((values) => values.newPassword === values.confirmation, { path: ['confirmation'], message: 'Les mots de passe ne correspondent pas.' })),
  initialValues: { currentPassword: '', newPassword: '', confirmation: '' },
})
const mutation = useMutation({
  mutationFn: (values: { currentPassword: string, newPassword: string }) => changePassword(values.currentPassword, values.newPassword),
  onSuccess: () => { form.resetForm(); toast.success('Mot de passe modifié') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier le mot de passe'),
})
const submit = form.handleSubmit((values) => { mutation.mutate({ currentPassword: values.currentPassword, newPassword: values.newPassword }) })
</script>

<template>
  <section class="min-h-[calc(100dvh-4rem)]">
    <div class="mx-auto max-w-3xl px-4 py-8 sm:px-5 sm:py-10 lg:py-14">
      <div><h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Mon compte</h1><p class="mt-2 text-sm text-muted-foreground">Connecté en tant que {{ authQuery.data.value?.user?.username }}.</p></div>
      <div class="mt-10 space-y-5">
      <ProfileDetailsForm :user="authQuery.data.value?.user" />
      <Card class="border-white/8 bg-card/60 shadow-none">
        <CardHeader><div class="flex items-center gap-3"><span class="grid size-10 place-items-center rounded-xl bg-primary/12 text-primary"><KeyRound class="size-5" /></span><div><CardTitle>Mot de passe</CardTitle><CardDescription>La modification déconnectera vos autres sessions.</CardDescription></div></div></CardHeader>
        <CardContent><form class="space-y-4" novalidate @submit="submit">
          <FormField v-slot="{ componentField }" name="currentPassword"><FormItem><FormLabel>Mot de passe actuel</FormLabel><FormControl><Input type="password" autocomplete="current-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <FormField v-slot="{ componentField }" name="newPassword"><FormItem><FormLabel>Nouveau mot de passe</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <FormField v-slot="{ componentField }" name="confirmation"><FormItem><FormLabel>Confirmer le nouveau mot de passe</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
          <div class="flex justify-end"><Button type="submit" :disabled="mutation.isPending.value"><LoaderCircle v-if="mutation.isPending.value" class="animate-spin" />Modifier le mot de passe</Button></div>
        </form></CardContent>
      </Card>
      </div>
    </div>
  </section>
</template>
