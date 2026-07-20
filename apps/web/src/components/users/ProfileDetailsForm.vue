<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { LoaderCircle, UserRound } from '@lucide/vue'
import { useForm } from 'vee-validate'
import { watch } from 'vue'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import type { AuthStatus, AuthUser } from '@/lib/api/auth'
import { updateProfile } from '@/lib/api/auth'
import { asForwardedProps } from '@/lib/utils'

const props = defineProps<{ user: AuthUser | undefined }>()
const queryClient = useQueryClient()
const form = useForm({
  validationSchema: toTypedSchema(z.object({ username: z.string().trim().min(3, 'Utilisez au moins 3 caractères.').max(64).regex(/^[A-Za-z0-9._-]+$/u, 'Utilisez uniquement des lettres, chiffres, points, tirets ou underscores.') })),
  initialValues: { username: props.user?.username ?? '' },
})
watch(() => props.user?.username, (username) => { if (username) form.resetForm({ values: { username } }) }, { immediate: true })
const mutation = useMutation({
  mutationFn: (username: string) => updateProfile(username),
  onSuccess: (user) => {
    queryClient.setQueryData<AuthStatus>(['auth-status'], (status) => status ? { ...status, user } : status)
    void queryClient.invalidateQueries({ queryKey: ['users'] })
    form.resetForm({ values: { username: user.username } })
    toast.success('Profil modifié')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier le profil'),
})
const submit = form.handleSubmit((values) => { mutation.mutate(values.username) })
</script>

<template>
  <Card class="border-white/8 bg-card/60 shadow-none">
    <CardHeader><div class="flex items-center gap-3"><span class="grid size-10 place-items-center rounded-xl bg-primary/12 text-primary"><UserRound class="size-5" /></span><div><CardTitle>Profil</CardTitle><CardDescription>Ce nom est utilisé pour vous connecter et vous identifier dans Flex.</CardDescription></div></div></CardHeader>
    <CardContent><form class="space-y-4" novalidate @submit="submit"><FormField v-slot="{ componentField }" name="username"><FormItem><FormLabel>Nom d’utilisateur</FormLabel><FormControl><Input autocomplete="username" autocapitalize="none" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField><div class="flex justify-end"><Button type="submit" :disabled="mutation.isPending.value || !form.meta.value.dirty"><LoaderCircle v-if="mutation.isPending.value" class="animate-spin" />Enregistrer</Button></div></form></CardContent>
  </Card>
</template>
