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
import type { AuthUser } from '@/lib/api/auth'
import { resetUserPassword } from '@/lib/api/users'
import { asForwardedProps } from '@/lib/utils'

const props = defineProps<{ user: AuthUser | undefined }>()
const open = defineModel<boolean>('open', { required: true })
const passwordSchema = z.string().min(12, 'Utilisez au moins 12 caractères.').max(256)
const form = useForm({ validationSchema: toTypedSchema(z.object({ password: passwordSchema })), initialValues: { password: '' } })
watch(open, (isOpen) => { if (isOpen) form.resetForm() })
const mutation = useMutation({
  mutationFn: (password: string) => {
    if (!props.user) throw new Error('Aucun utilisateur sélectionné')
    return resetUserPassword(props.user.id, password)
  },
  onSuccess: () => { open.value = false; toast.success('Mot de passe remplacé et sessions révoquées') },
  onError: (error) => toast.error(error instanceof Error ? error.message : 'Impossible de remplacer le mot de passe'),
})
const submit = form.handleSubmit((values) => { mutation.mutate(values.password) })
</script>

<template>
  <Dialog v-model:open="open"><DialogContent class="border-white/10 bg-background sm:max-w-md"><DialogHeader><DialogTitle>Réinitialiser le mot de passe</DialogTitle><DialogDescription>Les sessions ouvertes de {{ user?.username }} seront immédiatement révoquées.</DialogDescription></DialogHeader><form class="space-y-4" novalidate @submit="submit"><FormField v-slot="{ componentField }" name="password"><FormItem><FormLabel>Nouveau mot de passe</FormLabel><FormControl><Input type="password" autocomplete="new-password" v-bind="asForwardedProps(componentField)" /></FormControl><FormMessage /></FormItem></FormField><DialogFooter><Button type="button" variant="ghost" @click="open = false">Annuler</Button><Button type="submit" :disabled="mutation.isPending.value"><LoaderCircle v-if="mutation.isPending.value" class="animate-spin" />Remplacer</Button></DialogFooter></form></DialogContent></Dialog>
</template>
