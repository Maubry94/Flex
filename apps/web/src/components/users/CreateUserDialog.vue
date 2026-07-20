<script setup lang="ts">
import { useMutation } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { LoaderCircle } from '@lucide/vue'
import { useForm } from 'vee-validate'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { createUser } from '@/lib/api/users'
import { asForwardedProps } from '@/lib/utils'

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{ created: [] }>()

const usernameSchema = z.string().trim().min(3, 'Utilisez au moins 3 caractères.').max(64).regex(/^[A-Za-z0-9._-]+$/, 'Utilisez uniquement des lettres, chiffres, points, tirets ou underscores.')
const passwordSchema = z.string().min(12, 'Utilisez au moins 12 caractères.').max(256)
const form = useForm({
  validationSchema: toTypedSchema(z.object({ username: usernameSchema, password: passwordSchema, role: z.enum(['admin', 'user']) })),
  initialValues: { username: '', password: '', role: 'user' as const },
})

const mutation = useMutation({
  mutationFn: createUser,
  onSuccess: () => {
    open.value = false
    form.resetForm()
    emit('created')
    toast.success('Utilisateur créé')
  },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de créer l’utilisateur'),
})

const submit = form.handleSubmit((values) => { mutation.mutate(values) })
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="border-white/10 bg-background sm:max-w-md">
      <DialogHeader><DialogTitle>Ajouter un utilisateur</DialogTitle><DialogDescription>Ce compte pourra se connecter à ce serveur Flex.</DialogDescription></DialogHeader>
      <form class="space-y-4" novalidate @submit="submit">
        <FormField v-slot="{ componentField }" name="username"><FormItem><FormLabel>Nom d’utilisateur</FormLabel><FormControl><Input autocomplete="off" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
        <FormField v-slot="{ componentField }" name="password"><FormItem><FormLabel>Mot de passe temporaire</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField>
        <FormField v-slot="{ componentField }" name="role"><FormItem><FormLabel>Rôle</FormLabel><Select v-bind="asForwardedProps(componentField)"><FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl><SelectContent><SelectItem value="user">Utilisateur</SelectItem><SelectItem value="admin">Administrateur</SelectItem></SelectContent></Select><FormMessage /></FormItem></FormField>
        <DialogFooter><Button type="button" variant="ghost" @click="open = false">Annuler</Button><Button type="submit" :disabled="mutation.isPending.value"><LoaderCircle v-if="mutation.isPending.value" class="animate-spin" />Créer</Button></DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
