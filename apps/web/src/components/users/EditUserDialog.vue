<script setup lang="ts">
import { useMutation } from '@tanstack/vue-query'
import { toTypedSchema } from '@vee-validate/zod'
import { LoaderCircle } from '@lucide/vue'
import { useForm } from 'vee-validate'
import { watch } from 'vue'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { AuthUser } from '@/lib/api/auth'
import { updateUser } from '@/lib/api/users'
import { asForwardedProps } from '@/lib/utils'

const props = defineProps<{ user: AuthUser | undefined, currentUser: boolean }>()
const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{ updated: [] }>()
const usernameSchema = z.string().trim().min(3, 'Utilisez au moins 3 caractères.').max(64).regex(/^[A-Za-z0-9._-]+$/, 'Utilisez uniquement des lettres, chiffres, points, tirets ou underscores.')
const form = useForm({ validationSchema: toTypedSchema(z.object({ username: usernameSchema, role: z.enum(['admin', 'user']) })) })

watch([open, () => props.user], ([isOpen, user]) => {
  if (isOpen && user) form.resetForm({ values: { username: user.username, role: user.role } })
})

const mutation = useMutation({
  mutationFn: (values: { username: string, role: 'admin' | 'user' }) => {
    if (!props.user) throw new Error('Aucun utilisateur sélectionné')
    return updateUser(props.user.id, { ...values, active: props.user.active })
  },
  onSuccess: () => { open.value = false; emit('updated'); toast.success('Utilisateur modifié') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de modifier l’utilisateur'),
})
const submit = form.handleSubmit((values) => { mutation.mutate(values) })
</script>

<template>
  <Dialog v-model:open="open"><DialogContent class="border-white/10 bg-background sm:max-w-md"><DialogHeader><DialogTitle>Modifier l’utilisateur</DialogTitle><DialogDescription>Modifiez son nom ou ses permissions.</DialogDescription></DialogHeader><form class="space-y-4" novalidate @submit="submit"><FormField v-slot="{ componentField }" name="username"><FormItem><FormLabel>Nom d’utilisateur</FormLabel><FormControl><Input v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField><FormField v-slot="{ componentField }" name="role"><FormItem><FormLabel>Rôle</FormLabel><Select v-bind="asForwardedProps(componentField)" :disabled="currentUser"><FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl><SelectContent><SelectItem value="user">Utilisateur</SelectItem><SelectItem value="admin">Administrateur</SelectItem></SelectContent></Select><FormMessage /></FormItem></FormField><DialogFooter><Button type="button" variant="ghost" @click="open = false">Annuler</Button><Button type="submit" :disabled="mutation.isPending.value"><LoaderCircle v-if="mutation.isPending.value" class="animate-spin" />Enregistrer</Button></DialogFooter></form></DialogContent></Dialog>
</template>
